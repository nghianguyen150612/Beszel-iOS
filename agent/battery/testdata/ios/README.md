These are synthetic, sanitized shape fixtures, not new device captures.
Capacity and charging facts (42/100, raw 3989, connected/charging/installed)
come from docs/device-validation.md, STEP 39, dated 2026-09-17/18, Beszel 0.19
on iPad4,4 / iOS 12.5.7. Recursive and legacy pipe prefixes follow the existing
iOS parser's documented output forms. Nested BatteryData values 1/2 are negative
fixture data, not device facts. Registry IDs, serials, hostnames and unrelated
properties are omitted. Edge cases in tests are wholly synthetic. These prove
parser behavior, never v0.21 kernel or device behavior.
