#!/usr/bin/env python3
"""Validate a production frontend tree and optionally compare its SHA-256 manifest."""

from __future__ import annotations

import argparse
import hashlib
import html.parser
import json
import mimetypes
import posixpath
import re
import sys
from pathlib import Path, PurePosixPath
from urllib.parse import unquote, urlsplit


class AssetValidationError(ValueError):
    """Raised when the generated frontend is incomplete or inconsistent."""


class _HTMLReferences(html.parser.HTMLParser):
    def __init__(self) -> None:
        super().__init__(convert_charrefs=True)
        self.references: list[str] = []

    def handle_starttag(self, _tag: str, attrs: list[tuple[str, str | None]]) -> None:
        for name, value in attrs:
            if name.lower() in {"src", "href"} and value:
                self.references.append(value)


def _local_reference(reference: str, parent: str, files: dict[str, bytes], source: str) -> str | None:
    parsed = urlsplit(reference)
    if parsed.scheme or parsed.netloc or not parsed.path:
        return None

    path = unquote(parsed.path)
    if path.startswith("/"):
        raise AssetValidationError(f"{source} uses a non-relative asset path: {reference}")

    normalized = posixpath.normpath(posixpath.join(parent, path))
    if normalized == ".." or normalized.startswith("../") or PurePosixPath(normalized).is_absolute():
        raise AssetValidationError(f"{source} references an asset outside dist: {reference}")
    if normalized not in files:
        raise AssetValidationError(f"{source} references a missing asset: {reference}")
    return normalized


def _registered_locales(config: Path) -> list[str]:
    try:
        content = config.read_text(encoding="utf-8")
    except OSError as exc:
        raise AssetValidationError(f"cannot read locale config {config}: {exc}") from exc
    match = re.search(r"\blocales\s*:\s*\[(.*?)\]", content, re.DOTALL)
    if not match:
        raise AssetValidationError(f"could not find registered locales in {config}")
    locales = re.findall(r"[\"']([^\"']+)[\"']", match.group(1))
    if not locales or len(locales) != len(set(locales)):
        raise AssetValidationError(f"locale registration is empty or contains duplicates in {config}")
    return locales


def _read_dist(dist: Path) -> dict[str, bytes]:
    if not dist.exists():
        raise AssetValidationError(f"frontend dist is missing: {dist}")
    if not dist.is_dir():
        raise AssetValidationError(f"frontend dist is not a directory: {dist}")
    files: dict[str, bytes] = {}
    for path in sorted(item for item in dist.rglob("*") if item.is_file()):
        relative = path.relative_to(dist).as_posix()
        data = path.read_bytes()
        if not data:
            raise AssetValidationError(f"frontend asset is empty: {relative}")
        files[relative] = data
    if not files:
        raise AssetValidationError(f"frontend dist is empty: {dist}")
    if "index.html" not in files:
        raise AssetValidationError("frontend dist is missing index.html")
    if not any(path.startswith("assets/") and path.endswith(".js") for path in files):
        raise AssetValidationError("frontend dist is missing its JavaScript bundle")
    if not any(path.startswith("assets/") and path.endswith(".css") for path in files):
        raise AssetValidationError("frontend dist is missing its CSS bundle")
    return files


