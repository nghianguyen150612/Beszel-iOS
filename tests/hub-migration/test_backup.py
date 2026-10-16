"""Focused tests for P013 cold full-state backup and restore helpers."""

from __future__ import annotations

import pathlib
import sqlite3
import subprocess
import tempfile
import unittest

from backup import (
    BackupError,
    build_checkpoint_bundle,
    create_cold_backup,
    restore_and_compare,
    simulate_interrupted_archive,
    verify_backup,
)


HUB_PLIST = "synthetic Hub service definition\n"
AGENT_PLIST = "synthetic Agent service definition\n"


class ColdBackupTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory(prefix="p013-backup-test-")
        self.root = pathlib.Path(self.temporary.name)
        self.data = self.root / "source-data"
        self.data.mkdir()
        connection = sqlite3.connect(self.data / "data.db")
        connection.execute("CREATE TABLE synthetic_records(id TEXT PRIMARY KEY, value TEXT)")
        connection.execute("INSERT INTO synthetic_records VALUES('synthetic-id', 'synthetic-value')")
        connection.commit()
        connection.close()
        (self.data / "data.db-wal").write_bytes(b"committed synthetic WAL bytes")
        (self.data / "data.db-shm").write_bytes(b"synthetic SHM metadata")
        (self.data / "auxiliary.db").write_bytes(b"synthetic auxiliary database")
        (self.data / "config.yml").write_text("systems: []\n", encoding="utf-8")
        (self.data / "config.yml").chmod(0o600)
        (self.data / "id_ed25519").write_text("SYNTHETIC PRIVATE KEY MATERIAL ONLY\n", encoding="utf-8")
        (self.data / "id_ed25519").chmod(0o600)
        self.hub_binary = self.root / "old-hub"
        self.agent_binary = self.root / "old-agent"
        self.hub_binary.write_bytes(b"synthetic v0.19 Hub executable")
        self.agent_binary.write_bytes(b"synthetic v0.19 Agent executable")
        self.hub_binary.chmod(0o755)
        self.agent_binary.chmod(0o755)
        self.bundle = self.root / "checkpoint"
        self.output = self.root / "backup"

    def tearDown(self) -> None:
        self.temporary.cleanup()

    def make_bundle(self) -> pathlib.Path:
        build_checkpoint_bundle(
            source_data_dir=self.data,
            old_hub_binary=self.hub_binary,
            old_agent_binary=self.agent_binary,
            bundle_root=self.bundle,
            hub_release="v0.19.0-ios.2",
            agent_release="v0.19.0-ios.2",
            hub_service_plist=HUB_PLIST,
            agent_service_plist=AGENT_PLIST,
        )
        return self.bundle

    def make_backup(self) -> dict[str, object]:
        self.make_bundle()
        return create_cold_backup(
            bundle_root=self.bundle,
            output_dir=self.output,
            hub_pid=99999999,
            hub_exit_confirmed=True,
            agent_writer_pids=[],
            minimum_free_bytes=0,
        )

    def test_archive_contains_full_data_and_checkpoint_state_then_restores(self) -> None:
        result = self.make_backup()
        self.assertTrue(result["verified"])
        metadata = {item["path"]: item for item in result["file_metadata"]}
        self.assertEqual(metadata["var/lib/beszel-hub/config.yml"]["mode"], "0o0600")
        self.assertEqual(metadata["var/lib/beszel-hub/id_ed25519"]["mode"], "0o0600")
        self.assertEqual(metadata["usr/local/bin/beszel-hub"]["mode"], "0o0755")
        self.assertIn("uid", metadata["var/lib/beszel-hub/data.db"])
        self.assertIn("gid", metadata["var/lib/beszel-hub/data.db"])
        inventory = (self.output / "checkpoint-inventory.json").read_text(encoding="utf-8")
        for expected in (
            "data.db", "data.db-wal", "data.db-shm", "auxiliary.db", "id_ed25519",
            "beszel-hub", "beszel-agent", "dev.beszel.hub.plist", "dev.beszel.agent.plist", "install-state",
        ):
            self.assertIn(expected, inventory)
        restored = self.root / "restore"
        restore_result = restore_and_compare(self.output, restored)
        self.assertTrue(restore_result["restore_inventory_matches"])
        self.assertEqual(
            (restored / "checkpoint/var/lib/beszel-hub/config.yml").stat().st_mode & 0o777,
            0o600,
        )
        self.assertTrue((restored / "checkpoint/var/lib/beszel-hub/data.db-wal").is_file())

    def test_requires_confirmed_stopped_hub_and_agent_writers(self) -> None:
        self.make_bundle()
        with self.assertRaisesRegex(BackupError, "Hub writer"):
            create_cold_backup(
                bundle_root=self.bundle,
                output_dir=self.output,
                hub_pid=99999999,
                hub_exit_confirmed=False,
                agent_writer_pids=[],
                minimum_free_bytes=0,
            )

    def test_refuses_live_agent_writer_process(self) -> None:
        self.make_bundle()
        writer = subprocess.Popen(["sleep", "10"])
        try:
            with self.assertRaisesRegex(BackupError, "Agent writer processes remain active"):
                create_cold_backup(
                    bundle_root=self.bundle,
                    output_dir=self.output,
                    hub_pid=99999999,
                    hub_exit_confirmed=True,
                    agent_writer_pids=[writer.pid],
                    minimum_free_bytes=0,
                )
        finally:
            writer.terminate()
            writer.wait(timeout=5)

    def test_missing_required_key_fails_closed(self) -> None:
        self.make_bundle()
        (self.bundle / "var/lib/beszel-hub/id_ed25519").unlink()
        with self.assertRaisesRegex(BackupError, "required checkpoint file missing"):
            create_cold_backup(
                bundle_root=self.bundle,
                output_dir=self.output,
                hub_pid=99999999,
                hub_exit_confirmed=True,
                agent_writer_pids=[],
                minimum_free_bytes=0,
            )
        self.assertFalse(self.output.exists())

    def test_insufficient_space_does_not_publish_backup(self) -> None:
        self.make_bundle()
        with self.assertRaisesRegex(BackupError, "insufficient free space"):
            create_cold_backup(
                bundle_root=self.bundle,
                output_dir=self.output,
                hub_pid=99999999,
                hub_exit_confirmed=True,
                agent_writer_pids=[],
                minimum_free_bytes=10**30,
            )
        self.assertFalse(self.output.exists())

    def test_invalid_checksum_is_rejected(self) -> None:
        self.make_backup()
        with (self.output / "hub-state.tar.gz").open("ab") as archive:
            archive.write(b"tampered")
        with self.assertRaisesRegex(BackupError, "checksum mismatch"):
            verify_backup(self.output)

    def test_interrupted_real_tar_process_leaves_only_invalid_partial(self) -> None:
        self.make_bundle()
        # Keep tar busy after it emits the first archive block so the test
        # deterministically kills an in-progress archiver rather than racing a
        # tiny completed archive.
        large_synthetic = self.bundle / "var/lib/beszel-hub/synthetic-load.bin"
        large_synthetic.write_bytes(bytes(range(256)) * 65536)
        partial = self.root / "interrupted.tar"
        result = simulate_interrupted_archive(self.bundle, partial)
        self.assertFalse(result["final_archive_published"])
        self.assertGreater(result["partial_bytes"], 0)
        self.assertNotEqual(result["process_exit_code"], 0)
        self.assertFalse((self.root / "hub-state.tar.gz").exists())


if __name__ == "__main__":
    unittest.main()
