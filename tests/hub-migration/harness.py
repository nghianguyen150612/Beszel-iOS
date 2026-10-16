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
import http.server
import json
import os
import pathlib
import shutil
import signal
import socket
import subprocess
import sys
import tempfile
import threading
import time
import urllib.error
import urllib.request
from concurrent.futures import ThreadPoolExecutor
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

    def start(self, *, env: dict[str, str] | None = None, wait: bool = True, port: int | None = None) -> None:
        self.port = free_port() if port is None else port
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


def schema_field_types(schema: dict[str, dict[str, Any]], collection: str) -> dict[str, str]:
    if collection not in schema:
        raise HarnessError(f"migrated collection missing: {collection}")
    return {str(field.get("name")): str(field.get("type")) for field in schema[collection].get("fields", [])}


def validate_v021_schema(before: dict[str, dict[str, Any]], after: dict[str, dict[str, Any]]) -> dict[str, Any]:
    new_collection_names = {"network_monitors", "network_monitor_stats"}
    added_collections = set(after) - set(before)
    missing_legacy = sorted(set(before) - set(after))
    if missing_legacy:
        raise HarnessError(f"v0.21 migration removed legacy collections: {missing_legacy}")
    if added_collections != new_collection_names:
        raise HarnessError(f"v0.21 added unexpected collections: {sorted(added_collections)}")

    expected_added_fields = {
        "alerts": {"state": "json"},
        "alerts_history": {"monitor_name": "text"},
        "zfs_pools": {"status": "text"},
    }
    expected_new_collection_fields = {
        "network_monitors": {
            "id": "text", "system": "relation", "target": "text", "protocol": "select",
            "port": "number", "interval": "number", "res": "number", "resAvg1h": "number",
            "resMin1h": "number", "resMax1h": "number", "loss1h": "number", "enabled": "bool",
            "created": "autodate", "updated": "date", "certInfo": "json", "server": "text",
            "skipTlsVerify": "bool",
        },
        "network_monitor_stats": {
            "id": "text", "system": "relation", "monitor": "relation", "total_count": "number",
            "success_count": "number", "res_sum": "number", "res_min": "number", "res_max": "number",
            "type": "select", "created": "number",
        },
    }

    for collection, expected in expected_new_collection_fields.items():
        actual = schema_field_types(after, collection)
        if actual != expected:
            raise HarnessError(f"v0.21 {collection} fields differ from the accepted schema: {actual!r}")

    actual_added_fields: dict[str, dict[str, str]] = {}
    for collection in set(before) & set(after):
        old_fields = {str(field.get("name")): field for field in before[collection].get("fields", [])}
        new_fields = {str(field.get("name")): field for field in after[collection].get("fields", [])}
        removed = sorted(set(old_fields) - set(new_fields))
        if removed:
            raise HarnessError(f"v0.21 migration removed fields from {collection}: {removed}")
        added = set(new_fields) - set(old_fields)
        if added:
            actual_added_fields[collection] = {field: str(new_fields[field].get("type")) for field in sorted(added)}
        for field in set(old_fields) & set(new_fields):
            if old_fields[field] == new_fields[field]:
                continue
            if collection != "alerts" or field != "name":
                raise HarnessError(f"v0.21 migration unexpectedly changed {collection}.{field}")
            old_definition, new_definition = dict(old_fields[field]), dict(new_fields[field])
            old_values = old_definition.pop("values", [])
            new_values = new_definition.pop("values", [])
            if old_definition != new_definition or new_values != [*old_values, "NetworkMonitorLoss"]:
                raise HarnessError("v0.21 alerts.name changed beyond adding NetworkMonitorLoss")
    if actual_added_fields != expected_added_fields:
        raise HarnessError(f"v0.21 legacy field additions differ from the accepted migration: {actual_added_fields!r}")

    # The container updatable field and the older ZFS display/raw fields are
    # already present in a genuine 0.19 fixture; migration must preserve them.
    for collection, expected in {
        "containers": {"updatable": "bool"},
        "zfs_pools": {"display_name": "text", "raw": "bool"},
    }.items():
        old_types = schema_field_types(before, collection)
        new_types = schema_field_types(after, collection)
        for field, kind in expected.items():
            if old_types.get(field) != kind or new_types.get(field) != kind:
                raise HarnessError(f"v0.19 {collection}.{field} must survive v0.21 migration as {kind}")

    allowed_alert_index = "CREATE INDEX idx_alerts_system_name ON alerts (system, name)"
    index_changes: dict[str, dict[str, list[str]]] = {}
    for collection in set(before) & set(after):
        old_indexes = set(before[collection].get("indexes", []))
        new_indexes = set(after[collection].get("indexes", []))
        removed = old_indexes - new_indexes
        added = new_indexes - old_indexes
        if removed:
            raise HarnessError(f"v0.21 migration removed indexes from {collection}: {sorted(removed)}")
        if added:
            if collection != "alerts" or added != {allowed_alert_index}:
                raise HarnessError(f"v0.21 migration added unexpected indexes to {collection}: {sorted(added)}")
            index_changes[collection] = {"added": sorted(added), "removed": []}

    expected_alert_rules = {
        "createRule": '@request.auth.id != "" && user = @request.auth.id && system.users.id ?= @request.auth.id',
        "updateRule": '@request.auth.id != "" && user = @request.auth.id && @request.body.user:changed = false && @request.body.system:changed = false && system.users.id ?= @request.auth.id',
    }
    changed_rules: dict[str, dict[str, str]] = {}
    rule_keys = ("listRule", "viewRule", "createRule", "updateRule", "deleteRule")
    for collection in set(before) & set(after):
        changes = {
            key: str(after[collection].get(key) or "")
            for key in rule_keys
            if before[collection].get(key) != after[collection].get(key)
        }
        if changes:
            if collection != "alerts" or changes != expected_alert_rules:
                raise HarnessError(f"v0.21 migration changed unexpected {collection} authorization rules: {changes!r}")
            changed_rules[collection] = changes

    # Apart from migration timestamps, the full collection metadata must be
    # unchanged unless a field/index/rule delta above explicitly allows it.
    structural_keys = {"fields", "indexes", "updated", *rule_keys}
    for collection in set(before) & set(after):
        old_metadata = {key: value for key, value in before[collection].items() if key not in structural_keys}
        new_metadata = {key: value for key, value in after[collection].items() if key not in structural_keys}
        if old_metadata != new_metadata:
            raise HarnessError(f"v0.21 migration unexpectedly changed {collection} collection metadata")

    monitor_indexes = set(after["network_monitor_stats"].get("indexes", []))
    for index_name in ("idx_nms_system_type_created", "idx_nms_monitor_type_created", "idx_nms_type_created"):
        if not any(index_name in statement for statement in monitor_indexes):
            raise HarnessError(f"v0.21 monitor history index missing: {index_name}")
    systems_id = str(after["systems"].get("id", ""))
    monitors_id = str(after["network_monitors"].get("id", ""))
    monitor_fields = {field.get("name"): field for field in after["network_monitor_stats"].get("fields", [])}
    if monitor_fields.get("system", {}).get("collectionId") != systems_id:
        raise HarnessError("v0.21 network_monitor_stats.system relation targets the wrong collection")
    if monitor_fields.get("monitor", {}).get("collectionId") != monitors_id:
        raise HarnessError("v0.21 network_monitor_stats.monitor relation targets the wrong collection")

    expected_membership_rule = '@request.auth.id != "" && system.users.id ?= @request.auth.id'
    monitor_rules = after["network_monitors"]
    if monitor_rules.get("listRule") != expected_membership_rule or monitor_rules.get("viewRule") != expected_membership_rule:
        raise HarnessError("v0.21 network monitor reads must be restricted to system members")
    for rule in ("createRule", "updateRule", "deleteRule"):
        expected = expected_membership_rule + ' && @request.auth.role != "readonly"'
        if monitor_rules.get(rule) != expected:
            raise HarnessError(f"v0.21 network monitor {rule} must require membership and a writable role")
    if not any("idx_nm_system_enabled" in statement for statement in after["network_monitors"].get("indexes", [])):
        raise HarnessError("v0.21 network monitor system/enabled index missing")
    stats_rules = after["network_monitor_stats"]
    if stats_rules.get("listRule") != expected_membership_rule or any(
        stats_rules.get(rule) is not None for rule in ("viewRule", "createRule", "updateRule", "deleteRule")
    ):
        raise HarnessError("v0.21 monitor history rules do not enforce member reads and internal-only writes")

    for collection in ("alerts", "alerts_history"):
        record = after[collection]
        for rule in ("listRule", "deleteRule"):
            value = str(record.get(rule) or "")
            if "user = @request.auth.id" not in value:
                raise HarnessError(f"v0.21 {collection}.{rule} no longer binds access to the owning user")
    return {
        "legacy_collections_preserved": sorted(before),
        "added_collections": sorted(added_collections),
        "added_legacy_fields": actual_added_fields,
        "changed_fields": {"alerts.name.values_added": ["NetworkMonitorLoss"]},
        "index_changes": index_changes,
        "authorization_rule_changes": changed_rules,
        "preserved_preexisting_fields": {"containers.updatable": "bool", "zfs_pools.display_name": "text", "zfs_pools.raw": "bool"},
        "new_collection_fields": expected_new_collection_fields,
        "network_monitor_indexes": sorted(index for index in monitor_indexes if "idx_nms_" in index),
        "relation_targets": {"network_monitor_stats.system": systems_id, "network_monitor_stats.monitor": monitors_id},
        "network_monitor_rules": {rule: after["network_monitors"].get(rule) for rule in rule_keys},
        "network_monitor_stats_rules": {rule: after["network_monitor_stats"].get(rule) for rule in rule_keys},
        "alert_rules": {
            collection: {rule: after[collection].get(rule) for rule in ("listRule", "viewRule", "createRule", "updateRule", "deleteRule")}
            for collection in ("alerts", "alerts_history")
        },
    }