def asset_records(dist: Path, locale_config: Path) -> list[dict[str, object]]:
    files = _read_dist(dist)

    try:
        index = files["index.html"].decode("utf-8")
    except UnicodeDecodeError as exc:
        raise AssetValidationError("index.html is not valid UTF-8") from exc
    html_refs = _HTMLReferences()
    html_refs.feed(index)
    for reference in html_refs.references:
        _local_reference(reference, "", files, "index.html")

    import_pattern = re.compile(r"(?:import\s*\(|\bfrom\s*)[\"']([^\"']+)[\"']")
    css_url_pattern = re.compile(r"url\(\s*[\"']?([^\"')]+)")
    for path, data in files.items():
        suffix = PurePosixPath(path).suffix.lower()
        if suffix not in {".js", ".css"}:
            continue
        try:
            content = data.decode("utf-8")
        except UnicodeDecodeError as exc:
            raise AssetValidationError(f"{path} is not valid UTF-8") from exc
        references = import_pattern.findall(content) if suffix == ".js" else css_url_pattern.findall(content)
        for reference in references:
            parsed = urlsplit(reference)
            if parsed.scheme or parsed.netloc or not parsed.path.startswith("."):
                continue
            _local_reference(reference, posixpath.dirname(path), files, path)

    locales = _registered_locales(locale_config)
    source_root = locale_config.parent / "src" / "locales"
    for locale in locales:
        module = source_root / locale / f"{locale}.ts"
        if not module.is_file():
            raise AssetValidationError(f"registered locale module is missing: {module}")
        if locale != "en" and not any(
            path.startswith("assets/")
            and PurePosixPath(path).suffix == ".js"
            and PurePosixPath(path).name.startswith(f"{locale}-")
            for path in files
        ):
            raise AssetValidationError(f"registered locale has no built JavaScript chunk: {locale}")

    records = []
    for path, data in sorted(files.items()):
        records.append(
            {
                "path": path,
                "bytes": len(data),
                "sha256": hashlib.sha256(data).hexdigest(),
                "mime": mimetypes.guess_type(path)[0] or "application/octet-stream",
            }
        )
    return records


def manifest_text(records: list[dict[str, object]]) -> str:
    return "".join(f"{item['sha256']}  {item['path']}\n" for item in records)


def compare_manifest(records: list[dict[str, object]], path: Path) -> None:
    try:
        lines = path.read_text(encoding="utf-8").splitlines()
    except OSError as exc:
        raise AssetValidationError(f"cannot read asset manifest {path}: {exc}") from exc
    expected: dict[str, str] = {}
    for line_number, line in enumerate(lines, 1):
        match = re.fullmatch(r"([0-9a-f]{64})  (.+)", line)
        if not match:
            raise AssetValidationError(f"invalid manifest entry at {path}:{line_number}")
        digest, name = match.groups()
        if name in expected or PurePosixPath(name).is_absolute() or ".." in PurePosixPath(name).parts:
            raise AssetValidationError(f"invalid or duplicate manifest path at {path}:{line_number}")
        expected[name] = digest

    actual = {str(item["path"]): str(item["sha256"]) for item in records}
    if expected != actual:
        missing = sorted(set(actual) - set(expected))
        stale = sorted(set(expected) - set(actual))
        changed = sorted(name for name in set(actual) & set(expected) if actual[name] != expected[name])
        raise AssetValidationError(
            f"asset manifest mismatch: missing entries={missing}, stale entries={stale}, changed hashes={changed}"
        )


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dist", type=Path, default=Path("internal/site/dist"))
    parser.add_argument("--locale-config", type=Path, default=Path("internal/site/lingui.config.ts"))
    parser.add_argument("--manifest", type=Path)
    parser.add_argument("--write-manifest", type=Path)
    args = parser.parse_args(argv)

    try:
        records = asset_records(args.dist, args.locale_config)
        if args.manifest:
            compare_manifest(records, args.manifest)
        if args.write_manifest:
            args.write_manifest.parent.mkdir(parents=True, exist_ok=True)
            args.write_manifest.write_text(manifest_text(records), encoding="utf-8")
        canonical = manifest_text(records).encode("utf-8")
        print(
            json.dumps(
                {
                    "status": "PASS",
                    "files": len(records),
                    "bytes": sum(int(item["bytes"]) for item in records),
                    "manifest_sha256": hashlib.sha256(canonical).hexdigest(),
                    "locales": _registered_locales(args.locale_config),
                },
                ensure_ascii=False,
            )
        )
    except AssetValidationError as exc:
        print(f"frontend asset validation failed: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
