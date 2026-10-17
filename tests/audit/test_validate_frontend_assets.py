import tempfile
import unittest
from pathlib import Path

from scripts.audit.validate_frontend_assets import (
    AssetValidationError,
    asset_records,
    compare_manifest,
    manifest_text,
)


class FrontendAssetValidationTests(unittest.TestCase):
    def setUp(self):
        self.temp_dir = tempfile.TemporaryDirectory()
        self.root = Path(self.temp_dir.name)
        self.dist = self.root / "dist"
        self.locale_config = self.root / "lingui.config.ts"
        self.locale_config.write_text('export default { locales: ["en", "ar"] }\n', encoding="utf-8")
        for locale in ("en", "ar"):
            locale_dir = self.root / "src" / "locales" / locale
            locale_dir.mkdir(parents=True)
            (locale_dir / f"{locale}.ts").write_text("export const messages = {}\n", encoding="utf-8")
        self.write_valid_tree()

    def tearDown(self):
        self.temp_dir.cleanup()

    def write_valid_tree(self):
        (self.dist / "assets").mkdir(parents=True, exist_ok=True)
        (self.dist / "static").mkdir(parents=True, exist_ok=True)
        (self.dist / "index.html").write_text(
            '<link rel="stylesheet" href="./assets/main.css">\n'
            '<script type="module" src="./assets/main.js"></script>\n'
            '<img src="./static/icon.svg">\n',
            encoding="utf-8",
        )
        (self.dist / "assets" / "main.js").write_text('import("./ar-123.js");\n', encoding="utf-8")
        (self.dist / "assets" / "ar-123.js").write_text('export const locale = "ar";\n', encoding="utf-8")
        (self.dist / "assets" / "main.css").write_text('body{background:url("./font.woff2")}\n', encoding="utf-8")
        (self.dist / "assets" / "font.woff2").write_bytes(b"font-fixture")
        (self.dist / "static" / "icon.svg").write_text("<svg></svg>\n", encoding="utf-8")

    def test_valid_dist_produces_sorted_sha256_manifest(self):
        records = asset_records(self.dist, self.locale_config)
        names = [record["path"] for record in records]
        self.assertEqual(names, sorted(names))
        self.assertIn("assets/ar-123.js", names)
        self.assertIn("static/icon.svg", names)
        self.assertEqual(len(manifest_text(records).splitlines()), len(records))

    def test_missing_dist_is_rejected(self):
        with self.assertRaisesRegex(AssetValidationError, "dist is missing"):
            asset_records(self.root / "missing", self.locale_config)

    def test_empty_dist_is_rejected(self):
        empty = self.root / "empty"
        empty.mkdir()
        with self.assertRaisesRegex(AssetValidationError, "dist is empty"):
            asset_records(empty, self.locale_config)

    def test_missing_index_is_rejected(self):
        (self.dist / "index.html").unlink()
        with self.assertRaisesRegex(AssetValidationError, "missing index.html"):
            asset_records(self.dist, self.locale_config)

    def test_missing_javascript_bundle_is_rejected(self):
        (self.dist / "assets" / "main.js").unlink()
        (self.dist / "assets" / "ar-123.js").unlink()
        with self.assertRaisesRegex(AssetValidationError, "missing its JavaScript bundle"):
            asset_records(self.dist, self.locale_config)

    def test_missing_css_bundle_is_rejected(self):
        (self.dist / "assets" / "main.css").unlink()
        with self.assertRaisesRegex(AssetValidationError, "missing its CSS bundle"):
            asset_records(self.dist, self.locale_config)

    def test_missing_locale_chunk_is_rejected(self):
        (self.dist / "assets" / "ar-123.js").unlink()
        (self.dist / "assets" / "main.js").write_text("export {};\n", encoding="utf-8")
        with self.assertRaisesRegex(AssetValidationError, "no built JavaScript chunk: ar"):
            asset_records(self.dist, self.locale_config)

    def test_missing_html_and_css_references_are_rejected(self):
        (self.dist / "assets" / "main.css").write_text('body{background:url("./missing.woff2")}\n', encoding="utf-8")
        with self.assertRaisesRegex(AssetValidationError, "references a missing asset"):
            asset_records(self.dist, self.locale_config)

    def test_non_relative_html_asset_path_is_rejected(self):
        (self.dist / "index.html").write_text('<script src="/assets/main.js"></script>\n', encoding="utf-8")
        with self.assertRaisesRegex(AssetValidationError, "non-relative asset path"):
            asset_records(self.dist, self.locale_config)

    def test_stale_manifest_is_rejected(self):
        records = asset_records(self.dist, self.locale_config)
        manifest = self.root / "assets.sha256"
        manifest.write_text(manifest_text(records).replace("assets/main.js", "assets/stale.js"), encoding="utf-8")
        with self.assertRaisesRegex(AssetValidationError, "asset manifest mismatch"):
            compare_manifest(records, manifest)

    def test_empty_asset_is_rejected(self):
        (self.dist / "static" / "icon.svg").write_bytes(b"")
        with self.assertRaisesRegex(AssetValidationError, "asset is empty"):
            asset_records(self.dist, self.locale_config)

    def test_invalid_mime_type_is_not_silently_assigned(self):
        records = asset_records(self.dist, self.locale_config)
        mime_by_path = {record["path"]: record["mime"] for record in records}
        self.assertEqual(mime_by_path["assets/main.js"], "text/javascript")
        self.assertEqual(mime_by_path["assets/main.css"], "text/css")


if __name__ == "__main__":
    unittest.main()