def verify_fixture_api_state(
    base: str,
    *,
    owner_email: str,
    password: str,
    owner_id: str,
    regular_id: str,
    system_id: str,
    owner_settings_id: str,
    regular_settings_id: str,
    info: dict[str, Any],
    seeded: dict[str, dict[str, Any]],
    public_key: str,
    key_fingerprint: str,
) -> dict[str, Any]:
    """Check account, relation, historical-ID, settings, and key invariants via API."""
    token = auth_token(base, "_superusers", owner_email, password)
    user_token = auth_token(base, "users", owner_email, password)
    users = list_records(base, token, "users")
    roles = {str(item["id"]): item.get("role") for item in users}
    if roles != {owner_id: "admin", regular_id: "user"}:
        raise HarnessError(f"account IDs or roles changed: {roles!r}")

    settings = list_records(base, token, "user_settings")
    settings_by_id = {str(item["id"]): item for item in settings}
    if set(settings_by_id) != {owner_settings_id, regular_settings_id}:
        raise HarnessError("user settings IDs changed or unexpected settings were created")
    expected_settings = {owner_settings_id: (owner_id, "6h"), regular_settings_id: (regular_id, "24h")}
    for setting_id, (user_id, chart_time) in expected_settings.items():
        item = settings_by_id[setting_id]
        if item.get("user") != user_id or item.get("settings", {}).get("chartTime") != chart_time:
            raise HarnessError(f"user settings relation or value changed: {setting_id}")

    systems = list_records(base, token, "systems")
    if len(systems) != 1 or systems[0].get("id") != system_id:
        raise HarnessError("system ID changed or an unexpected system was created")
    if sorted(systems[0].get("users", [])) != sorted([owner_id, regular_id]):
        raise HarnessError("system ownership changed")
    if systems[0].get("info") != info:
        raise HarnessError("synthetic system details changed")

    collections = {
        "system_details": "system_details", "system_stats": "system_stats", "system_stats_wal": "system_stats",
        "container": "containers", "container_stats": "container_stats", "systemd_service": "systemd_services",
        "smart_device": "smart_devices", "zfs_pool": "zfs_pools", "alert": "alerts", "alert_history": "alerts_history",
    }
    preserved_ids: dict[str, str] = {}
    for key, collection in collections.items():
        item_id = str(seeded[key]["id"])
        matches = [item for item in list_records(base, token, collection) if item.get("id") == item_id]
        if len(matches) != 1:
            raise HarnessError(f"migration lost or duplicated {collection} record {item_id}")
        item = matches[0]
        if item.get("system") != system_id:
            raise HarnessError(f"migration changed {collection} system relation for {item_id}")
        for property_name in ("user", "alert_id", "type", "name", "value", "min", "triggered", "stats"):
            expected = seeded[key].get(property_name)
            if expected is not None and item.get(property_name) != expected:
                raise HarnessError(f"migration changed {collection}.{property_name} for {item_id}")
        preserved_ids[key] = item_id

    history_ids = {item.get("id") for item in list_records(base, token, "system_stats")}
    if seeded["system_stats"]["id"] not in history_ids or seeded["system_stats_wal"]["id"] not in history_ids:
        raise HarnessError("historical system metric IDs did not survive")
    _, _, info_body = request(base, "GET", "/api/beszel/info", token=user_token)
    if not isinstance(info_body, dict) or info_body.get("key") != public_key:
        raise HarnessError("Hub SSH public-key identity changed")
    fingerprints = list_records(base, token, "fingerprints")
    matching_fingerprint = [item for item in fingerprints if item.get("system") == system_id]
    if len(matching_fingerprint) != 1:
        raise HarnessError("Hub agent fingerprint relation changed")

    return {
        "account_roles": roles,
        "user_settings_ids": sorted(settings_by_id),
        "system_id": system_id,
        "system_users": sorted(systems[0].get("users", [])),
        "preserved_record_ids": preserved_ids,
        "history_count": len(history_ids),
        "fingerprint_id": matching_fingerprint[0]["id"],
        "ssh_public_key_fingerprint": key_fingerprint,
    }


