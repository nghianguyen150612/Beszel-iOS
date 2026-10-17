#!/usr/bin/env python3
"""Seed synthetic route/browser records into a disposable local Hub."""

from __future__ import annotations

import argparse
import json
import os
import sys
import time
from urllib.error import HTTPError, URLError
from urllib.parse import urlsplit
from urllib.request import Request, urlopen


class FixtureError(RuntimeError):
    pass


class HubAPI:
    def __init__(self, base_url: str):
        self.base_url = base_url.rstrip("/")

    def request(self, path: str, token: str | None = None, body: dict | None = None, method: str = "GET"):
        headers = {"Content-Type": "application/json"}
        if token:
            headers["Authorization"] = token
        request = Request(
            self.base_url + path,
            data=json.dumps(body).encode("utf-8") if body is not None else None,
            headers=headers,
            method=method,
        )
        try:
            with urlopen(request, timeout=15) as response:
                raw = response.read()
                return response.status, json.loads(raw) if raw else None
        except HTTPError as error:
            raw = error.read()
            try:
                payload = json.loads(raw)
            except json.JSONDecodeError:
                payload = raw.decode("utf-8", errors="replace")
            return error.code, payload
        except (OSError, URLError) as error:
            raise FixtureError(f"request failed for {path}: {error}") from error

    def checked(self, path: str, token: str | None = None, body: dict | None = None, method: str = "GET"):
        status, payload = self.request(path, token, body, method)
        if status not in (200, 201):
            raise FixtureError(f"{method} {path} returned HTTP {status}: {payload}")
        return payload

    def auth(self, collection: str, email: str, password: str) -> tuple[str, dict]:
        payload = self.checked(
            f"/api/collections/{collection}/auth-with-password",
            body={"identity": email, "password": password},
            method="POST",
        )
        return payload["token"], payload["record"]

    def list_records(self, token: str, collection: str) -> list[dict]:
        return self.checked(f"/api/collections/{collection}/records?perPage=500", token)["items"]

    def ensure_record(self, token: str, collection: str, identity: dict, fields: dict) -> dict:
        for record in self.list_records(token, collection):
            if all(record.get(key) == value for key, value in identity.items()):
                return self.checked(
                    f"/api/collections/{collection}/records/{record['id']}", token, fields, "PATCH"
                )
        return self.checked(f"/api/collections/{collection}/records", token, fields, "POST")

    def replace_synthetic_stats(self, token: str, collection: str, system_id: str, records: list[dict]) -> None:
        # PocketBase's created field is server managed, so patching old fixture
        # rows leaves them outside the dashboard's default one-hour chart range.
        for existing in self.list_records(token, collection):
            if existing.get("system") == system_id:
                status, payload = self.request(
                    f"/api/collections/{collection}/records/{existing['id']}", token, method="DELETE"
                )
                if status not in (200, 204):
                    raise FixtureError(f"DELETE synthetic {collection} row returned HTTP {status}: {payload}")
        for record in records:
            self.checked(f"/api/collections/{collection}/records", token, record, "POST")


