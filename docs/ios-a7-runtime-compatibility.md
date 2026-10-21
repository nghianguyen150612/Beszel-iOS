# Apple A7 runtime compatibility — P007

P007 reviews build-target compatibility and the Go runtime workaround. It does
not establish SIGILL-free execution of the 0.21 binaries on a physical Apple A7.
The historical iPad mini 2 / Beszel 0.19 record in `device-validation.md` remains
unchanged. A7-DEVICE-01 belongs to P018 / Gate F.

## Baseline and defects

Starting integration: `6b14ebcdadc44641f4f84a5d7999ebf964905aa3`.
Product `iOS`: `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`.
Frozen `main`: `bb23f99b3eeaced9608e11e280e7c986dba650be`.
All 178 canonical upstream commits are ancestors. This work uses
`ios/a7-runtime-compatibility`, with PR base `ios/upstream-v021-integration`.

The old patch accepted ordered substrings rather than the complete executable
function shape. An added NOP or changed delay arithmetic could pass its guards.
It wrote directly to the runtime file, so an interrupted write could truncate
it. Its replacement lacked the original zero-input guard; decrementing zero
would wrap and spin for 2^32 iterations. These are concrete tooling defects.

## Supported source and replacement

P007 originally validated Go 1.27.1. P019 updates the current candidate to
**Go 1.27.2**, matching `go.mod`; the patcher now requires exactly that version
and checks the complete executable source shape from the Go 1.27.2
`src/runtime/asm_arm64.s` function through its next-function comment boundary.
[Fixture provenance](../tests/audit/fixtures/go-runtime-provenance.md) records
both upstream sources, attribution, BSD license and fixture hashes.

The patch resolves `go env -json GOROOT GOVERSION`, requires a valid absolute
GOROOT and regular non-symlink runtime file, and checks one start and one end
marker. Every executable line, operand, label and directive must match the
reviewed function. Only comments and whitespace are normalized. Extra functions,
unknown instructions, altered arithmetic or counter sequences fail closed.
The prefix and suffix remain byte-identical; the assembly file is not rewritten
from a template.

The replacement loads the 32-bit delay argument, returns immediately on zero,
and otherwise executes YIELD, decrements once, and branches until zero. It
contains neither CNTFRQ_EL0 nor CNTVCT_EL0 access. This retains a bounded spin
hint, using the historical A7-compatible strategy. Go 1.27.1's original counter
loop measures nanoseconds; YIELD iterations are not a calibrated nanosecond
clock, so identical timing or performance is not claimed. Physical polling,
contention behavior and service endurance remain P018 obligations.

All verification finishes before a same-directory temporary file is created.
The patch flushes and fsyncs the replacement, preserves file permissions, checks
that the input did not change, then atomically renames it. Errors before rename
leave the original intact and remove the temporary file. There is no guarantee
against arbitrary concurrent writers after the last check; CI owns an exclusive
job-local copy. A repeated invocation deliberately exits nonzero, with no write.
Unsupported versions report the expected and observed version explicitly. The
1.27.1 fixture remains as historical P007 evidence; the P019 patch tests use a
separate 1.27.2 fixture.

## Toolchain isolation

CI copies the complete setup-go GOROOT into `RUNNER_TEMP/go-ios-a7`, retaining
`bin`, `pkg`, `src` and the correct directory layout. The copied toolchain must
report that GOROOT and the module's pinned version. PATH, GOROOT and
GOTOOLCHAIN=local select it for **both** iOS builds and the validation builds;
a dedicated iOS GOCACHE prevents reuse of unrelated host runtime objects.
No sudo or mutation of the setup-go installation is required. Its original
runtime source hash is checked after patching and after both native builds.
Unpatched macOS regression builds run before the isolation step.

All negative tests use disposable fake GOROOT fixtures. A separate disposable
copy of the actual Linux Go toolchain also passed patch application, repeat
rejection and original-source preservation. Hosted macOS remains authoritative
for the copied toolchain's native iOS link behavior.

## Test and binary evidence

The 13 runtime unit tests cover the known fixture, exact output and zero guard;
15 malformed source mutations; missing/invalid GOROOT, source and go command;
failed go env, invalid JSON and unsupported versions; symlink rejection; repeat
application; and injected flush/rename failures. Failed invocations preserve
input bytes and unrelated files and leave no temporary output.

`verify-ios-runtime-binary.py` inspects one `runtime.procyieldAsm[.abi0]` symbol
from separate unstripped Agent/Hub validation builds. It checks every ARM64
instruction word and the branch displacements, permitting only alignment zeros
after RET. Missing/stripped symbols, extra instructions and counter-read
mutations fail. Production `-s -w` flags stay intact. Source validation,
assembled-fixture inspection and linked-binary inspection are distinct evidence;
none substitutes for actual A7 execution.

P007 hosted run URLs and results are historical evidence for Go 1.27.1. P019
must record its own tested source SHA, Go 1.27.2 runtime hashes, artifact
provenance and results in the P019 checkpoint before the updated patch is
considered qualified. Exact hosted run URLs, tested source SHA, runtime hashes and results are in
[`ios-build-validation.json`](ios-build-validation.json) and
[PR #9](https://github.com/nghianguyen150612/Beszel-iOS/pull/9). The PR records the
final documentation-inclusive HEAD and its rerun, avoiding a self-referential
commit hash in this file.

P007 does not complete Gate C: P008–P012 still own telemetry, transport, supported
collectors, package updates and Wi-Fi. P018 owns physical-device acceptance.
