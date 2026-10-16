"""Cold, full-state checkpoint helpers for the disposable P013 harness.

This module only writes beneath caller-selected disposable paths. It refuses
the production Hub directory and requires the caller to prove that all known
writer processes have exited before it archives a checkpoint.
"""

from __future__ import annotations

import hashlib
import json
import os
import pathlib
import shutil
import signal
import subprocess
import tarfile
import tempfile
from typing import Any, Iterable


PROTECTED_DATA_DIR = pathlib.Path("/var/lib/beszel-hub")
REQUIRED_PATHS = (
    "var/lib/beszel-hub/data.db",
    "var/lib/beszel-hub/config.yml",
    "var/lib/beszel-hub/id_ed25519",
    "usr/local/bin/beszel-hub",
    "usr/local/bin/beszel-agent",
    "Library/LaunchDaemons/dev.beszel.hub.plist",
    "Library/LaunchDaemons/dev.beszel.agent.plist",
    "var/lib/beszel-ios/install-state",
)


class BackupError(RuntimeError):
    """The cold checkpoint was refused or did not verify."""


def _assert_outside_production(path: pathlib.Path) -> None:
    resolved = path.resolve()
    if resolved == PROTECTED_DATA_DIR or PROTECTED_DATA_DIR in resolved.parents:
        raise BackupError("refusing to read or write under /var/lib/beszel-hub")