def verify_migrated_monitor_alert_api(
    base: str,
    *,
    owner_email: str,
    regular_email: str,
    password: str,
    owner_id: str,
    regular_id: str,
    system_id: str,
) -> dict[str, Any]:
    """Exercise migrated collections with regular-user PocketBase API auth."""
    regular_token = auth_token(base, "users", regular_email, password)
    owner_token = auth_token(base, "users", owner_email, password)
    owner_superuser_token = auth_token(base, "_superusers", owner_email, password)
    monitor_status, _, monitor = request(base, "POST", "/api/collections/network_monitors/records", token=regular_token, body={
        "system": system_id,
        "target": "http://p013.synthetic.invalid/health",
        "protocol": "http",
        "port": 80,
        "interval": 60,
        "enabled": False,
        "server": "p013.synthetic.invalid",
        "skipTlsVerify": True,
        "certInfo": {"issuer": "P013 Synthetic CA", "expires": 1800000000000},
    })
    if monitor_status not in (200, 201) or not isinstance(monitor, dict) or not monitor.get("id"):
        raise HarnessError(f"regular-user network monitor create failed: HTTP {monitor_status}, {monitor}")
    monitor_id = str(monitor["id"])
    monitor_patch_status, _, patched_monitor = request(
        base,
        "PATCH",
        f"/api/collections/network_monitors/records/{monitor_id}",
        token=regular_token,
        # Target/protocol/port/server form the stable ID; update configuration
        # fields that preserve identity so the API returns the updated record.
        body={"interval": 120, "skipTlsVerify": False},
    )
    if monitor_patch_status != 200 or not isinstance(patched_monitor, dict):
        raise HarnessError(f"regular-user network monitor update failed: HTTP {monitor_patch_status}, {patched_monitor}")
    if patched_monitor.get("target") != "http://p013.synthetic.invalid/health" or patched_monitor.get("interval") != 120 or patched_monitor.get("skipTlsVerify") is not False:
        raise HarnessError("network monitor update did not persist through the supported API")
    monitor_read_status, _, monitor_read = request(
        base, "GET", f"/api/collections/network_monitors/records/{monitor_id}", token=regular_token,
    )
    if monitor_read_status != 200 or not isinstance(monitor_read, dict) or monitor_read.get("system") != system_id:
        raise HarnessError(f"regular-user network monitor read failed: HTTP {monitor_read_status}, {monitor_read}")
    monitor_list_status, _, monitor_list = request(
        base, "GET", "/api/collections/network_monitors/records?perPage=500", token=regular_token,
    )
    if monitor_list_status != 200 or not isinstance(monitor_list, dict) or monitor_id not in {item.get("id") for item in monitor_list.get("items", [])}:
        raise HarnessError("regular-user network monitor list did not include the owned monitor")
    # Network-monitor results are written by the Hub's internal Agent ingestion
    # path, not by regular users. Use a synthetic superuser through the supported
    # collection API to create one result and exercise history visibility/rules.
    monitor_stats_status, _, monitor_stats = request(
        base,
        "POST",
        "/api/collections/network_monitor_stats/records",
        token=owner_superuser_token,
        body={
            "system": system_id,
            "monitor": monitor_id,
            "type": "1m",
            "created": int(time.time() * 1000),
            "res_min": 12,
            "res_max": 24,
            "total_count": 4,
            "success_count": 3,
            "res_sum": 54,
        },
    )
    if monitor_stats_status not in (200, 201) or not isinstance(monitor_stats, dict) or not monitor_stats.get("id"):
        raise HarnessError(f"synthetic monitor result create failed: HTTP {monitor_stats_status}, {monitor_stats}")
    monitor_stats_id = str(monitor_stats["id"])
    stats_status, _, stats = request(
        base,
        "GET",
        f"/api/collections/network_monitor_stats/records?filter=monitor%3D%22{monitor_id}%22&perPage=500",
        token=regular_token,
    )
    if stats_status != 200 or not isinstance(stats, dict):
        raise HarnessError(f"regular-user network monitor history query failed: HTTP {stats_status}, {stats}")
    history_matches = [item for item in stats.get("items", []) if item.get("id") == monitor_stats_id]
    if len(history_matches) != 1:
        raise HarnessError("owned network monitor history query omitted its synthetic result")
    history_record = history_matches[0]
    if history_record.get("system") != system_id or history_record.get("monitor") != monitor_id:
        raise HarnessError("synthetic network monitor result has invalid system/monitor relations")
    if history_record.get("total_count") != 4 or history_record.get("success_count") != 3 or history_record.get("res_sum") != 54:
        raise HarnessError("synthetic network monitor result values changed through the history API")

    alert_status, _, alert = request(base, "POST", "/api/collections/alerts/records", token=regular_token, body={
        "user": regular_id,
        "system": system_id,
        "name": "Memory",
        "min": 40,
        "value": 60,
        "triggered": False,
    })
    if alert_status not in (200, 201) or not isinstance(alert, dict) or not alert.get("id"):
        raise HarnessError(f"regular-user alert create failed: HTTP {alert_status}, {alert}")
    alert_id = str(alert["id"])
    alert_patch_status, _, patched_alert = request(
        base, "PATCH", f"/api/collections/alerts/records/{alert_id}", token=regular_token, body={"value": 61},
    )
    if alert_patch_status != 200 or not isinstance(patched_alert, dict) or patched_alert.get("value") != 61:
        raise HarnessError(f"regular-user alert update failed: HTTP {alert_patch_status}, {patched_alert}")
    alert_read_status, _, _ = request(
        base, "GET", f"/api/collections/alerts/records/{alert_id}", token=owner_token,
    )
    if alert_read_status not in (403, 404):
        raise HarnessError(f"another account read a regular user's alert: HTTP {alert_read_status}")
    cross_user_update_status, _, _ = request(
        base, "PATCH", f"/api/collections/alerts/records/{alert_id}", token=owner_token, body={"value": 62},
    )
    if cross_user_update_status not in (403, 404):
        raise HarnessError(f"another account updated a regular user's alert: HTTP {cross_user_update_status}")
    regular_history_status, _, regular_history = request(
        base, "GET", "/api/collections/alerts_history/records?perPage=500", token=regular_token,
    )
    if regular_history_status != 200 or not isinstance(regular_history, dict):
        raise HarnessError(f"regular-user alert history query failed: HTTP {regular_history_status}, {regular_history}")
    if alert_id in {item.get("id") for item in regular_history.get("items", [])}:
        raise HarnessError("alert creation unexpectedly fabricated an alert-history event")

    outsider_email = "p013-outsider@example.invalid"
    outsider_create_status, _, outsider = request(
        base,
        "POST",
        "/api/collections/users/records",
        token=owner_superuser_token,
        body={
            "email": outsider_email,
            "password": password,
            "passwordConfirm": password,
            "role": "user",
            "verified": True,
        },
    )
    if outsider_create_status not in (200, 201) or not isinstance(outsider, dict) or not outsider.get("id"):
        raise HarnessError(f"could not create synthetic nonmember through the supported API: HTTP {outsider_create_status}, {outsider}")
    outsider_id = str(outsider["id"])
    if outsider.get("role") != "user":
        raise HarnessError(f"new synthetic OAuth/API-style account received unexpected role: {outsider.get('role')}")
    outsider_token = auth_token(base, "users", outsider_email, password)

    outsider_monitor_read, _, _ = request(
        base, "GET", f"/api/collections/network_monitors/records/{monitor_id}", token=outsider_token,
    )
    if outsider_monitor_read not in (403, 404):
        raise HarnessError(f"nonmember read another system's monitor: HTTP {outsider_monitor_read}")
    outsider_monitor_update, _, _ = request(
        base,
        "PATCH",
        f"/api/collections/network_monitors/records/{monitor_id}",
        token=outsider_token,
        body={"interval": 300},
    )
    if outsider_monitor_update not in (403, 404):
        raise HarnessError(f"nonmember updated another system's monitor: HTTP {outsider_monitor_update}")
    outsider_monitors = list_records(base, outsider_token, "network_monitors")
    if monitor_id in {item.get("id") for item in outsider_monitors}:
        raise HarnessError("nonmember monitor list exposed a foreign monitor")
    outsider_stats_status, _, outsider_stats = request(
        base,
        "GET",
        f"/api/collections/network_monitor_stats/records?filter=monitor%3D%22{monitor_id}%22&perPage=500",
        token=outsider_token,
    )
    if outsider_stats_status != 200 or not isinstance(outsider_stats, dict):
        raise HarnessError(f"nonmember monitor-history query failed: HTTP {outsider_stats_status}, {outsider_stats}")
    if monitor_id in {item.get("monitor") for item in outsider_stats.get("items", [])}:
        raise HarnessError("nonmember monitor-history query exposed foreign records")

    outsider_alert_read, _, _ = request(
        base, "GET", f"/api/collections/alerts/records/{alert_id}", token=outsider_token,
    )
    if outsider_alert_read not in (403, 404):
        raise HarnessError(f"nonmember read another user's alert: HTTP {outsider_alert_read}")
    outsider_alert_update, _, _ = request(
        base,
        "PATCH",
        f"/api/collections/alerts/records/{alert_id}",
        token=outsider_token,
        body={"value": 99},
    )
    if outsider_alert_update not in (403, 404):
        raise HarnessError(f"nonmember updated another user's alert: HTTP {outsider_alert_update}")
    outsider_alerts = list_records(base, outsider_token, "alerts")
    if alert_id in {item.get("id") for item in outsider_alerts}:
        raise HarnessError("nonmember alert list exposed another user's alert")
    outsider_history_status, _, outsider_history = request(
        base, "GET", "/api/collections/alerts_history/records?perPage=500", token=outsider_token,
    )
    if outsider_history_status != 200 or not isinstance(outsider_history, dict):
        raise HarnessError(f"nonmember alert-history query failed: HTTP {outsider_history_status}, {outsider_history}")
    if regular_id in {item.get("user") for item in outsider_history.get("items", [])}:
        raise HarnessError("nonmember alert-history query exposed another user's history")

    outsider_delete_status, _, _ = request(
        base, "DELETE", f"/api/collections/users/records/{outsider_id}", token=owner_superuser_token,
    )
    if outsider_delete_status not in (200, 204):
        raise HarnessError(f"could not remove synthetic nonmember through the supported API: HTTP {outsider_delete_status}")

    return {
        "regular_user_monitor_create_update_read_list": True,
        "regular_user_monitor_history_query_http": stats_status,
        "regular_user_monitor_history_record_count": len(stats.get("items", [])),
        "regular_user_monitor_result_preserved_by_history_api": True,
        "monitor_stats_id": monitor_stats_id,
        "nonmember_monitor_view_http": outsider_monitor_read,
        "nonmember_monitor_update_http": outsider_monitor_update,
        "nonmember_monitor_query_no_leak": True,
        "nonmember_monitor_history_query_no_leak": True,
        "regular_user_alert_create_update": True,
        "cross_user_alert_read_http": alert_read_status,
        "cross_user_alert_update_http": cross_user_update_status,
        "regular_user_alert_history_query_http": regular_history_status,
        "nonmember_alert_view_http": outsider_alert_read,
        "nonmember_alert_update_http": outsider_alert_update,
        "nonmember_alert_query_no_leak": True,
        "nonmember_alert_history_query_no_leak": True,
        "synthetic_nonmember_account_deleted_http": outsider_delete_status,
        "monitor_id": monitor_id,
        "alert_id": alert_id,
        "system_id": system_id,
        "regular_user_id": regular_id,
        "records_must_survive_restart": True,
    }


