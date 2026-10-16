import importlib.util
from pathlib import Path
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('wifi_abi', ROOT / 'scripts/audit/ios_wifi/abi.py')
abi = importlib.util.module_from_spec(spec)
spec.loader.exec_module(abi)


class ABIPinTests(unittest.TestCase):
    def test_untrusted_headers_fail_before_declaration_parsing(self):
        with tempfile.TemporaryDirectory() as tmp:
            for filename in abi.PINNED:
                (Path(tmp) / filename).write_text('fabricated header')
            with self.assertRaisesRegex(ValueError, 'unexpected pinned source'):
                abi.declarations(Path(tmp))
