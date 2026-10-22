# P024 memory telemetry compatibility

## Wire contract

The pinned upstream protocol adds these fields to `system.Stats`:

| Meaning | Go type | JSON name | CBOR key | Unit |
|---|---|---|---:|---|
| Total, used, cache/buffers, ZFS ARC | `[4]uint64` | `mem` | 41 | bytes |
| Maximum used memory | `uint64` | `memm` | not sent over CBOR | bytes |
| Swap total, used | `[2]uint64` | `swap` | 42 | bytes |

The deprecated `m`, `mu`, `mb`, `mz`, `mm`, `s`, and `su` fields remain in GB for older Hubs, Agents, and integrations. The migration code never interprets those values as bytes. Modern nonzero byte arrays take precedence. JSON and CBOR fixtures cover the pinned names/keys, old-peer decoding, and exact uint64 round trips.

## Missing and unsupported values

The protocol uses fixed arrays, so it cannot distinguish an unavailable array element from a measured zero. P024 keeps the upstream wire contract and does not add keys. The iOS collector reports RAM total from `hw.memsize` and used from supported Darwin page accounting. It does not use the Linux htop formula, infer cache or ARC from inactive pages, or claim swap usage. The iOS cache and ARC slots remain zero in `mem`; the frontend omits their zero-valued series because zero also represents an unsupported source. An all-zero `swap` array is omitted by the existing `omitzero` JSON and CBOR tags.

The Agent clears per-sample values before collection. If the source temporarily fails, the sample fields remain omitted/zero rather than carrying forward stale measurements. A real measured RAM capacity of zero is treated as unavailable; Hub details retain their prior known capacity until a nonzero measurement arrives.

The frontend shows swap only when capacity is positive and a sample carries that capacity; later measured zero-use samples remain visible. If every sample is all-zero, swap remains unavailable and is hidden. This follows the upstream fixed-array wire contract: it cannot distinguish an unsupported element from a measured zero. P024 adds no wire keys and does not claim that a remote host has no swap when the array is all-zero.

## Compatibility and accounting

The Hub converts legacy GB values only if the byte arrays are absent. Conversion rejects negative, NaN, and infinite values and saturates finite overflow at `MaxUint64`. Byte values with used/cache/ARC greater than total are bounded by total; swap usage is bounded by swap capacity. Zero-capacity arrays normalize to all zero. Existing GB values are kept for old integrations.

Record averaging uses independent observation counts for RAM and swap. Unknown capacity samples do not dilute measured samples. Byte sums use a 128-bit accumulator and exact integer division, avoiding uint64 wrap for large capacities. Percentages are averaged only over valid memory samples; invalid, non-finite, or out-of-range percentages are derived from bounded byte values. Max memory remains a high-water value and is not added to the byte-array averages.

The frontend prefers exact bytes, falls back to GB only when the exact value is absent, rejects non-finite/negative values, bounds usage to total capacity, and bounds cache/ARC stack values to remaining capacity. Zero cache and ARC series are hidden. A swap chart is shown when total capacity is positive and usage is measured, including a measured zero-use sample; an omitted/all-zero unsupported array stays hidden. Plotted usage is bounded to total.

## Local evidence

Implemented and locally tested: Go JSON v1/JSON v2 and CBOR round trips, CBOR key 41/42 presence, legacy-peer decoding, NaN/infinity/overflow conversion, used-greater-than-total bounding, uint64 record averaging, missing-sample averaging, synthetic iOS swap unavailability, direct Agent-to-legacy-Hub serialization, and frontend conversion helpers including measured zero-use swap. The full frontend build, full Go suite, and affected race suites pass locally. Hosted checks and final native qualification are tracked separately in `ios-p024-checkpoints.md` and `ios-p024-validation.json`.

Protocol fixtures cover legacy JSON and CBOR peers plus historical GB-only values. A disposable PocketBase fixture verifies mixed GB/byte samples through all four Hub system rollup tiers and restart. Hosted P013 separately exercises a genuine v0.19 migration fixture; it does not itself contain mixed P024 byte/GB records. Native CI verifies source selection and compilation, not runtime execution on an iOS device. The all-zero wire representation remains ambiguous by upstream design; P024 treats it as unavailable for charting.