def verify_and_delete_migrated_monitor_alert_api(base: str, *, regular_email: str, password: str, expected: dict[str, Any]) -> dict[str, Any]:
    """Verify migrated API records survive restart, then remove via API."""
    token = auth_token(base, "users", regular_email, password)
    ids = {
        "network_monitors": str(expected["monitor_id"]),
        "network_monitor_stats": str(expected["monitor_stats_id"]),
        "alerts": str(expected["alert_id"]),
    }
    for collection, record_id in ids.items():
        if collection == "alerts":
            # The migrated alerts collection intentionally has no view rule;
            # owners read their alerts through the authorized list endpoint.
            matches = [item for item in list_records(base, token, collection) if item.get("id") == record_id]
            record = matches[0] if len(matches) == 1 else None
            status = 200 if record is not None else 404
        elif collection == "network_monitor_stats":
            matches = [item for item in list_records(base, token, collection) if item.get("id") == record_id]
            record = matches[0] if len(matches) == 1 else None
            status = 200 if record is not None else 404
        else:
            status, _, record = request(base, "GET", f"/api/collections/{collection}/records/{record_id}", token=token)
        if status != 200 or not isinstance(record, dict):
            raise HarnessError(f"{collection} record {record_id} did not survive Hub restart: HTTP {status}, {record}")
        if record.get("system") != expected["system_id"]:
            raise HarnessError(f"{collection} record {record_id} changed system ownership across restart")
        if collection == "network_monitor_stats" and record.get("monitor") != expected["monitor_id"]:
            raise HarnessError(f"monitor result {record_id} changed monitor ownership across restart")
        if collection == "alerts" and record.get("user") != expected["regular_user_id"]:
            raise HarnessError(f"alert record {record_id} changed user ownership across restart")
    deleted: dict[str, int] = {}
    for collection, record_id in ids.items():
        if collection == "network_monitor_stats":
            # The monitor's delete route owns cascade cleanup of result history;
            # the result collection intentionally denies direct user deletes.
            continue
        status, _, _ = request(base, "DELETE", f"/api/collections/{collection}/records/{record_id}", token=token)
        if status not in (200, 204):
            raise HarnessError(f"regular-user {collection} delete failed: HTTP {status}")
        deleted[collection] = status
        if collection == "alerts":
            remaining = [item for item in list_records(base, token, collection) if item.get("id") == record_id]
            if remaining:
                raise HarnessError(f"deleted alert record {record_id} remained in the authorized list")
        else:
            if collection == "network_monitor_stats":
                remaining = [item for item in list_records(base, token, collection) if item.get("id") == record_id]
                if remaining:
                    raise HarnessError(f"deleted monitor or monitor result remained visible: {record_id}")
            else:
                read_status, _, _ = request(base, "GET", f"/api/collections/{collection}/records/{record_id}", token=token)
                if read_status != 404:
                    raise HarnessError(f"deleted {collection} record remained visible: HTTP {read_status}")
    return {"post_restart_records_preserved": True, "regular_user_delete_http": deleted, "deleted_records_not_found": True}


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


def clone_backup_data(backup_dir: pathlib.Path, root: pathlib.Path, name: str) -> pathlib.Path:
    """Extract a verified full checkpoint and make an isolated candidate DB copy."""
    restored = root / f"{name}-restore"
    restore_and_compare(backup_dir, restored)
    source_data = restored / "checkpoint/var/lib/beszel-hub"
    candidate_data = root / f"{name}-candidate-data"
    shutil.copytree(source_data, candidate_data, symlinks=True)
    return candidate_data


def inject_migration_ledger_failure(database: pathlib.Path, migration_name: str) -> None:
    """Add a synthetic SQLite trigger that stops one named ledger insert."""
    import sqlite3

    connection = sqlite3.connect(database)
    try:
        columns = [row[1] for row in connection.execute('PRAGMA table_info("_migrations")')]
        file_column = next((name for name in ("file", "name", "id") if name in columns), None)
        if not file_column:
            raise HarnessError("cannot inject migration failure: _migrations filename column is unknown")
        literal = migration_name.replace("'", "''")
        connection.execute(
            f'''CREATE TRIGGER p013_fail_named_migration
                BEFORE INSERT ON "_migrations"
                WHEN NEW."{file_column}" = '{literal}'
                BEGIN SELECT RAISE(ABORT, 'P013 deterministic migration-sequence failpoint'); END''',
        )
        connection.commit()
    finally:
        connection.close()


def inject_post_schema_startup_failure(database: pathlib.Path) -> None:
    """Fail collection auth initialization only after all schema migrations ledger."""
    import sqlite3

    connection = sqlite3.connect(database)
    try:
        columns = [row[1] for row in connection.execute('PRAGMA table_info("_migrations")')]
        file_column = next((name for name in ("file", "name", "id") if name in columns), None)
        if not file_column:
            raise HarnessError("cannot inject post-schema failure: _migrations filename column is unknown")
        connection.execute(
            f'''CREATE TRIGGER p013_fail_after_schema
                BEFORE UPDATE ON "_collections"
                WHEN OLD."name" = 'alerts'
                  AND EXISTS (SELECT 1 FROM "_migrations" WHERE "{file_column}" = '1790964708_network_monitor_skip_tls_verify.go')
                BEGIN SELECT RAISE(ABORT, 'P013 deterministic post-schema startup failpoint'); END'''
        )
        connection.commit()
    finally:
        connection.close()


def wait_for_candidate_failure(hub: HubProcess, timeout: float = 90) -> int:
    """Wait for a deliberate startup failure and reject any healthy candidate."""
    if hub.process is None:
        raise HarnessError("candidate process was not started")
    process = hub.process
    deadline = time.monotonic() + timeout
    healthy = False
    while time.monotonic() < deadline and process.poll() is None:
        try:
            status, _, _ = request(hub.base, "GET", "/api/health")
            if status == 200:
                healthy = True
                break
        except Exception:
            pass
        time.sleep(0.1)
    if healthy or process.poll() is None:
        hub.stop(force=True)
        if healthy:
            raise HarnessError(f"failure injection unexpectedly produced a healthy Hub: {hub.logs}")
        raise HarnessError(f"candidate did not exit at the deterministic failure point: {hub.logs}\n{tail(hub.logs)}")
    hub.stop()
    if hub.last_exit_code in (None, 0):
        raise HarnessError(f"candidate failure point returned unexpected exit code {hub.last_exit_code}: {hub.logs}")
    return int(hub.last_exit_code)


