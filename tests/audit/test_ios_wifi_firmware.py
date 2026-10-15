import importlib.util
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('wifi_firmware', ROOT / 'scripts/audit/ios_wifi/firmware.py')
firmware = importlib.util.module_from_spec(spec)
spec.loader.exec_module(firmware)


class FirmwareManifestTests(unittest.TestCase):
    def fixture(self):
        return {'ProductVersion': '12.5.7', 'ProductBuildVersion': '16H81',
                'SupportedProductTypes': ['iPad4,4'], 'BuildIdentities': [
                    {'Info': {'DeviceClass': 'j85ap'}, 'Manifest': {'OS': {'Info': {'Path': 'root.dmg'}}}}]}

    def test_exact_model_board_and_version(self):
        self.assertEqual(firmware.identity(self.fixture()), 'root.dmg')

    def test_reject_wrong_firmware_board_and_ambiguous_or_traversal_image(self):
        for key, value in [('ProductVersion', '12.4'), ('ProductBuildVersion', '16H88'),
                           ('SupportedProductTypes', ['iPhone7,1']), ('BuildIdentities', [])]:
            m = self.fixture(); m[key] = value
            with self.subTest(key=key), self.assertRaises(ValueError):
                firmware.identity(m)
        for path in ['/root.dmg', '../root.dmg']:
            m = self.fixture(); m['BuildIdentities'][0]['Manifest']['OS']['Info']['Path'] = path
            with self.assertRaises(ValueError): firmware.identity(m)
        m = self.fixture(); m['BuildIdentities'].append(
            {'Info': {'DeviceClass': 'j85ap'}, 'Manifest': {'OS': {'Info': {'Path': 'other.dmg'}}}})
        with self.assertRaises(ValueError): firmware.identity(m)