def seed(api: HubAPI, admin_email: str, outsider_email: str, password: str) -> dict:
    super_token, _ = api.auth("_superusers", admin_email, password)
    users = api.list_records(super_token, "users")
    accounts = {record.get("email"): record for record in users}
    if admin_email not in accounts:
        raise FixtureError("the synthetic administrator account must be created through the Hub first-run UI")
    admin = accounts[admin_email]
    outsider = accounts.get(outsider_email)
    if outsider is None:
        outsider = api.checked(
            "/api/collections/users/records",
            super_token,
            {
                "email": outsider_email,
                "password": password,
                "passwordConfirm": password,
                "verified": True,
                "role": "user",
            },
            "POST",
        )

    for user, email in ((admin, admin_email), (outsider, outsider_email)):
        api.ensure_record(
            super_token,
            "user_settings",
            {"user": user["id"]},
            {"user": user["id"], "settings": {"chartTime": "1h", "lang": "en", "emails": [email], "webhooks": []}},
        )

    info = {
        "h": "p015-synthetic-host",
        "k": "6.8.0-p015-synthetic",
        "cpu": 27.5,
        "t": 16,
        "c": 8,
        "m": "Synthetic 8 core CPU",
        "la": [1.25, 1.1, 0.95],
        "o": "Linux",
        "u": 86400,
        "mp": 50,
        "dp": 40,
        "b": 0,
        "v": "0.21.0",
        "os": 0,
    }
    system = api.ensure_record(
        super_token,
        "systems",
        {"name": "p015-synthetic-system"},
        {
            "name": "p015-synthetic-system",
            "host": "127.0.0.1",
            "port": "45876",
            "users": [admin["id"]],
            "status": "down",
            "info": info,
        },
    )
    outsider_system = api.ensure_record(
        super_token,
        "systems",
        {"name": "p015-outsider-system"},
        {
            "name": "p015-outsider-system",
            "host": "127.0.0.1",
            "port": "45877",
            "users": [outsider["id"]],
            "status": "down",
            "info": {**info, "h": "p015-outsider-host"},
        },
    )

    system_id = system["id"]
    now_ms = int(time.time() * 1000)
    details = {
        "system": system_id,
        "hostname": "p015-synthetic-host",
        "os": 0,
        "os_name": "Synthetic Linux",
        "kernel": "6.8.0-p015-synthetic",
        "cpu": "Synthetic 8 core CPU",
        "arch": "amd64",
        "cores": 8,
        "threads": 16,
        "memory": 17179869184,
        "podman": False,
    }
    api.ensure_record(super_token, "system_details", {"system": system_id}, details)

    system_stats = [
        {"cpu": 0, "m": 8, "mu": 0, "mp": 0, "mb": 0, "s": 2, "su": 0, "d": 100, "du": 0, "dp": 0, "dr": 0, "dw": 0, "ns": 0, "nr": 0, "la": [0, 0, 0]},
        {"cpu": 27.5, "m": 8, "mu": 4, "mp": 50, "mb": 1, "s": 2, "su": 0.5, "d": 100, "du": 40, "dp": 40, "dr": 1, "dw": 2, "ns": 0.1, "nr": 0.2, "b": [102400, 204800], "la": [1.25, 1.1, 0.95]},
        {"cpu": 19.5, "m": 8, "mu": 3.5, "mp": 43.75, "mb": 1, "s": 2, "su": 0.4, "d": 100, "du": 39, "dp": 39, "dr": 0.5, "dw": 1.2, "ns": 0.3, "nr": 0.4, "la": [0.75, 0.8, 0.7]},
    ]
    api.replace_synthetic_stats(
        super_token,
        "system_stats",
        system_id,
        [{"system": system_id, "type": "1m", "stats": stats} for stats in system_stats],
    )

    api.ensure_record(
        super_token,
        "containers",
        {"system": system_id, "name": "p015-synthetic-container"},
        {
            "system": system_id,
            "name": "p015-synthetic-container",
            "status": "running",
            "health": 1,
            "cpu": 4.25,
            "memory": 256,
            "net": 32,
            "image": "registry.synthetic.invalid/p015:only",
            "ports": "8080/tcp",
            "updated": now_ms,
            "updatable": False,
        },
    )
    api.replace_synthetic_stats(
        super_token,
        "container_stats",
        system_id,
        [{
            "system": system_id,
            "type": "1m",
            "stats": [{"n": "p015-synthetic-container", "c": 4.25, "m": 256, "ns": 0.02, "nr": 0.01}],
        }],
    )
    api.ensure_record(
        super_token,
        "smart_devices",
        {"system": system_id, "name": "p015-synthetic-disk"},
        {
            "system": system_id,
            "name": "p015-synthetic-disk",
            "model": "Synthetic NVMe",
            "state": "PASSED",
            "capacity": 1000000000000,
            "temp": 31,
            "firmware": "P015",
            "serial": "P015-SYNTHETIC",
            "type": "NVMe",
            "hours": 42,
            "cycles": 7,
            "attributes": {"synthetic": True},
        },
    )
    api.ensure_record(
        super_token,
        "systemd_services",
        {"system": system_id, "name": "p015-synthetic.service"},
        {"system": system_id, "name": "p015-synthetic.service", "state": 0, "sub": 1, "cpu": 0.25, "memory": 1, "updated": now_ms},
    )

    monitor = api.ensure_record(
        super_token,
        "network_monitors",
        {"system": system_id, "target": "https://monitor.synthetic.invalid/health"},
        {
            "system": system_id,
            "target": "https://monitor.synthetic.invalid/health",
            "protocol": "http",
            "port": 443,
            "interval": 60,
            "enabled": True,
            "server": "p015-synthetic-monitor",
            "skipTlsVerify": False,
            "certInfo": {"issuer": "P015 Synthetic CA", "expires": 1800000000000},
            "res": 24,
            "resAvg1h": 27,
            "resMin1h": 20,
            "resMax1h": 31,
            "loss1h": 0,
        },
    )
    api.ensure_record(
        super_token,
        "network_monitor_stats",
        {"system": system_id, "monitor": monitor["id"], "type": "1m"},
        {
            "system": system_id,
            "monitor": monitor["id"],
            "type": "1m",
            "created": now_ms,
            "total_count": 10,
            "success_count": 9,
            "res_sum": 240,
            "res_min": 20,
            "res_max": 40,
        },
    )

    alert = api.ensure_record(
        super_token,
        "alerts",
        {"system": system_id, "user": admin["id"], "name": "CPU"},
        {
            "user": admin["id"],
            "system": system_id,
            "name": "CPU",
            "value": 55,
            "min": 50,
            "triggered": True,
            "pending_since": "2026-10-17T00:00:00Z",
            "state": {"synthetic": True},
        },
    )
    api.ensure_record(
        super_token,
        "alerts_history",
        {"system": system_id, "alert_id": alert["id"]},
        {"user": admin["id"], "system": system_id, "alert_id": alert["id"], "name": "CPU", "value": 55},
    )

    return {
        "status": "PASS",
        "hub_url": api.base_url,
        "admin_email": admin_email,
        "outsider_email": outsider_email,
        "system_id": system_id,
        "outsider_system_id": outsider_system["id"],
        "container_id": "p015-synthetic-container",
        "monitor_id": monitor["id"],
        "alert_id": alert["id"],
        "records": ["system details", "zero and populated system stats", "container", "container stats", "SMART device", "systemd service", "monitor and monitor history", "active alert and alert history", "user settings"],
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--hub-url", default="http://127.0.0.1:8097")
    args = parser.parse_args()
    parsed = urlsplit(args.hub_url)
    if parsed.scheme != "http" or parsed.hostname not in {"127.0.0.1", "localhost", "::1"} or parsed.path not in {"", "/"}:
        print("fixture seeding is restricted to a local HTTP Hub origin", file=sys.stderr)
        return 2
    admin_email = os.environ.get("P015_ADMIN_EMAIL", "p015-admin@example.invalid")
    outsider_email = os.environ.get("P015_OUTSIDER_EMAIL", "p015-outsider@example.invalid")
    password = os.environ.get("P015_SYNTHETIC_PASSWORD")
    if not password:
        print("P015_SYNTHETIC_PASSWORD is required", file=sys.stderr)
        return 2
    try:
        print(json.dumps(seed(HubAPI(args.hub_url), admin_email, outsider_email, password), indent=2))
    except (FixtureError, KeyError, TypeError) as error:
        print(f"P015 fixture seeding failed: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