def sha256_file(path: pathlib.Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as source:
        for block in iter(lambda: source.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def _path_metadata(path: pathlib.Path, relative: str) -> dict[str, Any]:
    info = path.lstat()
    entry: dict[str, Any] = {
        "path": relative,
        "kind": "symlink" if path.is_symlink() else "directory" if path.is_dir() else "file",
        "mode": info.st_mode & 0o7777,
        "uid": info.st_uid,
        "gid": info.st_gid,
        "size": info.st_size,
        "mtime_seconds": int(info.st_mtime),
    }
    if path.is_symlink():
        entry["target"] = os.readlink(path)
    elif path.is_file():
        entry["sha256"] = sha256_file(path)
    return entry


def inventory_tree(root: pathlib.Path) -> list[dict[str, Any]]:
    """Return a content-free metadata inventory with hashes for regular files."""
    root = root.resolve()
    _assert_outside_production(root)
    if not root.is_dir():
        raise BackupError(f"checkpoint directory missing: {root}")
    entries: list[dict[str, Any]] = []
    for directory, dirs, files in os.walk(root, topdown=True, followlinks=False):
        dirs.sort()
        files.sort()
        base = pathlib.Path(directory)
        for name in list(dirs):
            path = base / name
            relative = path.relative_to(root).as_posix()
            entries.append(_path_metadata(path, relative))
            if path.is_symlink():
                dirs.remove(name)
        for name in files:
            path = base / name
            entries.append(_path_metadata(path, path.relative_to(root).as_posix()))
    return sorted(entries, key=lambda entry: entry["path"])


def _required_path(root: pathlib.Path, relative: str) -> pathlib.Path:
    candidate = root / pathlib.PurePosixPath(relative)
    if not candidate.exists() or candidate.is_dir():
        raise BackupError(f"required checkpoint file missing: {relative}")
    return candidate


def validate_required_files(root: pathlib.Path) -> None:
    _assert_outside_production(root)
    for relative in REQUIRED_PATHS:
        _required_path(root, relative)


def process_alive(pid: int) -> bool:
    if pid <= 0:
        return False
    try:
        os.kill(pid, 0)
    except ProcessLookupError:
        return False
    except PermissionError:
        return True
    return True


def build_checkpoint_bundle(
    *,
    source_data_dir: pathlib.Path,
    old_hub_binary: pathlib.Path,
    old_agent_binary: pathlib.Path,
    bundle_root: pathlib.Path,
    hub_release: str,
    agent_release: str,
    hub_service_plist: str,
    agent_service_plist: str,
) -> dict[str, Any]:
    """Build a synthetic 0.19 restore set, preserving the complete data tree."""
    for path in (source_data_dir, old_hub_binary, old_agent_binary, bundle_root):
        _assert_outside_production(path)
    if not source_data_dir.is_dir() or not old_hub_binary.is_file() or not old_agent_binary.is_file():
        raise BackupError("source data and both old component binaries are required")
    if bundle_root.exists():
        raise BackupError(f"checkpoint bundle destination already exists: {bundle_root}")

    data_destination = bundle_root / "var/lib/beszel-hub"
    data_destination.parent.mkdir(parents=True)
    shutil.copytree(source_data_dir, data_destination, symlinks=True)
    # Include the Agent state directory as an empty synthetic tree because the
    # Agent writer is deliberately not started in this Hub migration fixture.
    (bundle_root / "var/lib/beszel-agent").mkdir(parents=True, mode=0o700)
    binaries = bundle_root / "usr/local/bin"
    binaries.mkdir(parents=True)
    shutil.copy2(old_hub_binary, binaries / "beszel-hub")
    shutil.copy2(old_agent_binary, binaries / "beszel-agent")
    launchd = bundle_root / "Library/LaunchDaemons"
    launchd.mkdir(parents=True)
    (launchd / "dev.beszel.hub.plist").write_text(hub_service_plist, encoding="utf-8")
    (launchd / "dev.beszel.agent.plist").write_text(agent_service_plist, encoding="utf-8")
    for path in launchd.iterdir():
        path.chmod(0o644)
    install_state = bundle_root / "var/lib/beszel-ios/install-state"
    install_state.parent.mkdir(parents=True)
    install_state.write_text(
        "STATE_VERSION=1\n"
        f"AGENT_RELEASE={agent_release}\n"
        f"AGENT_ASSET_SHA256={sha256_file(binaries / 'beszel-agent')}\n"
        f"HUB_RELEASE={hub_release}\n"
        f"HUB_ASSET_SHA256={sha256_file(binaries / 'beszel-hub')}\n",
        encoding="utf-8",
    )
    install_state.chmod(0o644)
    return {
        "root": str(bundle_root),
        "data_dir": "var/lib/beszel-hub",
        "sidecars": {
            name: (data_destination / name).is_file()
            for name in ("data.db-wal", "data.db-shm", "auxiliary.db")
        },
        "hub_release": hub_release,
        "agent_release": agent_release,
        "files": inventory_tree(bundle_root),
    }


def _archive_names(archive: pathlib.Path) -> set[str]:
    try:
        with tarfile.open(archive, "r:gz") as source:
            names: set[str] = set()
            for member in source.getmembers():
                name = pathlib.PurePosixPath(member.name)
                if name.is_absolute() or ".." in name.parts:
                    raise BackupError(f"unsafe path in checkpoint archive: {member.name!r}")
                normalized = pathlib.PurePosixPath(*[part for part in name.parts if part not in ("", ".")]).as_posix()
                if not normalized:
                    continue
                if normalized != "checkpoint" and not normalized.startswith("checkpoint/"):
                    raise BackupError(f"unexpected top-level archive entry: {member.name!r}")
                names.add(normalized)
            return names
    except (tarfile.TarError, OSError) as exc:
        raise BackupError(f"checkpoint archive is unreadable: {exc}") from exc


def _expected_archive_names(inventory: Iterable[dict[str, Any]]) -> set[str]:
    return {"checkpoint", *[f"checkpoint/{entry['path']}" for entry in inventory]}


def verify_backup(backup_dir: pathlib.Path) -> dict[str, Any]:
    """Verify checksums, archive inventory, and required recovery files."""
    _assert_outside_production(backup_dir)
    archive = backup_dir / "hub-state.tar.gz"
    inventory_path = backup_dir / "checkpoint-inventory.json"
    sums_path = backup_dir / "SHA256SUMS"
    for path in (archive, inventory_path, sums_path):
        if not path.is_file():
            raise BackupError(f"backup verification input missing: {path.name}")
    expected_sums: dict[str, str] = {}
    for line in sums_path.read_text(encoding="ascii").splitlines():
        fields = line.split(maxsplit=1)
        if len(fields) != 2 or len(fields[0]) != 64:
            raise BackupError("invalid SHA256SUMS format")
        expected_sums[fields[1].strip()] = fields[0].lower()
    if set(expected_sums) != {"hub-state.tar.gz", "checkpoint-inventory.json"}:
        raise BackupError("SHA256SUMS does not cover the archive and inventory exactly")
    if sha256_file(archive) != expected_sums["hub-state.tar.gz"]:
        raise BackupError("checkpoint archive checksum mismatch")
    if sha256_file(inventory_path) != expected_sums["checkpoint-inventory.json"]:
        raise BackupError("checkpoint inventory checksum mismatch")
    try:
        inventory_doc = json.loads(inventory_path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise BackupError(f"checkpoint inventory is invalid: {exc}") from exc
    inventory = inventory_doc.get("entries")
    if not isinstance(inventory, list):
        raise BackupError("checkpoint inventory entries missing")
    root_names = {entry.get("path") for entry in inventory if isinstance(entry, dict)}
    missing = sorted(set(REQUIRED_PATHS) - root_names)
    if missing:
        raise BackupError(f"checkpoint inventory lacks required files: {', '.join(missing)}")
    if _archive_names(archive) != _expected_archive_names(inventory):
        raise BackupError("checkpoint archive entries differ from the full-state inventory")
    return {
        "archive_sha256": expected_sums["hub-state.tar.gz"],
        "inventory_sha256": expected_sums["checkpoint-inventory.json"],
        "entry_count": len(inventory),
        "sidecars": inventory_doc.get("sidecars", {}),
        "verified": True,
    }


def create_cold_backup(
    *,
    bundle_root: pathlib.Path,
    output_dir: pathlib.Path,
    hub_pid: int,
    hub_exit_confirmed: bool,
    agent_writer_pids: Iterable[int],
    minimum_free_bytes: int,
) -> dict[str, Any]:
    """Create and verify a cold full-state archive, then publish atomically."""
    _assert_outside_production(bundle_root)
    _assert_outside_production(output_dir)
    validate_required_files(bundle_root)
    if not hub_exit_confirmed or process_alive(hub_pid):
        raise BackupError("Hub writer has not been confirmed stopped")
    live_agents = [pid for pid in agent_writer_pids if process_alive(int(pid))]
    if live_agents:
        raise BackupError(f"synthetic Agent writer processes remain active: {live_agents}")
    if output_dir.exists():
        raise BackupError(f"refusing to overwrite an existing backup: {output_dir}")
    parent = output_dir.parent
    parent.mkdir(parents=True, exist_ok=True)
    available_bytes = shutil.disk_usage(parent).free
    if minimum_free_bytes < 0 or available_bytes < minimum_free_bytes:
        raise BackupError(
            f"insufficient free space: available={available_bytes} required={minimum_free_bytes}"
        )

    entries = inventory_tree(bundle_root)
    inventory_doc = {
        "format": "p013-cold-checkpoint-v1",
        "created_by": "tests/hub-migration/backup.py",
        "hub_pid": hub_pid,
        "hub_exit_confirmed": True,
        "agent_writer_pids": list(agent_writer_pids),
        "available_bytes_before": available_bytes,
        "required_free_bytes": minimum_free_bytes,
        "sidecars": {
            name: any(entry["path"] == f"var/lib/beszel-hub/{name}" for entry in entries)
            for name in ("data.db-wal", "data.db-shm", "auxiliary.db")
        },
        "entries": entries,
    }
    staging = pathlib.Path(tempfile.mkdtemp(prefix=".p013-cold-backup-", dir=parent))
    try:
        partial = staging / "hub-state.tar.gz.partial"
        completed = subprocess.run(
            ["tar", "--numeric-owner", "--format=posix", "-czpf", str(partial), "-C", str(bundle_root.parent), bundle_root.name],
            check=False,
            capture_output=True,
            text=True,
        )
        if completed.returncode != 0:
            raise BackupError(f"archive creation failed ({completed.returncode}): {completed.stderr[-2000:]}")
        archive_names = _archive_names(partial)
        if archive_names != _expected_archive_names(entries):
            raise BackupError("created archive does not contain the full checkpoint inventory")
        archive = staging / "hub-state.tar.gz"
        partial.replace(archive)
        inventory_path = staging / "checkpoint-inventory.json"
        inventory_path.write_text(json.dumps(inventory_doc, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        sums = (
            f"{sha256_file(archive)}  hub-state.tar.gz\n"
            f"{sha256_file(inventory_path)}  checkpoint-inventory.json\n"
        )
        sums_path = staging / "SHA256SUMS"
        sums_path.write_text(sums, encoding="ascii")
        sums_path.chmod(0o600)
        verified = verify_backup(staging)
        output_dir.parent.mkdir(parents=True, exist_ok=True)
        staging.rename(output_dir)
        verified["backup_path"] = str(output_dir)
        verified["required_free_bytes"] = minimum_free_bytes
        verified["available_bytes_before"] = available_bytes
        return verified
    except Exception:
        shutil.rmtree(staging, ignore_errors=True)
        raise


def restore_and_compare(backup_dir: pathlib.Path, restore_parent: pathlib.Path) -> dict[str, Any]:
    """Extract a verified archive to a separate disposable location and compare."""
    verification = verify_backup(backup_dir)
    _assert_outside_production(restore_parent)
    if restore_parent.exists():
        raise BackupError(f"restore destination already exists: {restore_parent}")
    restore_parent.mkdir(parents=True)
    inventory_doc = json.loads((backup_dir / "checkpoint-inventory.json").read_text(encoding="utf-8"))
    try:
        with tarfile.open(backup_dir / "hub-state.tar.gz", "r:gz") as source:
            source.extractall(restore_parent, filter="data")
    except (tarfile.TarError, OSError) as exc:
        raise BackupError(f"checkpoint restore failed: {exc}") from exc
    restored = inventory_tree(restore_parent / "checkpoint")
    expected = inventory_doc["entries"]
    if restored != expected:
        expected_by_path = {entry["path"]: entry for entry in expected}
        restored_by_path = {entry["path"]: entry for entry in restored}
        missing = sorted(set(expected_by_path) - set(restored_by_path))
        unexpected = sorted(set(restored_by_path) - set(expected_by_path))
        differences: dict[str, dict[str, Any]] = {}
        for relative in sorted(set(expected_by_path) & set(restored_by_path)):
            before, after = expected_by_path[relative], restored_by_path[relative]
            fields = {key for key in set(before) | set(after) if before.get(key) != after.get(key)}
            if fields:
                differences[relative] = {
                    field: {"expected": before.get(field), "restored": after.get(field)}
                    for field in sorted(fields)
                }
        detail = {"missing": missing[:10], "unexpected": unexpected[:10], "metadata_or_hash_differences": dict(list(differences.items())[:10])}
        raise BackupError("restored checkpoint differs from verified inventory: " + json.dumps(detail, sort_keys=True))
    validate_required_files(restore_parent / "checkpoint")
    return {
        **verification,
        "restore_path": str(restore_parent / "checkpoint"),
        "restore_inventory_matches": True,
    }


def simulate_interrupted_archive(bundle_root: pathlib.Path, partial_path: pathlib.Path) -> dict[str, Any]:
    """Start a real tar writer, stop it after the first block, and retain no final."""
    _assert_outside_production(bundle_root)
    _assert_outside_production(partial_path)
    partial_path.parent.mkdir(parents=True, exist_ok=True)
    process = subprocess.Popen(
        ["tar", "--format=posix", "-cf", "-", "-C", str(bundle_root.parent), bundle_root.name],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    try:
        assert process.stdout is not None
        block = process.stdout.read(512)
        if not block:
            stderr = process.stderr.read().decode(errors="replace") if process.stderr else ""
            raise BackupError(f"tar emitted no archive data before interruption: {stderr}")
        partial_path.write_bytes(block)
        process.send_signal(signal.SIGKILL)
        process.wait(timeout=10)
    finally:
        if process.poll() is None:
            process.kill()
            process.wait(timeout=5)
        if process.stdout:
            process.stdout.close()
        if process.stderr:
            process.stderr.close()
    if process.returncode == 0:
        raise BackupError("interrupted archive process unexpectedly completed successfully")
    try:
        with tarfile.open(partial_path, "r:") as source:
            source.getmembers()
    except (tarfile.TarError, OSError):
        pass
    else:
        raise BackupError("interrupted partial archive was unexpectedly readable")
    return {"process_exit_code": process.returncode, "partial_bytes": partial_path.stat().st_size, "final_archive_published": False}
