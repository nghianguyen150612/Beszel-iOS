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

## Compatibility and accounting

The Hub converts legacy GB values only if the byte arrays are absent. Conversion rejects negative, NaN, and infinite values and saturates finite overflow at `MaxUint64`. Byte values with used/cache/ARC greater than total are bounded by total; swap usage is bounded by swap capacity. Zero-capacity arrays normalize to all zero. Existing GB values are kept for old integrations.

Record averaging uses independent observation counts for RAM and swap. Unknown capacity samples do not dilute measured samples. Byte sums use a 128-bit accumulator and exact integer division, avoiding uint64 wrap for large capacities. Percentages are averaged only over valid memory samples; invalid, non-finite, or out-of-range percentages are derived from bounded byte values. Max memory remains a high-water value and is not added to the byte-array averages.

The frontend prefers exact bytes, falls back to GB only when the exact value is absent, rejects non-finite/negative values, bounds usage to total capacity, and bounds cache/ARC stack values to remaining capacity. Zero cache and ARC series are hidden. Swap is shown only when both total and used bytes are positive, and plotted usage is bounded to total.

## Local evidence

Implemented and locally tested: Go JSON v1/JSON v2 and CBOR round trips, CBOR key 41/42 presence, legacy-peer decoding, NaN/infinity/overflow conversion, used-greater-than-total bounding, uint64 record averaging, missing-sample averaging, synthetic iOS swap unavailability, and frontend conversion helpers. Focused commands and outcomes are recorded in `ios-p024-checkpoints.md` and `ios-p024-validation.json`.

Not yet established by this document: full Agent/Hub/database integration, all rollup intervals, native iOS execution, hosted CI, source artifact verification, and final Gate I.