def restore_and_verify_019(
    *,
    backup_dir: pathlib.Path,
    root: pathlib.Path,
    label: str,
    logs: pathlib.Path,
    owner_email: str,
    regular_email: str,
    password: str,
    owner_id: str,
    regular_id: str,
    system_id: str,
    owner_settings_id: str,
    regular_settings_id: str,
    info: dict[str, Any],
    seeded: dict[str, dict[str, Any]],
    public_key: str,
    key_fingerprint: str,
    expected_config_sha256: str,
    expected_hub_sha256: str,
    expected_agent_sha256: str,
    evidence_output_dir: pathlib.Path | None = None,
) -> dict[str, Any]:
    """Recover only from a fresh full archive extraction, then verify v0.19."""
    restore_parent = root / f"{label}-full-restore"
    extracted = restore_and_compare(backup_dir, restore_parent)
    checkpoint = restore_parent / "checkpoint"
    data_dir = checkpoint / "var/lib/beszel-hub"
    old_hub_binary = checkpoint / "usr/local/bin/beszel-hub"
    old_agent_binary = checkpoint / "usr/local/bin/beszel-agent"
    if file_sha256(old_hub_binary) != expected_hub_sha256 or file_sha256(old_agent_binary) != expected_agent_sha256:
        raise HarnessError(f"{label}: restored executable checksums differ from the original v0.19 set")
    if file_sha256(data_dir / "config.yml") != expected_config_sha256:
        raise HarnessError(f"{label}: restored Hub config checksum changed")
    before = database_snapshot(data_dir / "data.db")
    if before["integrity_check"] != ["ok"]:
        raise HarnessError(f"{label}: restored pre-start SQLite integrity failed: {before['integrity_check']}")
    if any(str(row[0]).startswith("179") or row[0] == "0_collections_snapshot_0_20_0.go" for row in before["migration_ledger"]):
        raise HarnessError(f"{label}: backup unexpectedly contains v0.20/v0.21 migration entries")

    hub = HubProcess(old_hub_binary, data_dir, logs / f"{label}-recovered-019.log")
    hub.start(env={"APP_URL": "https://p013.synthetic.invalid"})
    health, _, _ = request(hub.base, "GET", "/api/health")
    if health != 200:
        raise HarnessError(f"{label}: recovered v0.19 health check failed with HTTP {health}")
    invariants = verify_fixture_api_state(
        hub.base,
        owner_email=owner_email,
        password=password,
        owner_id=owner_id,
        regular_id=regular_id,
        system_id=system_id,
        owner_settings_id=owner_settings_id,
        regular_settings_id=regular_settings_id,
        info=info,
        seeded=seeded,
        public_key=public_key,
        key_fingerprint=key_fingerprint,
    )
    restored_schema = collection_schema(hub.base, auth_token(hub.base, "_superusers", owner_email, password))
    if "network_monitors" in restored_schema or "network_monitor_stats" in restored_schema:
        raise HarnessError(f"{label}: v0.19 recovery opened a migrated schema instead of a clean backup")
    hub.stop()
    after = database_snapshot(data_dir / "data.db")
    if after["integrity_check"] != ["ok"]:
        raise HarnessError(f"{label}: recovered v0.19 SQLite integrity failed: {after['integrity_check']}")
    for key in ("tables", "schema_sha256", "migration_ledger", "application_settings"):
        if after[key] != before[key]:
            raise HarnessError(f"{label}: v0.19 reopen changed pre-upgrade {key}")
    before_counts = {key: value for key, value in before["record_counts"].items() if not key.startswith("sqlite_")}
    after_counts = {key: value for key, value in after["record_counts"].items() if not key.startswith("sqlite_")}
    if after_counts != before_counts:
        raise HarnessError(f"{label}: v0.19 recovery changed persistent record counts")
    recovery_log = preserve_failure_log(hub.logs, evidence_output_dir, f"{label}-v019-recovered")
    return {
        "failure_case": label,
        "restore_inventory_matches": extracted["restore_inventory_matches"],
        "old_hub_binary_sha256": file_sha256(old_hub_binary),
        "old_agent_binary_sha256": file_sha256(old_agent_binary),
        "health_http": health,
        "sqlite_integrity_check": after["integrity_check"],
        "migration_ledger_exactly_pre_upgrade": after["migration_ledger"] == before["migration_ledger"],
        "schema_hash_pre_upgrade_preserved": after["schema_sha256"] == before["schema_sha256"],
        "record_counts_pre_upgrade_preserved": after_counts == before_counts,
        "api_invariants": invariants,
        "config_sha256_preserved": file_sha256(data_dir / "config.yml"),
        **recovery_log,
    }


def database_migration_names(database: pathlib.Path) -> list[str]:
    return [str(row[0]) for row in database_snapshot(database)["migration_ledger"]]


def sanitized_log_copy(source: pathlib.Path, destination: pathlib.Path) -> str:
    """Preserve useful synthetic diagnostics without publishing fixture secrets."""
    text = source.read_text(encoding="utf-8", errors="replace")
    text = text.replace(SYNTHETIC_PASSWORD, "<redacted synthetic credential>")
    text = text.replace(SYNTHETIC_TOKEN, "<redacted synthetic token>")
    destination.parent.mkdir(parents=True, exist_ok=True)
    destination.write_text(text, encoding="utf-8")
    return file_sha256(destination)


def preserve_failure_log(source: pathlib.Path, output_dir: pathlib.Path | None, label: str) -> dict[str, str]:
    if output_dir is None:
        return {"diagnostic_log": "available in the hosted workflow log"}
    destination = output_dir / "failure-logs" / f"{label}.log"
    return {
        "diagnostic_log": destination.relative_to(output_dir).as_posix(),
        "diagnostic_log_sha256": sanitized_log_copy(source, destination),
    }


class HealthFailureHandler(http.server.BaseHTTPRequestHandler):
    """A synthetic failed readiness probe used with an occupied service port."""

    def do_GET(self) -> None:  # noqa: N802 - stdlib handler contract
        self.send_response(503)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(b'{"status":"p013-injected-unhealthy"}')

    def log_message(self, *_: Any) -> None:
        return


def ssh_key_fingerprint(public_key: str, temp_dir: pathlib.Path) -> str:
    public_file = temp_dir / "synthetic-hub.pub"
    write_private(public_file, public_key.rstrip() + "\n")
    result = subprocess.run(["ssh-keygen", "-lf", str(public_file), "-E", "sha256"], check=True, text=True, capture_output=True)
    return result.stdout.split()[1]


