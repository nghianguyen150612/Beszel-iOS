#!/usr/bin/env python3
"""Disposable, real-binary Hub migration harness (P013).

The harness builds the production iOS branch's genuine v0.19 Hub and the
current checkout's v0.21 Hub, seeds only synthetic data through supported Hub
and PocketBase APIs, and keeps all generated state beneath a temporary root.
It never opens /var/lib/beszel-hub.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import pathlib
import shutil
import signal
import socket
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
from datetime import datetime, timezone
from typing import Any

from backup import (
    build_checkpoint_bundle,
    create_cold_backup,
    process_alive,
    restore_and_compare,
)


OLD_019_REV = "0c3e4cd22ac24264834c401ad412e44aeda4e4f3"
P013_STARTING_INTEGRATION_SHA = "d6239915e404668c10453ab2ec44f797b2f0178e"
PROTECTED_DATA_DIR = pathlib.Path("/var/lib/beszel-hub")
HEALTH_TIMEOUT_SECONDS = 45
SYNTHETIC_PASSWORD = "P013-synthetic-only-password-not-a-real-credential"
SYNTHETIC_TOKEN = "P013-synthetic-agent-token-not-a-secret"
ACTIVE_HUBS: list["HubProcess"] = []
ACTIVE_TEMP_PATHS: list[pathlib.Path] = []


class HarnessError(RuntimeError):
    pass


def run(command: list[str], *, cwd: pathlib.Path | None = None, env: dict[str, str] | None = None) -> None:
    print("+", " ".join(command), flush=True)
    subprocess.run(command, cwd=cwd, env=env, check=True)


def free_port() -> int:
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


def request(base: str, method: str, path: str, *, token: str = "", body: Any = None) -> tuple[int, dict[str, str], Any]:
    payload = None if body is None else json.dumps(body).encode()
    headers = {"Accept": "application/json"}
    if payload is not None:
        headers["Content-Type"] = "application/json"
    if token:
        headers["Authorization"] = token
    req = urllib.request.Request(base + path, data=payload, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=10) as response:
            raw = response.read()
            status, response_headers = response.status, dict(response.headers.items())
    except urllib.error.HTTPError as response:
        raw = response.read()
        status, response_headers = response.code, dict(response.headers.items())
    if not raw:
        decoded: Any = None
    else:
        try:
            decoded = json.loads(raw)
        except json.JSONDecodeError:
            decoded = raw.decode("utf-8", errors="replace")
    return status, response_headers, decoded


class HubProcess:
    def __init__(self, binary: pathlib.Path, data_dir: pathlib.Path, logs: pathlib.Path):
        self.binary = binary
        self.data_dir = data_dir
        self.logs = logs
        self.port = free_port()
        self.base = f"http://127.0.0.1:{self.port}"
        self.process: subprocess.Popen[bytes] | None = None
        self.log_file: Any = None
        self.last_pid: int | None = None
        self.last_exit_code: int | None = None

    def start(self, *, env: dict[str, str] | None = None, wait: bool = True) -> None:
        self.port = free_port()
        self.base = f"http://127.0.0.1:{self.port}"
        merged_env = os.environ.copy()
        if env:
            merged_env.update(env)
        self.log_file = self.logs.open("ab")
        self.process = subprocess.Popen(
            [str(self.binary), "serve", "--dir", str(self.data_dir), "--http", f"127.0.0.1:{self.port}"],
            cwd=self.binary.parent,
            env=merged_env,
            stdout=self.log_file,
            stderr=subprocess.STDOUT,
        )
        ACTIVE_HUBS.append(self)
        if wait:
            self.wait_healthy()

    def wait_healthy(self) -> None:
        assert self.process is not None
        deadline = time.monotonic() + HEALTH_TIMEOUT_SECONDS
        last_error = "no response"
        while time.monotonic() < deadline:
            code = self.process.poll()
            if code is not None:
                raise HarnessError(f"Hub exited with {code}; log: {self.logs}\n{tail(self.logs)}")
            try:
                status, _, _ = request(self.base, "GET", "/api/health")
                if status == 200:
                    return
                last_error = f"HTTP {status}"
            except Exception as exc:  # server is expected to refuse connections during startup
                last_error = str(exc)
            time.sleep(0.25)
        raise HarnessError(f"Hub did not become healthy ({last_error}); log: {self.logs}\n{tail(self.logs)}")

    def stop(self, *, force: bool = False) -> None:
        if self.process is None:
            return
        pid = self.process.pid
        if self.process.poll() is None:
            if force:
                self.process.send_signal(signal.SIGKILL)
            else:
                self.process.terminate()
            try:
                self.process.wait(timeout=15)
            except subprocess.TimeoutExpired:
                self.process.kill()
                self.process.wait(timeout=5)
        self.last_pid = pid
        self.last_exit_code = self.process.returncode
        if self.log_file is not None:
            self.log_file.close()
            self.log_file = None
        self.process = None
        if self in ACTIVE_HUBS:
            ACTIVE_HUBS.remove(self)

    def __enter__(self) -> "HubProcess":
        return self

    def __exit__(self, *_: Any) -> None:
        self.stop()


def tail(path: pathlib.Path, limit: int = 5000) -> str:
    try:
        return path.read_text(errors="replace")[-limit:]
    except OSError:
        return "<no log>"


def auth_token(base: str, collection: str, email: str, password: str) -> str:
    status, _, body = request(base, "POST", f"/api/collections/{collection}/auth-with-password", body={"identity": email, "password": password})
    if status != 200 or not isinstance(body, dict) or not body.get("token"):
        raise HarnessError(f"synthetic {collection} authentication failed: HTTP {status}, {body}")
    return str(body["token"])


def create_record(base: str, token: str, collection: str, fields: dict[str, Any]) -> dict[str, Any]:
    status, _, body = request(base, "POST", f"/api/collections/{collection}/records", token=token, body=fields)
    if status not in (200, 201) or not isinstance(body, dict) or not body.get("id"):
        raise HarnessError(f"creating synthetic {collection} record failed: HTTP {status}, {body}")
    return body


def list_records(base: str, token: str, collection: str) -> list[dict[str, Any]]:
    status, _, body = request(base, "GET", f"/api/collections/{collection}/records?perPage=500", token=token)
    if status != 200 or not isinstance(body, dict):
        raise HarnessError(f"listing {collection} failed: HTTP {status}, {body}")
    return list(body.get("items", []))


def collection_schema(base: str, token: str) -> dict[str, dict[str, Any]]:
    status, _, body = request(base, "GET", "/api/collections?perPage=500", token=token)
    if status != 200 or not isinstance(body, dict):
        raise HarnessError(f"cannot read synthetic 0.19 collection schema: HTTP {status}, {body}")
    return {str(item["name"]): item for item in body.get("items", [])}


def write_private(path: pathlib.Path, contents: str) -> None:
    path.write_text(contents, encoding="utf-8")
    path.chmod(0o600)


def fixture_user_id(base: str, token: str, email: str) -> str:
    matches = [item for item in list_records(base, token, "users") if item.get("email") == email]
    if len(matches) != 1:
        raise HarnessError(f"expected one synthetic user {email}, found {len(matches)}")
    return str(matches[0]["id"])


def file_sha256(path: pathlib.Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as source:
        for block in iter(lambda: source.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def required_module_version(source: pathlib.Path, module: str) -> str:
    for line in (source / "go.mod").read_text(encoding="utf-8").splitlines():
        fields = line.strip().split()
        if len(fields) >= 2 and fields[0] == module:
            return fields[1]
    raise HarnessError(f"{module} is not directly pinned in {source / 'go.mod'}")


def open_database_readonly(path: pathlib.Path) -> Any:
    import sqlite3

    connection = sqlite3.connect(f"file:{path}?mode=ro", uri=True)
    connection.row_factory = sqlite3.Row
    return connection


def database_snapshot(path: pathlib.Path) -> dict[str, Any]:
    if not path.is_file():
        raise HarnessError(f"expected Hub database missing: {path}")
    connection = open_database_readonly(path)
    try:
        integrity = [row[0] for row in connection.execute("PRAGMA integrity_check")]
        tables = [row[0] for row in connection.execute("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")]
        migrations = []
        if "_migrations" in tables:
            columns = [row[1] for row in connection.execute("PRAGMA table_info(_migrations)")]
            file_column = next((name for name in ("file", "name", "id") if name in columns), None)
            applied_column = next((name for name in ("applied", "created") if name in columns), None)
            if file_column:
                sql = f'SELECT "{file_column}"' + (f', "{applied_column}"' if applied_column else "") + ' FROM "_migrations" ORDER BY rowid'
                migrations = [list(row) for row in connection.execute(sql)]
        counts: dict[str, int] = {}
        for table in tables:
            if table.startswith("_"):
                continue
            try:
                counts[table] = int(connection.execute(f'SELECT COUNT(*) FROM "{table}"').fetchone()[0])
            except Exception:
                counts[table] = -1
        schema = [list(row) for row in connection.execute(
            "SELECT type, name, tbl_name, sql FROM sqlite_master WHERE type IN ('table','index','trigger','view') ORDER BY type,name"
        )]
        app_settings: dict[str, Any] = {}
        if "_params" in tables:
            params_columns = [row[1] for row in connection.execute("PRAGMA table_info(_params)")]
            if {"id", "value"}.issubset(params_columns):
                row = connection.execute('SELECT "value" FROM "_params" WHERE "id" = ?', ("settings",)).fetchone()
                if row:
                    try:
                        value = json.loads(row[0])
                        meta = value.get("meta", {}) if isinstance(value, dict) else {}
                        app_settings = {"appName": meta.get("appName", ""), "appURL": meta.get("appURL", "")}
                    except (TypeError, json.JSONDecodeError):
                        app_settings = {"parse_error": True}
        return {
            "integrity_check": integrity,
            "tables": tables,
            "record_counts": counts,
            "migration_ledger": migrations,
            "application_settings": app_settings,
            "schema_sha256": hashlib.sha256(json.dumps(schema, sort_keys=True, separators=(",", ":")).encode()).hexdigest(),
        }
    finally:
        connection.close()


def ssh_key_fingerprint(public_key: str, temp_dir: pathlib.Path) -> str:
    public_file = temp_dir / "synthetic-hub.pub"
    write_private(public_file, public_key.rstrip() + "\n")
    result = subprocess.run(["ssh-keygen", "-lf", str(public_file), "-E", "sha256"], check=True, text=True, capture_output=True)
    return result.stdout.split()[1]


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=pathlib.Path, help="copy the generated evidence directory here")
    parser.add_argument("--old-rev", default=OLD_019_REV)
    args = parser.parse_args()
    if args.output:
        args.output = args.output.resolve()
        if args.output == PROTECTED_DATA_DIR or PROTECTED_DATA_DIR in args.output.parents:
            raise HarnessError("refusing to write evidence beneath /var/lib/beszel-hub")

    repo = pathlib.Path(__file__).resolve().parents[2]
    if not (repo / ".git").exists():
        raise HarnessError("run from a git checkout")
    dirty = subprocess.check_output(["git", "status", "--porcelain"], cwd=repo, text=True).strip()
    if dirty:
        raise HarnessError("the source checkout must be clean so the reported Hub source SHA matches the built tree")
    source_commit = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=repo, text=True).strip()
    integration_base = subprocess.check_output(
        ["git", "merge-base", "HEAD", P013_STARTING_INTEGRATION_SHA], cwd=repo, text=True,
    ).strip()
    if integration_base != P013_STARTING_INTEGRATION_SHA:
        raise HarnessError(
            f"checkout does not descend from the verified P013 integration head {P013_STARTING_INTEGRATION_SHA}"
        )
    if args.old_rev != OLD_019_REV:
        raise HarnessError("the v0.19 fixture source revision is pinned and cannot be overridden")
    try:
        subprocess.run(["git", "cat-file", "-e", f"{args.old_rev}:internal/migrations/0_collections_snapshot_0_19_0.go"], cwd=repo, check=True)
    except subprocess.CalledProcessError as exc:
        raise HarnessError("pinned production v0.19 migration source is unavailable") from exc

    with tempfile.TemporaryDirectory(prefix="beszel-p013-hub-migration-") as scratch:
        root = pathlib.Path(scratch).resolve()
        if root == PROTECTED_DATA_DIR or PROTECTED_DATA_DIR in root.parents:
            raise HarnessError("refusing a work directory under /var/lib/beszel-hub")
        print(f"synthetic_root={root}", flush=True)
        print(f"source_head={source_commit}", flush=True)
        print(f"starting_integration_sha={integration_base}", flush=True)
        print(f"v019_source={args.old_rev}", flush=True)
        old_source, data_dir, binaries, logs = root / "source-019", root / "release" / "data", root / "release" / "bin", root / "logs"
        old_source.mkdir()
        data_dir.mkdir(parents=True)
        binaries.mkdir(parents=True)
        logs.mkdir()
        with subprocess.Popen(["git", "archive", args.old_rev], cwd=repo, stdout=subprocess.PIPE) as archive:
            subprocess.run(["tar", "-x", "-C", str(old_source)], stdin=archive.stdout, check=True)
            assert archive.stdout is not None
            archive.stdout.close()
            if archive.wait() != 0:
                raise HarnessError("failed to materialize the pinned genuine v0.19 source")

        go = os.environ.get("GO", "go")
        bun = os.environ.get("BUN", "bun")
        go_version = subprocess.check_output([go, "version"], text=True).strip()
        bun_version = subprocess.check_output([bun, "--version"], text=True).strip()
        if "go1.27.1" not in go_version or bun_version != "1.4.0":
            raise HarnessError(f"P013 fixture requires Go 1.27.1 and Bun 1.4.0 (found {go_version!r}, Bun {bun_version!r})")
        old_binary = binaries / "beszel-hub-0.19"
        old_agent_binary = binaries / "beszel-agent-0.19"
        new_binary = binaries / "beszel-hub-0.21"
        old_pocketbase = required_module_version(old_source, "github.com/pocketbase/pocketbase")
        for source in (old_source,):
            site = source / "internal" / "site"
            run([bun, "install", "--frozen-lockfile"], cwd=site)
            run([bun, "run", "build"], cwd=site)
        run([go, "build", "-trimpath", "-o", str(old_binary), "./internal/cmd/hub"], cwd=old_source)
        run([go, "build", "-trimpath", "-o", str(old_agent_binary), "./internal/cmd/agent"], cwd=old_source)

        new_source = root / "source-021"
        new_source.mkdir()
        with subprocess.Popen(["git", "archive", source_commit], cwd=repo, stdout=subprocess.PIPE) as archive:
            subprocess.run(["tar", "-x", "-C", str(new_source)], stdin=archive.stdout, check=True)
            assert archive.stdout is not None
            archive.stdout.close()
            if archive.wait() != 0:
                raise HarnessError("failed to materialize the current v0.21 Hub source")
        site = new_source / "internal" / "site"
        new_pocketbase = required_module_version(new_source, "github.com/pocketbase/pocketbase")
        run([bun, "install", "--frozen-lockfile"], cwd=site)
        run([bun, "run", "build"], cwd=site)
        run([go, "build", "-trimpath", "-o", str(new_binary), "./internal/cmd/hub"], cwd=new_source)
        old_version = subprocess.check_output([str(old_binary), "--version"], text=True).strip()
        old_agent_version = subprocess.check_output([str(old_agent_binary), "--version"], text=True).strip()
        new_version = subprocess.check_output([str(new_binary), "--version"], text=True).strip()
        if "0.19" not in old_version or "0.21" not in new_version:
            raise HarnessError(f"Hub binary version mismatch: v0.19={old_version!r}, v0.21={new_version!r}")
        if "0.19" not in old_agent_version:
            raise HarnessError(f"Agent binary version mismatch: {old_agent_version!r}")
        print(f"hub_019_sha256={hashlib.sha256(old_binary.read_bytes()).hexdigest()}", flush=True)
        print(f"hub_021_sha256={hashlib.sha256(new_binary.read_bytes()).hexdigest()}", flush=True)

        old_hub = HubProcess(old_binary, data_dir, logs / "hub-019.log")
        old_hub.start(env={"APP_URL": "https://p013.synthetic.invalid"})
        status, _, created = request(old_hub.base, "POST", "/api/beszel/create-user", body={
            "email": "p013-owner@example.invalid", "password": SYNTHETIC_PASSWORD,
        })
        if status != 200:
            raise HarnessError(f"synthetic first-user bootstrap failed: HTTP {status}, {created}")
        owner_email = "p013-owner@example.invalid"
        regular_email = "p013-regular@example.invalid"
        owner_superuser_token = auth_token(old_hub.base, "_superusers", owner_email, SYNTHETIC_PASSWORD)
        owner_token = auth_token(old_hub.base, "users", owner_email, SYNTHETIC_PASSWORD)
        schema = collection_schema(old_hub.base, owner_superuser_token)
        if "users" not in schema or "systems" not in schema or "system_stats" not in schema:
            raise HarnessError("the genuine 0.19 Hub did not initialize its expected collection set")
        if "network_monitors" in schema or "network_monitor_stats" in schema:
            raise HarnessError("v0.19 fixture unexpectedly contains post-v0.19 monitor collections")
        users = list_records(old_hub.base, owner_superuser_token, "users")
        owner_id = fixture_user_id(old_hub.base, owner_superuser_token, owner_email)
        owner_record = next(item for item in users if item.get("id") == owner_id)
        if owner_record.get("role") != "admin":
            raise HarnessError(f"synthetic first user has unexpected role: {owner_record.get('role')!r}")

        regular = create_record(old_hub.base, owner_superuser_token, "users", {
            "email": regular_email,
            "password": SYNTHETIC_PASSWORD,
            "passwordConfirm": SYNTHETIC_PASSWORD,
            "verified": True,
            "role": "user",
        })
        regular_id = str(regular["id"])
        if regular.get("role") != "user":
            raise HarnessError("synthetic regular user did not retain the regular role")

        owner_settings = create_record(old_hub.base, owner_superuser_token, "user_settings", {
            "user": owner_id,
            "settings": {"chartTime": "6h", "emails": [owner_email], "webhooks": []},
        })
        regular_settings = create_record(old_hub.base, owner_superuser_token, "user_settings", {
            "user": regular_id,
            "settings": {"chartTime": "24h", "emails": [regular_email], "webhooks": []},
        })

        system_port = free_port()
        config_text = (
            "systems:\n"
            "  - name: p013-synthetic-system\n"
            "    host: 127.0.0.1\n"
            f"    port: {system_port}\n"
            f"    token: {SYNTHETIC_TOKEN}\n"
            f"    users: [{owner_email}, {regular_email}]\n"
        )
        config_file = data_dir / "config.yml"
        write_private(config_file, config_text)
        old_hub.stop()
        old_hub.start(env={"APP_URL": "https://p013.synthetic.invalid"})
        owner_superuser_token = auth_token(old_hub.base, "_superusers", owner_email, SYNTHETIC_PASSWORD)
        owner_token = auth_token(old_hub.base, "users", owner_email, SYNTHETIC_PASSWORD)

        systems = list_records(old_hub.base, owner_superuser_token, "systems")
        if len(systems) != 1:
            raise HarnessError(f"expected one system from supported 0.19 config.yml, found {len(systems)}")
        system = systems[0]
        system_id = str(system["id"])
        system_users = list(system.get("users", []))
        if sorted(system_users) != sorted([owner_id, regular_id]):
            raise HarnessError(f"config.yml system ownership mismatch: {system_users!r}")

        info = {
            "os": "P013 Synthetic OS",
            "kernel": "P013 Synthetic Kernel 1.0",
            "arch": "amd64",
            "agentVersion": "0.19.0",
        }
        status, _, updated_system = request(old_hub.base, "PATCH", f"/api/collections/systems/records/{system_id}", token=owner_superuser_token, body={"info": info})
        if status != 200:
            raise HarnessError(f"could not seed synthetic system details: HTTP {status}, {updated_system}")

        _, _, public_info = request(old_hub.base, "GET", "/api/beszel/info", token=owner_token)
        public_key = str(public_info.get("key", "")) if isinstance(public_info, dict) else ""
        key_file = data_dir / "id_ed25519"
        if not key_file.is_file() or not public_key.startswith("ssh-ed25519 "):
            raise HarnessError("the initialized 0.19 Hub did not create/read its synthetic SSH identity")
        key_fingerprint = ssh_key_fingerprint(public_key, root)
        fingerprints = list_records(old_hub.base, owner_superuser_token, "fingerprints")
        if not any(item.get("system") == system_id for item in fingerprints):
            raise HarnessError("the system's synthetic agent configuration fingerprint record is missing")

        now = datetime.now(timezone.utc).isoformat(timespec="seconds").replace("+00:00", "Z")
        unix_ms = int(time.time() * 1000)
        seeded: dict[str, dict[str, Any]] = {}
        seeded["system_details"] = create_record(old_hub.base, owner_superuser_token, "system_details", {
            "system": system_id,
            "hostname": "p013-synthetic-host",
            "os": 7,
            "os_name": "Synthetic Linux",
            "kernel": "P013 synthetic kernel",
            "cpu": "Synthetic 8 core CPU",
            "arch": "amd64",
            "cores": 8,
            "threads": 16,
            "memory": 17179869184,
            "podman": False,
        })
        seeded["system_stats"] = create_record(old_hub.base, owner_superuser_token, "system_stats", {
            "system": system_id,
            "type": "1m",
            "stats": {"cpu": 27.5, "mem": 8192, "load": 1.25, "disk": 4096, "net": {"sent": 100, "recv": 200}},
        })
        seeded["container"] = create_record(old_hub.base, owner_superuser_token, "containers", {
            "system": system_id,
            "name": "p013-synthetic-container",
            "status": "running",
            "health": 1,
            "cpu": 4.25,
            "memory": 268435456,
            "net": 32,
            "image": "registry.synthetic.invalid/p013:0.19",
            "ports": "8080/tcp",
            "updated": unix_ms,
        })
        seeded["container_stats"] = create_record(old_hub.base, owner_superuser_token, "container_stats", {
            "system": system_id,
            "type": "1m",
            "stats": [{"name": "p013-synthetic-container", "cpu": 4.25, "memory": 268435456, "net": 32}],
        })
        seeded["systemd_service"] = create_record(old_hub.base, owner_superuser_token, "systemd_services", {
            "system": system_id, "name": "p013-synthetic.service", "state": 0, "sub": 1,
            "cpu": 0.25, "memory": 1048576, "updated": unix_ms,
        })
        seeded["smart_device"] = create_record(old_hub.base, owner_superuser_token, "smart_devices", {
            "system": system_id, "name": "p013-synthetic-disk", "model": "Synthetic Model",
            "state": "PASSED", "capacity": 1000000000000, "temp": 31,
            "firmware": "P013.1", "serial": "SYNTHETIC-ONLY", "type": "NVMe", "hours": 42,
            "cycles": 7, "attributes": {"synthetic": True},
        })
        seeded["zfs_pool"] = create_record(old_hub.base, owner_superuser_token, "zfs_pools", {
            "system": system_id, "name": "p013-synthetic-pool", "health": "ONLINE",
            "size": 1000000000000, "alloc": 250000000000, "free": 750000000000,
            "scrub": {"state": "finished"}, "vdevs": [], "datasets": [],
            "details_updated": now,
        })
        seeded["alert"] = create_record(old_hub.base, owner_superuser_token, "alerts", {
            "user": owner_id, "system": system_id, "name": "CPU", "value": 55, "min": 50,
            "triggered": True, "pending_since": now,
        })
        seeded["alert_history"] = create_record(old_hub.base, owner_superuser_token, "alerts_history", {
            "user": owner_id, "system": system_id, "alert_id": seeded["alert"]["id"],
            "name": "CPU", "value": 55,
        })

        for key, item in seeded.items():
            if item.get("created"):
                item["created"] = str(item["created"])

        # A second genuine v0.19 startup proves that the untouched original
        # fixture can still be opened before its disposable v0.21 clone exists.
        old_hub.stop()
        old_hub.start(env={"APP_URL": "https://p013.synthetic.invalid"})
        reopen_owner_token = auth_token(old_hub.base, "_superusers", owner_email, SYNTHETIC_PASSWORD)
        reopened_users = list_records(old_hub.base, reopen_owner_token, "users")
        reopened_systems = list_records(old_hub.base, reopen_owner_token, "systems")
        if {item.get("id") for item in reopened_users} != {owner_id, regular_id}:
            raise HarnessError("genuine 0.19 reopen changed or lost synthetic user identities")
        reopened_roles = {item["id"]: item.get("role") for item in reopened_users}
        if reopened_roles != {owner_id: "admin", regular_id: "user"}:
            raise HarnessError(f"genuine 0.19 reopen changed user roles: {reopened_roles!r}")
        if len(reopened_systems) != 1 or reopened_systems[0].get("id") != system_id:
            raise HarnessError("genuine 0.19 reopen changed or lost the synthetic system identity")
        if sorted(reopened_systems[0].get("users", [])) != sorted([owner_id, regular_id]):
            raise HarnessError("genuine 0.19 reopen changed system ownership")
        expected_seeded_collections = {
            "system_details": "system_details", "system_stats": "system_stats", "container": "containers",
            "container_stats": "container_stats", "systemd_service": "systemd_services",
            "smart_device": "smart_devices", "zfs_pool": "zfs_pools", "alert": "alerts",
            "alert_history": "alerts_history",
        }
        for key, collection in expected_seeded_collections.items():
            ids = {item.get("id") for item in list_records(old_hub.base, reopen_owner_token, collection)}
            if seeded[key].get("id") not in ids:
                raise HarnessError(f"genuine 0.19 reopen lost synthetic {collection} record {seeded[key].get('id')}")
        # Leave one API-committed synthetic row immediately before an abrupt
        # stop. On SQLite builds that retain a WAL, the cold archive must
        # capture it with the main database rather than copying data.db alone.
        seeded["system_stats_wal"] = create_record(old_hub.base, reopen_owner_token, "system_stats", {
            "system": system_id,
            "type": "1m",
            "stats": {"cpu": 19.5, "mem": 4096, "load": 0.75, "disk": 2048, "net": {"sent": 300, "recv": 400}},
        })
        old_hub.stop(force=True)
        if old_hub.last_pid is None or old_hub.last_exit_code is None or process_alive(old_hub.last_pid):
            raise HarnessError("the synthetic v0.19 Hub writer was not confirmed stopped")

        checkpoint_root = root / "checkpoint"
        checkpoint_root.mkdir()
        hub_port = free_port()
        hub_service_plist = f'''<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>dev.beszel.hub</string>
<key>ProgramArguments</key><array><string>/usr/local/bin/beszel-hub</string><string>serve</string><string>--http</string><string>0.0.0.0:{hub_port}</string><string>--dir</string><string>/var/lib/beszel-hub</string></array>
<key>WorkingDirectory</key><string>/var/lib/beszel-hub</string>
<key>RunAtLoad</key><true/><key>KeepAlive</key><true/>
</dict></plist>
'''
        agent_service_plist = f'''<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>dev.beszel.agent</string>
<key>ProgramArguments</key><array><string>/usr/local/bin/beszel-agent</string></array>
<key>EnvironmentVariables</key><dict>
<key>DATA_DIR</key><string>/var/lib/beszel-agent</string>
<key>LISTEN</key><string>:45876</string><key>KEY</key><string>{public_key.strip()}</string>
<key>LOG_LEVEL</key><string>info</string><key>PATH</key><string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
</dict><key>WorkingDirectory</key><string>/var/lib/beszel-agent</string>
<key>RunAtLoad</key><true/><key>KeepAlive</key><true/>
</dict></plist>
'''
        built_bundle = build_checkpoint_bundle(
            source_data_dir=data_dir,
            old_hub_binary=old_binary,
            old_agent_binary=old_agent_binary,
            bundle_root=checkpoint_root,
            hub_release="v0.19.0-ios.2",
            agent_release="v0.19.0-ios.2",
            hub_service_plist=hub_service_plist,
            agent_service_plist=agent_service_plist,
        )
        checkpoint_bytes = sum(
            entry["size"] for entry in built_bundle["files"] if entry["kind"] == "file"
        )
        backup_store = pathlib.Path(tempfile.mkdtemp(prefix="beszel-p013-backup-outside-staging-")).resolve()
        ACTIVE_TEMP_PATHS.append(backup_store)
        backup_dir = backup_store / "backup-019"
        backup_evidence = create_cold_backup(
            bundle_root=checkpoint_root,
            output_dir=backup_dir,
            hub_pid=old_hub.last_pid,
            hub_exit_confirmed=old_hub.last_exit_code is not None,
            agent_writer_pids=[],
            minimum_free_bytes=(checkpoint_bytes * 120 + 99) // 100,
        )
        restore_evidence = restore_and_compare(backup_dir, root / "restore-019-inventory")
        restored_checkpoint = root / "restore-019-inventory" / "checkpoint"
        restored_data = restored_checkpoint / "var/lib/beszel-hub"
        restored_db_snapshot = database_snapshot(restored_data / "data.db")
        if restored_db_snapshot["integrity_check"] != ["ok"]:
            raise HarnessError(f"restored cold backup SQLite integrity failed: {restored_db_snapshot['integrity_check']}")
        restored_hub = HubProcess(
            restored_checkpoint / "usr/local/bin/beszel-hub", restored_data, logs / "hub-019-restored.log",
        )
        restored_hub.start(env={"APP_URL": "https://p013.synthetic.invalid"})
        restore_health, _, _ = request(restored_hub.base, "GET", "/api/health")
        if restore_health != 200:
            raise HarnessError(f"restored v0.19 Hub health check failed: HTTP {restore_health}")
        restored_owner_token = auth_token(restored_hub.base, "_superusers", owner_email, SYNTHETIC_PASSWORD)
        restored_users = list_records(restored_hub.base, restored_owner_token, "users")
        restored_settings = list_records(restored_hub.base, restored_owner_token, "user_settings")
        restored_systems = list_records(restored_hub.base, restored_owner_token, "systems")
        restored_stats = list_records(restored_hub.base, restored_owner_token, "system_stats")
        if {item.get("id") for item in restored_users} != {owner_id, regular_id}:
            raise HarnessError("cold backup restore changed or lost the original v0.19 account IDs")
        if len(restored_systems) != 1 or restored_systems[0].get("id") != system_id:
            raise HarnessError("cold backup restore changed or lost the original v0.19 system")
        restored_roles = {item["id"]: item.get("role") for item in restored_users}
        if restored_roles != {owner_id: "admin", regular_id: "user"}:
            raise HarnessError(f"cold backup restore changed account roles: {restored_roles!r}")
        if {item.get("id") for item in restored_settings} != {owner_settings["id"], regular_settings["id"]}:
            raise HarnessError("cold backup restore changed or lost user settings")
        if sorted(restored_systems[0].get("users", [])) != sorted([owner_id, regular_id]):
            raise HarnessError("cold backup restore changed system membership")
        if seeded["system_stats_wal"]["id"] not in {item.get("id") for item in restored_stats}:
            raise HarnessError("cold backup restore lost the committed synthetic history row")
        _, _, restored_public_info = request(restored_hub.base, "GET", "/api/beszel/info", token=auth_token(restored_hub.base, "users", owner_email, SYNTHETIC_PASSWORD))
        if not isinstance(restored_public_info, dict) or restored_public_info.get("key") != public_key:
            raise HarnessError("cold backup restore changed the Hub SSH public-key identity")
        restored_hub.stop()
        backup_evidence.pop("backup_path", None)
        restore_evidence.pop("backup_path", None)
        restore_evidence.pop("restore_path", None)

        fixture = {
            "fixture": "synthetic",
            "source_head": source_commit,
            "starting_integration_sha": integration_base,
            "hub_019_source": args.old_rev,
            "hub_019_version": old_version,
            "hub_019_pocketbase_version": old_pocketbase,
            "hub_019_binary_sha256": file_sha256(old_binary),
            "agent_019_version": old_agent_version,
            "agent_019_binary_sha256": file_sha256(old_agent_binary),
            "hub_021_source": source_commit,
            "hub_021_version": new_version,
            "hub_021_pocketbase_version": new_pocketbase,
            "hub_021_binary_sha256": file_sha256(new_binary),
            "toolchain": {"go": go_version, "bun": bun_version, "frontend_builds": "v0.19 and v0.21 production builds passed"},
            "initialized_by": "genuine Beszel v0.19 Hub binary and supported startup/bootstrap APIs",
            "captured_at_utc": datetime.now(timezone.utc).isoformat(),
            "application_version": "0.19.0",
            "accounts": [
                {"id": owner_id, "role": owner_record.get("role"), "is_superuser": True},
                {"id": regular_id, "role": regular.get("role"), "is_superuser": False},
            ],
            "user_settings": [
                {"id": owner_settings["id"], "user": owner_id, "chartTime": "6h", "created": owner_settings.get("created", "")},
                {"id": regular_settings["id"], "user": regular_id, "chartTime": "24h", "created": regular_settings.get("created", "")},
            ],
            "systems": [{"id": system_id, "users": system_users, "info": info}],
            "synthetic_records": {
                key: {
                    "id": str(item.get("id", "")),
                    "system": item.get("system", system_id),
                    "user": item.get("user", ""),
                    "alert_id": item.get("alert_id", ""),
                    "type": item.get("type", ""),
                    "name": item.get("name", ""),
                    "value": item.get("value", None),
                    "min": item.get("min", None),
                    "triggered": item.get("triggered", None),
                    "stats": item.get("stats", None),
                    "created": item.get("created", ""),
                    "updated": item.get("updated", ""),
                    "pending_since": item.get("pending_since", ""),
                    "details_updated": item.get("details_updated", ""),
                    "relation_count": 1 if item.get("system", system_id) == system_id else 0,
                }
                for key, item in seeded.items()
            },
            "agent_config": {"path": "data/config.yml", "sha256": file_sha256(config_file), "mode": oct(config_file.stat().st_mode & 0o777)},
            "ssh_key": {"path": "data/id_ed25519", "fingerprint": key_fingerprint, "mode": oct(key_file.stat().st_mode & 0o777)},
            "agent_fingerprint_records": [
                {"id": item["id"], "system": item.get("system", ""), "token_sha256": hashlib.sha256(str(item.get("token", "")).encode()).hexdigest()}
                for item in fingerprints if item.get("system") == system_id
            ],
            "collections": sorted(schema),
            "collection_schema": schema,
            "cold_backup": {
                **backup_evidence,
                "sidecars": built_bundle["sidecars"],
                "restore_inventory_matches": restore_evidence["restore_inventory_matches"],
                "restored_hub_health_http": restore_health,
                "restored_database_integrity_check": restored_db_snapshot["integrity_check"],
                "restored_account_ids_preserved": True,
                "restored_account_roles_preserved": True,
                "restored_user_settings_ids_preserved": True,
                "restored_system_id_preserved": True,
                "restored_system_membership_preserved": True,
                "restored_committed_history_id": seeded["system_stats_wal"]["id"],
                "restored_ssh_public_key_fingerprint_preserved": key_fingerprint,
                "backup_stored_outside_fixture_staging": backup_store not in root.parents,
                "agent_writer_pids": [],
            },
        }
        old_hub.stop()
        fixture["database_019"] = database_snapshot(data_dir / "data.db")
        if fixture["database_019"]["integrity_check"] != ["ok"]:
            raise HarnessError(f"0.19 database integrity check failed: {fixture['database_019']['integrity_check']}")
        if fixture["database_019"]["application_settings"].get("appURL") != "https://p013.synthetic.invalid":
            raise HarnessError("the synthetic Hub application URL setting was not persisted by v0.19")
        (root / "fixture-manifest.json").write_text(json.dumps(fixture, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        print("fixture_schema=initialized_by_genuine_019_hub", flush=True)
        print("fixture_bootstrap=synthetic_first_user_endpoint", flush=True)
        print("fixture_accounts=admin_superuser_and_regular_user", flush=True)
        print(f"fixture_019_collections={len(schema)}", flush=True)
        print(f"fixture_system_id={system_id}", flush=True)
        print(f"fixture_owner_id={owner_id}", flush=True)
        print(f"fixture_regular_id={regular_id}", flush=True)
        print(f"fixture_ssh_fingerprint={key_fingerprint}", flush=True)
        print(f"fixture_seeded_record_groups={len(seeded)}", flush=True)
        print(f"fixture_database_sha256={file_sha256(data_dir / 'data.db')}", flush=True)
        old_hub.stop()

        if args.output:
            args.output.mkdir(parents=True, exist_ok=True)
            shutil.copy2(root / "fixture-manifest.json", args.output / "fixture-manifest.json")
        print("P013 genuine v0.19 fixture: PASS", flush=True)
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except HarnessError as exc:
        print(f"P013 harness error: {exc}", file=sys.stderr)
        raise SystemExit(1)
    finally:
        # Keep failed or interrupted fixture runs from leaving a synthetic Hub
        # process writing into a temporary directory after it has been removed.
        for hub in list(ACTIVE_HUBS):
            hub.stop(force=True)
        for temporary_path in ACTIVE_TEMP_PATHS:
            shutil.rmtree(temporary_path, ignore_errors=True)