def verify_first_user_bootstrap(binary: pathlib.Path, root: pathlib.Path, logs: pathlib.Path) -> dict[str, Any]:
    """Exercise failed/restarted and concurrent first-user setup on a fresh Hub."""
    import sqlite3

    data_dir = root / "bootstrap-021-data"
    data_dir.mkdir()
    bootstrap_env = {
        "APP_URL": "https://p013-bootstrap.synthetic.invalid",
        "BESZEL_HUB_USER_EMAIL": "",
        "BESZEL_HUB_USER_PASSWORD": "",
        "USER_EMAIL": "",
        "USER_PASSWORD": "",
    }
    hub = HubProcess(binary, data_dir, logs / "hub-021-first-user-bootstrap.log")
    hub.start(env=bootstrap_env)
    health, _, _ = request(hub.base, "GET", "/api/health")
    if health != 200:
        raise HarnessError(f"fresh v0.21 bootstrap Hub health failed: HTTP {health}")
    first_run_status, _, first_run = request(hub.base, "GET", "/api/beszel/first-run")
    if first_run_status != 200 or not isinstance(first_run, dict) or first_run.get("firstRun") is not True:
        raise HarnessError(f"fresh v0.21 Hub did not report first-run setup: HTTP {first_run_status}, {first_run}")
    hub.stop()

    failed_email = "p013-bootstrap-fail@example.invalid"
    connection = sqlite3.connect(data_dir / "data.db")
    try:
        connection.execute(
            f'''CREATE TRIGGER p013_fail_first_superuser
                BEFORE INSERT ON "_superusers"
                WHEN NEW."email" = '{failed_email}'
                BEGIN SELECT RAISE(ABORT, 'P013 synthetic partial bootstrap failure'); END'''
        )
        connection.commit()
    finally:
        connection.close()

    hub.start(env=bootstrap_env)
    failed_status, _, failed_body = request(
        hub.base,
        "POST",
        "/api/beszel/create-user",
        body={"email": failed_email, "password": SYNTHETIC_PASSWORD},
    )
    if failed_status != 500:
        raise HarnessError(f"injected first-user partial failure returned HTTP {failed_status}, expected 500")
    hub.stop()

    connection = sqlite3.connect(data_dir / "data.db")
    try:
        connection.execute("DROP TRIGGER p013_fail_first_superuser")
        connection.commit()
        user_count = int(connection.execute('SELECT COUNT(*) FROM "users"').fetchone()[0])
        superuser_emails = [str(row[0]) for row in connection.execute('SELECT "email" FROM "_superusers" ORDER BY "email"')]
    finally:
        connection.close()
    if user_count != 0 or superuser_emails != ["_@b.b"]:
        raise HarnessError(f"failed bootstrap transaction leaked users/admins: users={user_count}, superusers={superuser_emails}")

    # This restart is after a failure between regular-user and superuser saves.
    hub.start(env=bootstrap_env)
    restarted_count = int(sqlite3.connect(data_dir / "data.db").execute('SELECT COUNT(*) FROM "users"').fetchone()[0])
    if restarted_count != 0:
        raise HarnessError("first-user bootstrap failure survived as a partially created regular user")
    connection = sqlite3.connect(data_dir / "data.db")
    try:
        superuser_emails_after_restart = [str(row[0]) for row in connection.execute('SELECT "email" FROM "_superusers" ORDER BY "email"')]
    finally:
        connection.close()
    if superuser_emails_after_restart != ["_@b.b"]:
        raise HarnessError(f"restart duplicated or removed the temporary superuser: {superuser_emails_after_restart}")

    candidates = ["p013-bootstrap-one@example.invalid", "p013-bootstrap-two@example.invalid"]
    with ThreadPoolExecutor(max_workers=2) as executor:
        responses = list(executor.map(
            lambda email: request(
                hub.base,
                "POST",
                "/api/beszel/create-user",
                body={"email": email, "password": SYNTHETIC_PASSWORD},
            ),
            candidates,
        ))
    status_by_email = {email: response[0] for email, response in zip(candidates, responses, strict=True)}
    if sorted(status_by_email.values()) != [200, 403]:
        raise HarnessError(f"concurrent first-user requests did not produce exactly one owner: {status_by_email}")
    owner_email = next(email for email, status in status_by_email.items() if status == 200)
    owner_token = auth_token(hub.base, "_superusers", owner_email, SYNTHETIC_PASSWORD)
    users = list_records(hub.base, owner_token, "users")
    superusers = list_records(hub.base, owner_token, "_superusers")
    if len(users) != 1 or users[0].get("email") != owner_email or users[0].get("role") != "admin":
        raise HarnessError(f"concurrent bootstrap produced unexpected users/roles: {users}")
    if len(superusers) != 1 or superusers[0].get("email") != owner_email:
        raise HarnessError(f"concurrent bootstrap left a duplicate/temporary superuser: {superusers}")
    first_run_after, _, state_after = request(hub.base, "GET", "/api/beszel/first-run")
    if first_run_after != 200 or not isinstance(state_after, dict) or state_after.get("firstRun") is not False:
        raise HarnessError("first-run state remained enabled after the sole owner was created")
    hub.stop()
    return {
        "fresh_hub_health_http": health,
        "first_run_before_setup": first_run["firstRun"],
        "partial_user_creation_failure_http": failed_status,
        "partial_user_creation_failure_body": failed_body,
        "users_after_failure": user_count,
        "temporary_superuser_after_failure": superuser_emails,
        "users_after_restart": restarted_count,
        "superusers_after_restart": superuser_emails_after_restart,
        "concurrent_first_user_http_by_email": status_by_email,
        "successful_owner_email": owner_email,
        "final_user_count": len(users),
        "final_admin_role": users[0].get("role"),
        "final_superuser_count": len(superusers),
        "temporary_superuser_removed": "_@b.b" not in {record.get("email") for record in superusers},
        "first_run_after_setup": state_after["firstRun"],
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=pathlib.Path, help="copy the generated evidence directory here")
    parser.add_argument("--old-rev", default=OLD_019_REV)
    args = parser.parse_args()
    if args.output:
        args.output = args.output.resolve()
        if args.output == PROTECTED_DATA_DIR or PROTECTED_DATA_DIR in args.output.parents:
            raise HarnessError("refusing to write evidence beneath /var/lib/beszel-hub")
        args.output.mkdir(parents=True, exist_ok=True)

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

        bootstrap_evidence = verify_first_user_bootstrap(new_binary, root, logs)

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
        hub_port = free_port()
        hub_service_plist = f'''<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>dev.beszel.hub</string>
<key>ProgramArguments</key><array><string>/usr/local/bin/beszel-hub</string><string>serve</string><string>--http</string><string>0.0.0.0:{hub_port}</string><string>--dir</string><string>/var/lib/beszel-hub</string></array>
<key>EnvironmentVariables</key><dict><key>PATH</key><string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string></dict>
<key>WorkingDirectory</key><string>/var/lib/beszel-hub</string>
<key>RunAtLoad</key><true/><key>KeepAlive</key><true/><key>ThrottleInterval</key><integer>10</integer>
<key>StandardOutPath</key><string>/var/log/beszel-hub.log</string>
<key>StandardErrorPath</key><string>/var/log/beszel-hub.err.log</string>
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
<key>StandardOutPath</key><string>/var/log/beszel-agent.log</string>
<key>StandardErrorPath</key><string>/var/log/beszel-agent.err.log</string>
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

        # Run the integrated v0.21 Hub against a fresh clone extracted from
        # the verified cold 0.19 archive. The original 0.19 fixture and the
        # v0.19 restore validation copy remain untouched by migration.
        migration_backup_clone = root / "migration-backup-clone"
        restore_and_compare(backup_dir, migration_backup_clone)
        migration_data_dir = root / "migration-021-data"
        shutil.copytree(
            migration_backup_clone / "checkpoint/var/lib/beszel-hub",
            migration_data_dir,
            symlinks=True,
        )
        migration_hub = HubProcess(new_binary, migration_data_dir, logs / "hub-021-migration.log")
        migration_hub.start(env={"APP_URL": "https://p013.synthetic.invalid"})
        migration_health, _, _ = request(migration_hub.base, "GET", "/api/health")
        if migration_health != 200:
            raise HarnessError(f"migrated v0.21 Hub health check failed: HTTP {migration_health}")
        migration_schema = collection_schema(
            migration_hub.base,
            auth_token(migration_hub.base, "_superusers", owner_email, SYNTHETIC_PASSWORD),
        )
        schema_validation = validate_v021_schema(schema, migration_schema)
        migration_api_state = verify_fixture_api_state(
            migration_hub.base,
            owner_email=owner_email,
            password=SYNTHETIC_PASSWORD,
            owner_id=owner_id,
            regular_id=regular_id,
            system_id=system_id,
            owner_settings_id=str(owner_settings["id"]),
            regular_settings_id=str(regular_settings["id"]),
            info=info,
            seeded=seeded,
            public_key=public_key,
            key_fingerprint=key_fingerprint,
        )
        if file_sha256(migration_data_dir / "config.yml") != file_sha256(config_file):
            raise HarnessError("v0.21 startup changed the original synthetic Hub configuration")
        migration_hub.stop()
        migrated_first = database_snapshot(migration_data_dir / "data.db")
        if migrated_first["integrity_check"] != ["ok"]:
            raise HarnessError(f"migrated v0.21 database integrity failed: {migrated_first['integrity_check']}")
        previous_ledger = [str(row[0]) for row in database_snapshot(data_dir / "data.db")["migration_ledger"]]
        migrated_ledger = [str(row[0]) for row in migrated_first["migration_ledger"]]
        if len(migrated_ledger) != len(set(migrated_ledger)):
            raise HarnessError("v0.21 migration ledger contains duplicate migration entries")
        if not set(previous_ledger).issubset(set(migrated_ledger)):
            raise HarnessError("v0.21 migration removed previously applied v0.19 ledger entries")
        required_migrations = [
            "0_collections_snapshot_0_20_0.go",
            "1790193183_network_monitor_cert.go",
            "1790273174_network_monitor_server.go",
            "1790924727_zfs_pool_status.go",
            "1790964708_network_monitor_skip_tls_verify.go",
        ]
        missing_migrations = [name for name in required_migrations if migrated_ledger.count(name) != 1]
        if missing_migrations:
            raise HarnessError(f"v0.21 migrations missing or repeated: {missing_migrations}")
        migration_positions = [migrated_ledger.index(name) for name in required_migrations]
        if migration_positions != sorted(migration_positions):
            raise HarnessError(f"v0.21 schema migration order is unexpected: {migration_positions}")

        # A second full startup must not add ledger entries, mutate schema, or
        # change record counts. API invariants are checked again after restart.
        migration_hub.start(env={"APP_URL": "https://p013.synthetic.invalid"})
        second_health, _, _ = request(migration_hub.base, "GET", "/api/health")
        if second_health != 200:
            raise HarnessError(f"second v0.21 startup health check failed: HTTP {second_health}")
        second_schema = collection_schema(
            migration_hub.base,
            auth_token(migration_hub.base, "_superusers", owner_email, SYNTHETIC_PASSWORD),
        )
        validate_v021_schema(schema, second_schema)
        second_api_state = verify_fixture_api_state(
            migration_hub.base,
            owner_email=owner_email,
            password=SYNTHETIC_PASSWORD,
            owner_id=owner_id,
            regular_id=regular_id,
            system_id=system_id,
            owner_settings_id=str(owner_settings["id"]),
            regular_settings_id=str(regular_settings["id"]),
            info=info,
            seeded=seeded,
            public_key=public_key,
            key_fingerprint=key_fingerprint,
        )
        migration_hub.stop()
        migrated_second = database_snapshot(migration_data_dir / "data.db")
        if migrated_second["integrity_check"] != ["ok"]:
            raise HarnessError(f"second v0.21 database integrity failed: {migrated_second['integrity_check']}")
        for key in ("schema_sha256", "record_counts", "migration_ledger", "application_settings"):
            if migrated_second[key] != migrated_first[key]:
                raise HarnessError(f"second v0.21 startup changed database {key}")
        if migration_api_state != second_api_state:
            raise HarnessError("second v0.21 startup changed preserved API identities or relations")

        # The separate v0.21 process reopens the migrated data directory, runs
        # the registered migrations immediately before the production record
        # managers, then exercises rollup and retention on disposable rows.
        rollup_probe = subprocess.run(
            [go, "run", "./tests/hub-migration/rollup_probe.go", "--data-dir", str(migration_data_dir)],
            cwd=repo,
            check=False,
            capture_output=True,
            text=True,
            timeout=180,
        )
        if rollup_probe.returncode != 0:
            raise HarnessError(
                "post-migration rollup/retention probe failed:\n"
                + rollup_probe.stdout[-4000:]
                + "\n"
                + rollup_probe.stderr[-4000:]
            )
        try:
            rollup_evidence = json.loads(rollup_probe.stdout.strip().splitlines()[-1])
        except (IndexError, json.JSONDecodeError) as exc:
            raise HarnessError(f"rollup probe emitted invalid JSON: {rollup_probe.stdout[-2000:]}") from exc

        migration_hub.start(env={"APP_URL": "https://p013.synthetic.invalid"})
        functional_health, _, _ = request(migration_hub.base, "GET", "/api/health")
        if functional_health != 200:
            raise HarnessError(f"functional API startup health check failed: HTTP {functional_health}")
        functional_api_evidence = verify_migrated_monitor_alert_api(
            migration_hub.base,
            owner_email=owner_email,
            regular_email=regular_email,
            password=SYNTHETIC_PASSWORD,
            owner_id=owner_id,
            regular_id=regular_id,
            system_id=system_id,
        )
        migration_hub.stop()

        # Newly-created records must survive a service restart before they are
        # deleted through the same user-facing API.
        migration_hub.start(env={"APP_URL": "https://p013.synthetic.invalid"})
        functional_restart_health, _, _ = request(migration_hub.base, "GET", "/api/health")
        if functional_restart_health != 200:
            raise HarnessError(f"functional API restart health check failed: HTTP {functional_restart_health}")
        functional_restart_evidence = verify_and_delete_migrated_monitor_alert_api(
            migration_hub.base,
            regular_email=regular_email,
            password=SYNTHETIC_PASSWORD,
            expected=functional_api_evidence,
        )
        migration_hub.stop()
        after_api_cleanup = database_snapshot(migration_data_dir / "data.db")
        if after_api_cleanup["integrity_check"] != ["ok"]:
            raise HarnessError(f"post-API migration SQLite integrity failed: {after_api_cleanup['integrity_check']}")
        for key in ("schema_sha256", "migration_ledger"):
            if after_api_cleanup[key] != migrated_second[key]:
                raise HarnessError(f"monitor/alert API exercise changed migrated database {key}")
        cleaned_counts = {key: value for key, value in after_api_cleanup["record_counts"].items() if not key.startswith("sqlite_")}
        baseline_counts = {key: value for key, value in migrated_second["record_counts"].items() if not key.startswith("sqlite_")}
        if cleaned_counts != baseline_counts:
            raise HarnessError("monitor/alert API cleanup did not restore original record counts")

        recovery_kwargs = {
            "backup_dir": backup_dir,
            "root": root,
            "logs": logs,
            "owner_email": owner_email,
            "regular_email": regular_email,
            "password": SYNTHETIC_PASSWORD,
            "owner_id": owner_id,
            "regular_id": regular_id,
            "system_id": system_id,
            "owner_settings_id": str(owner_settings["id"]),
            "regular_settings_id": str(regular_settings["id"]),
            "info": info,
            "seeded": seeded,
            "public_key": public_key,
            "key_fingerprint": key_fingerprint,
            "expected_config_sha256": file_sha256(config_file),
            "expected_hub_sha256": file_sha256(old_binary),
            "expected_agent_sha256": file_sha256(old_agent_binary),
            "evidence_output_dir": args.output,
        }

        # Roll back a database that has completed 0.21 migration and restart.
        # Recovery always extracts a fresh complete archive, including the old
        # binaries and service/install state, into a separate destination.
        migration_state_sha256 = file_sha256(migration_data_dir / "data.db")
        rollback_evidence = restore_and_verify_019(label="post-migration-rollback", **recovery_kwargs)
        rollback_evidence["migrated_candidate_database_sha256_before_quarantine"] = migration_state_sha256
        rollback_evidence["verified_backup_archive_sha256"] = backup_evidence["archive_sha256"]

        interrupted_evidence: list[dict[str, Any]] = []

        # Fail before the Hub executable is invoked. The original database
        # remains untouched, but recovery still uses a separate full restore.
        before_data = clone_backup_data(backup_dir, root, "fail-before-migration")
        before_snapshot = database_snapshot(before_data / "data.db")
        prestart_wrapper = root / "fail-before-migration-wrapper"
        prestart_wrapper.write_text("#!/bin/sh\necho P013_FAILPOINT_BEFORE_MIGRATION >&2\nexit 86\n", encoding="utf-8")
        prestart_wrapper.chmod(0o755)
        before_hub = HubProcess(prestart_wrapper, before_data, logs / "fail-before-migration-candidate.log")
        before_hub.start(wait=False)
        before_exit = wait_for_candidate_failure(before_hub)
        before_log_evidence = preserve_failure_log(before_hub.logs, args.output, "before-migration")
        if before_exit != 86 or "P013_FAILPOINT_BEFORE_MIGRATION" not in tail(before_hub.logs):
            raise HarnessError("before-migration failure injection did not stop before invoking the candidate")
        after_before_failure = database_snapshot(before_data / "data.db")
        if after_before_failure != before_snapshot:
            raise HarnessError("before-migration failure injection modified the 0.19 database")
        before_result = {
            "failure_point": "before migration starts (candidate launcher exits before binary invocation)",
            "candidate_exit_code": before_exit,
            "candidate_database_unchanged": True,
            **before_log_evidence,
            "recovery": restore_and_verify_019(label="recovery-before-migration", **recovery_kwargs),
        }
        interrupted_evidence.append(before_result)

        # Stop the named migration ledger insertion after earlier schema
        # migrations have run. This leaves a real partial 0.21 database; no
        # v0.19 process is ever pointed at that candidate directory.
        during_data = clone_backup_data(backup_dir, root, "fail-during-migration")
        failing_migration = "1790273174_network_monitor_server.go"
        inject_migration_ledger_failure(during_data / "data.db", failing_migration)
        during_hub = HubProcess(new_binary, during_data, logs / "fail-during-migration-candidate.log")
        during_hub.start(env={"APP_URL": "https://p013.synthetic.invalid"}, wait=False)
        during_exit = wait_for_candidate_failure(during_hub)
        during_log_evidence = preserve_failure_log(during_hub.logs, args.output, "during-migration")
        during_log = tail(during_hub.logs)
        if "P013 deterministic migration-sequence failpoint" not in during_log:
            raise HarnessError(f"candidate did not reach the named migration interruption point:\n{during_log}")
        during_snapshot = database_snapshot(during_data / "data.db")
        during_names = [str(row[0]) for row in during_snapshot["migration_ledger"]]
        if failing_migration in during_names or during_snapshot["integrity_check"] != ["ok"]:
            raise HarnessError("mid-migration failpoint did not leave the expected incomplete, inspectable candidate")
        during_result = {
            "failure_point": f"during schema sequence at {failing_migration}",
            "candidate_exit_code": during_exit,
            "partial_candidate_database_sha256": file_sha256(during_data / "data.db"),
            "candidate_integrity_check": during_snapshot["integrity_check"],
            "migration_ledger_after_failure": during_names,
            "failing_migration_absent_from_ledger": failing_migration not in during_names,
            "partial_candidate_opened_by_v019": False,
            **during_log_evidence,
            "recovery": restore_and_verify_019(label="recovery-during-migration", **recovery_kwargs),
        }
        interrupted_evidence.append(during_result)

        # The final migration has completed before the OnServe initializer
        # updates alert rules. Abort that update only after the new migration
        # is recorded, leaving complete schema work but no normal service.
        post_schema_data = clone_backup_data(backup_dir, root, "fail-after-schema")
        inject_post_schema_startup_failure(post_schema_data / "data.db")
        post_schema_hub = HubProcess(new_binary, post_schema_data, logs / "fail-after-schema-candidate.log")
        post_schema_hub.start(env={"APP_URL": "https://p013.synthetic.invalid"}, wait=False)
        post_schema_exit = wait_for_candidate_failure(post_schema_hub)
        post_schema_log_evidence = preserve_failure_log(post_schema_hub.logs, args.output, "after-schema-before-health")
        post_schema_log = tail(post_schema_hub.logs)
        if "P013 deterministic post-schema startup failpoint" not in post_schema_log:
            raise HarnessError(f"candidate did not reach the post-schema startup failpoint:\n{post_schema_log}")
        post_schema_snapshot = database_snapshot(post_schema_data / "data.db")
        post_schema_names = [str(row[0]) for row in post_schema_snapshot["migration_ledger"]]
        if any(post_schema_names.count(name) != 1 for name in required_migrations):
            raise HarnessError("post-schema failpoint occurred before all schema migrations were applied once")
        if post_schema_snapshot["integrity_check"] != ["ok"]:
            raise HarnessError("post-schema failed startup left an SQLite integrity error")
        post_schema_result = {
            "failure_point": "after schema migration sequence, during Hub startup before health",
            "candidate_exit_code": post_schema_exit,
            "migration_ledger_after_failure": post_schema_names,
            "all_required_migrations_applied_once": True,
            "candidate_integrity_check": post_schema_snapshot["integrity_check"],
            "partial_candidate_opened_by_v019": False,
            **post_schema_log_evidence,
            "recovery": restore_and_verify_019(label="recovery-after-schema", **recovery_kwargs),
        }
        interrupted_evidence.append(post_schema_result)

        # Simulate a failed readiness probe after migrations, caused by a
        # reserved service port returning 503. The candidate must not be
        # treated as healthy, and recovery still replaces data and binaries.
        failed_health_data = clone_backup_data(backup_dir, root, "fail-before-health")
        failed_health_port = free_port()
        failed_health_server = http.server.ThreadingHTTPServer(("127.0.0.1", failed_health_port), HealthFailureHandler)
        failed_health_thread = threading.Thread(target=failed_health_server.serve_forever, daemon=True)
        failed_health_thread.start()
        failed_health_hub = HubProcess(new_binary, failed_health_data, logs / "fail-before-health-candidate.log")
        try:
            failed_health_hub.start(env={"APP_URL": "https://p013.synthetic.invalid"}, wait=False, port=failed_health_port)
            failed_probe_status, _, _ = request(failed_health_hub.base, "GET", "/api/health")
            if failed_probe_status != 503:
                raise HarnessError(f"health failure stub returned HTTP {failed_probe_status}, expected 503")
            failed_health_exit = wait_for_candidate_failure(failed_health_hub)
        finally:
            failed_health_server.shutdown()
            failed_health_server.server_close()
            failed_health_thread.join(timeout=5)
        failed_health_log_evidence = preserve_failure_log(failed_health_hub.logs, args.output, "failed-health-before-first-pass")
        failed_health_snapshot = database_snapshot(failed_health_data / "data.db")
        failed_health_names = [str(row[0]) for row in failed_health_snapshot["migration_ledger"]]
        if any(failed_health_names.count(name) != 1 for name in required_migrations):
            raise HarnessError("failed readiness startup did not finish the schema sequence exactly once")
        failed_health_result = {
            "failure_point": "candidate startup after schema; readiness endpoint returns injected HTTP 503 before any candidate health pass",
            "health_probe_http": failed_probe_status,
            "candidate_exit_code": failed_health_exit,
            "migration_ledger_after_failure": failed_health_names,
            "candidate_integrity_check": failed_health_snapshot["integrity_check"],
            "partial_candidate_opened_by_v019": False,
            **failed_health_log_evidence,
            "recovery": restore_and_verify_019(label="recovery-after-failed-health", **recovery_kwargs),
        }
        interrupted_evidence.append(failed_health_result)

        # A controlled v0.21 service restart that cannot bind is treated as a
        # failed upgrade health check. This begins from a real healthy migrated
        # run, then recovers from the complete 0.19 archive set.
        restart_data = clone_backup_data(backup_dir, root, "fail-controlled-restart")
        restart_hub = HubProcess(new_binary, restart_data, logs / "controlled-restart-candidate.log")
        restart_hub.start(env={"APP_URL": "https://p013.synthetic.invalid"})
        restart_initial_health, _, _ = request(restart_hub.base, "GET", "/api/health")
        if restart_initial_health != 200:
            raise HarnessError(f"controlled restart setup was not healthy: HTTP {restart_initial_health}")
        restart_hub.stop()
        restart_port = free_port()
        restart_server = http.server.ThreadingHTTPServer(("127.0.0.1", restart_port), HealthFailureHandler)
        restart_thread = threading.Thread(target=restart_server.serve_forever, daemon=True)
        restart_thread.start()
        try:
            restart_hub.start(env={"APP_URL": "https://p013.synthetic.invalid"}, wait=False, port=restart_port)
            restart_probe_status, _, _ = request(restart_hub.base, "GET", "/api/health")
            if restart_probe_status != 503:
                raise HarnessError(f"controlled restart failure stub returned HTTP {restart_probe_status}, expected 503")
            restart_exit = wait_for_candidate_failure(restart_hub)
        finally:
            restart_server.shutdown()
            restart_server.server_close()
            restart_thread.join(timeout=5)
        restart_log_evidence = preserve_failure_log(restart_hub.logs, args.output, "controlled-restart-health-failure")
        restart_snapshot = database_snapshot(restart_data / "data.db")
        restart_names = [str(row[0]) for row in restart_snapshot["migration_ledger"]]
        if any(restart_names.count(name) != 1 for name in required_migrations):
            raise HarnessError("controlled restart changed the migrated schema ledger unexpectedly")
        restart_result = {
            "failure_point": "controlled service restart after a prior healthy v0.21 start",
            "initial_health_http": restart_initial_health,
            "restart_health_probe_http": restart_probe_status,
            "restart_candidate_exit_code": restart_exit,
            "migration_ledger_after_failure": restart_names,
            "candidate_integrity_check": restart_snapshot["integrity_check"],
            "partial_candidate_opened_by_v019": False,
            **restart_log_evidence,
            "recovery": restore_and_verify_019(label="recovery-controlled-restart", **recovery_kwargs),
        }
        interrupted_evidence.append(restart_result)

        interrupted_migration_evidence = {
            "status": "PASS — all injected disposable failure checkpoints recovered by a verified complete v0.19 restore",
            "cases": interrupted_evidence,
        }
        migration_evidence = {
            "source_backup_sha256": backup_evidence["archive_sha256"],
            "target_version": new_version,
            "target_pocketbase_version": new_pocketbase,
            "first_health_http": migration_health,
            "second_health_http": second_health,
            "sqlite_integrity_first": migrated_first["integrity_check"],
            "sqlite_integrity_second": migrated_second["integrity_check"],
            "migration_ledger_before": previous_ledger,
            "migration_ledger_after": migrated_ledger,
            "required_migrations_ordered_once": required_migrations,
            "schema_validation": schema_validation,
            "collection_schema_after": migration_schema,
            "api_invariants_first": migration_api_state,
            "api_invariants_second": second_api_state,
            "config_sha256_preserved": file_sha256(migration_data_dir / "config.yml"),
            "schema_hash_second_start_stable": migrated_second["schema_sha256"] == migrated_first["schema_sha256"],
            "record_counts_second_start_stable": migrated_second["record_counts"] == migrated_first["record_counts"],
            "rollup_and_retention_after_migration": rollup_evidence,
            "functional_api_health_http": functional_health,
            "functional_api": functional_api_evidence,
            "functional_api_restart_health_http": functional_restart_health,
            "functional_api_restart": functional_restart_evidence,
            "post_api_cleanup_record_counts_match": cleaned_counts == baseline_counts,
        }

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
            "first_user_bootstrap_021": bootstrap_evidence,
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
            "migration_021": migration_evidence,
            "rollback_019": rollback_evidence,
            "interrupted_migration": interrupted_migration_evidence,
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
