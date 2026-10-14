# P006 — Hoàn tất tích hợp snapshot Beszel upstream đã đóng băng

**Cổng tích hợp mã nguồn P006: PASS. Gate B: PASS.** Đã tích hợp đầy đủ lịch sử upstream đã đóng băng; đây không phải chấp thuận phát hành. Gates C–F và P007–P018 vẫn bắt buộc. Không phát hành, triển khai hoặc bắt đầu P007.

## Nguồn gốc Git và phạm vi lịch sử

| Reference | Full SHA |
| --- | --- |
| Starting integration | `26a6447801fe66f59eddec376f15cb6d0b94aed9` |
| Frozen upstream target / main | `bb23f99b3eeaced9608e11e280e7c986dba650be` |
| Upstream merge | `211bdf5c131328f1a6bc691ecb1547de7e9f68c9` |
| Merge parent 1 | `26a6447801fe66f59eddec376f15cb6d0b94aed9` |
| Merge parent 2 | `bb23f99b3eeaced9608e11e280e7c986dba650be` |
| Tested source candidate | `15bc4226eac9f1dd8a9c5d0dd6de5a4ccd2966f5` |
| Unchanged product iOS | `0c3e4cd22ac24264834c401ad412e44aeda4e4f3` |

Cloud checkout dùng `git clone --branch iOS --single-branch` tại `/workspace/p006-cloud/Beszel-iOS`, giữ nguyên checkout có sẵn. Các ref đầu vào khớp SHA yêu cầu. Đã đọc hồ sơ P004/P005 và kiểm tra merge/các PR bổ sung #6/#7 trước khi sửa nguồn. `TARGET..INTEGRATION` có 57 commit phía tích hợp, không phải commit chỉ có ở upstream. `INTEGRATION..TARGET` và tập SHA P006 chính xác từ P002 đều có 37 commit.

Nhánh làm việc `ios/merge-upstream-snapshot`; PR [#8](https://github.com/nghianguyen150612/Beszel-iOS/pull/8), base `ios/upstream-v021-integration`. Đã chạy:

```sh
git merge --no-ff --no-commit bb23f99b3eeaced9608e11e280e7c986dba650be
git commit -m "merge(ios): integrate remaining frozen upstream changes"
```

Đã xác minh hai parent theo đúng thứ tự trong bảng. Hai regression commit riêng `fd6eb7f69bf6240f36d20c0a3ad7f397d45035fd` và `15bc4226eac9f1dd8a9c5d0dd6de5a4ccd2966f5` bổ sung test P006 và license/attribution fixture; không cần sửa production ngoài merge. Hồ sơ bằng chứng được commit riêng với tiêu đề `docs(ios): record completed upstream snapshot integration`.

**P004 25/25; P005 116/116; P006 37/37; total 178/178; P006 P0 4/4; iOS patches 13/13.**

Đã kiểm tra bằng chương trình rằng toàn bộ 178 SHA đầy đủ trong CSV canonical đều reachable. Ba khoảng fork→v0.20, v0.20→v0.21, v0.21→target có 25, 116, 37 commit. Các mốc đóng băng, merge và hồ sơ P004/P005 đều còn nguyên. JSON lưu từng SHA, dòng CSV, tệp thay đổi, ưu tiên, owner P003 và kết quả ancestry; không dùng chuỗi phiên bản để thay thế bằng chứng lịch sử.

## Xung đột và đối chiếu cây nguồn

**Không có xung đột văn bản.** Không dùng chiến lược ours/theirs hàng loạt hoặc thay thế nguồn thủ công. Đã xem xét ngữ nghĩa các automatic merge ở Docker workflow, `agent/system.go`, Bun lock và metadata của hai Helm chart:

- Giữ sáu biến thể Docker build một lần/push hai registry của upstream và bộ lọc iOS `!v*-ios.*`. Giữ cách ly generic release và workflow iOS chuyên dụng.
- Giữ cgroup CPU/RAM opt-in, sửa rollback counter và tái cấu trúc RAM cùng iOS CPU bypass và `adjustPlatformSystemDetails()`. `GOOS=ios GOARCH=arm64 CGO_ENABLED=1 go list` thực tế chọn CPU/RAM fallback không phải Linux, metadata/battery iOS và GPU/Wi-Fi fallback an toàn. iOS không phụ thuộc cgroup Linux.
- Bun 1.4.0 frozen install không đổi lock SHA256 `16643e3228f9b0fac1e5134d89782c096f49e8071489cc2a4f977f3565f02afa`. Khác biệt jiti hoisting kế thừa không đổi 501 mục package/version so với target; package.json, npm lock và Go dependencies khớp upstream.
- Giữ chart package **0.1.8**, appVersion thành **0.21.0**. Lint/render/package và kiểm tra version chưa xuất bản đều pass. Không xuất bản; lỗi trùng version 0.1.7 của P005 không tái diễn.

JSON liệt kê mọi khác biệt với cây target, giải thích và Git blob ID. Nguồn auth/ownership/history/transport dùng chung khớp upstream. Khác biệt production dùng chung chỉ là battery/metadata iOS, loại trừ collector macOS và fallback vắng dữ liệu đã có. Các khác biệt build/release/installer giữ hợp đồng iOS. Những khác biệt khác là test status lock, môi trường Bun test, dọn khoảng trắng, tài liệu sản phẩm và audit fixture. Không bỏ tính năng production upstream.

## Mười ba bản vá logic và bốn bất biến P0

Snapshot trước merge lưu 13 bề mặt bản vá, SHA256 nguồn, build settings, workflow và installer pin tại SHA tích hợp đầu vào. [JSON preservation](upstream-v021-p006-preservation.json) lưu hash sau merge và giải thích từng mục. Đánh giá lại P006: 10 PRESERVED, 3 RECONCILED WITH TEST EVIDENCE, 0 BLOCKED.

| Logical patch | Canonical owner | P006 disposition |
| --- | --- | --- |
| `IOS-A7-RUNTIME` | P007 | PRESERVED |
| `IOS-AGENT-BATTERY` | P008 | PRESERVED |
| `IOS-AGENT-BUILD-PROBE` | P007 | PRESERVED |
| `IOS-AGENT-SYSTEM-HOOK` | P008 | RECONCILED WITH TEST EVIDENCE |
| `IOS-FRONTEND-BUILD` | P015 | RECONCILED WITH TEST EVIDENCE |
| `IOS-HUB-BUILD-PIPELINE` | P015 | PRESERVED |
| `IOS-INSTALLER-ARCH-DETECTION` | P016 | PRESERVED |
| `IOS-INSTALLER-BOOTSTRAP-INTEGRITY` | P016 | PRESERVED |
| `IOS-INSTALLER-LIFECYCLE` | P016 | PRESERVED |
| `IOS-INSTALLER-VALIDATION` | P017 | PRESERVED |
| `IOS-PORT-DOCUMENTATION` | P017 | PRESERVED |
| `IOS-RELEASE-ISOLATION` | P016 | RECONCILED WITH TEST EVIDENCE |
| `IOS-UPSTREAM-SYNC-CI` | P017 | PRESERVED |

Runtime script/workflow native không đổi: ios/arm64, CGO=1, clang wrapper dùng iPhoneOS và minimum iOS 12.0. Fixture Go 1.27.1 hợp lệ và năm trường hợp fail-closed đều pass, kể cả áp dụng lặp. Giữ AppleARMPMUCharger, Darwin exclusion, sysctl/plist và nhận diện A7. Giữ installer 1.0.0, state format 1, engine pin `d4b4e381a76c75ac79cb7c99558853b62c7c2163879a72c0333acc5c40092d0c`.

| Full upstream SHA | Executed focused tests | Outcome |
| --- | --- | --- |
| `36c14e903d73b226a2df5bbf7ab4fbf1943c2906` | `TestLegacyHubGetsSinglePayload`; `TestModernHubRequestResponse`; `TestModernHubMalformedFallbackSinglePayload` | PASS |
| `7305eda972cb423bd51ac18f4966312b346360d0` | `TestUpdatePingsAgentAfterFailedFetch`; `TestSlowAgentKeepsConnectionAcrossPolls`; `TestFetchDataTimeoutKeepsWebSocketOpen` | PASS |
| `d91f06f3c533b231346dc1d0ca5bf0bca265e462` | `TestSlowAgentKeepsConnectionAcrossPolls` | PASS |
| `f8d3fe1817b2ac977c051243d6ffc9077a000ba3` | `TestZeroValueJSONMatchesV1`; `TestSerializationPresenceContracts`; `TestSerializationCBORKeysAndOptionalZero`; `TestPublicAndAlertJSONZeroContracts`; `TestFetchDataViaWebSocketDoesNotRetainOmittedFields` | PASS |

Legacy fix thoát ngay sau một payload theo phiên bản; malformed modern request và các phiên fallback lặp cũng chỉ gửi một payload. Deadline Hub 100s vượt 60s polling + 30s timeout, cho phép collection chậm tồn tại qua poll lỗi. Timestamp atomic theo dõi message Agent; dừng keepalive khi im lặng ba phút, sau đó deadline đóng peer cũ. Test chứng minh không có vòng keepalive vô hạn; response muộn bị bỏ và poll sau khôi phục được.

Inventory nil không có omitempty giữ mặc định hiện hữu: v1 null, v2 []; inventory rỗng tường minh là [] ở cả hai. Test JSON v1/v2 bao phủ absent/zero/false, chuỗi/map/slice rỗng, inventory rỗng tường minh, con trỏ zero, cấu trúc lồng, zero bắt buộc và trường scalar bị lược bỏ. Giữ numeric CBOR keys, battery/zero-pointer, guard Agent cũ và fresh decode. `Details` chứa Go duration, truyền bằng CBOR; marshal trực tiếp non-nil Details qua JSON v2 mặc định thiếu duration format và không phải đường lưu stats. Test theo hợp đồng thực tế, không sửa wire tags.

## Chức năng, bảo mật và cơ sở dữ liệu

HTTPS monitor xác minh TLS mặc định. Opt-out riêng từng monitor dùng cloned transport riêng; chuỗi secure–insecure–secure trên cùng manager chứng minh cách ly. Giữ bool migration, persistence/config query, CBOR key 6, sync, API copy và switch/badge frontend. Record không phải HTTP bị xóa tùy chọn; không thêm global TLS bypass.

HUB_HEADERS giữ kiểm tra tên/giá trị và lỗi không lộ secret value. X-Token/X-Beszel override sau canonicalization bị bỏ qua; token/version cấu hình vẫn quyết định auth. Dữ liệu lỗi làm startup thất bại. Test signature/fingerprint/token, custom CA/hostname, reconnect, SSH fallback/deadline/cancellation đều pass. Giữ và kiểm thử ownership monitor, quyền alert/delivery, collection permissions và systemd private cache của upstream.

Giữ các sửa collector: CPU rollback, cgroup Linux opt-in, reset counter interface/container, apt-wrapper/xbps, SMART/NVMe wide counter/empty/error/rescan namespace, Intel JSON/buffer, NVML power fallback, ZFS status/root capacity/refquota và systemd availability/user journal. Khả năng iOS không hỗ trợ vẫn absent/no-op. Hub giữ model telemetry Linux/Windows/macOS từ xa. Adapter package native và RSSI thuộc P011/P012.

Giữ sửa chart empty/isolated, latency tolerance, hủy realtime subscription resolve muộn, textContent an toàn, update count khi host down và layout dialog ổn định. Frontend test/build pass; ma trận browser/device/remote-OS đầy đủ vẫn là công việc sau.

P006 thêm `1790924727_zfs_pool_status.go` rồi `1790964708_network_monitor_skip_tls_verify.go`, trước `initial-settings.go` theo thứ tự tên hiện hữu. Không đổi ID, nội dung, thứ tự migration, ownership, retention/history. Test database mới dùng dữ liệu tổng hợp và targeted migration down/up/idempotence pass. Down migration làm mất giá trị trường mới. Negative test cho thấy generic down/up có thể chạy lại initial-settings một chiều và trùng superuser. **Full 0.19 backup/upgrade/restore/interruption vẫn BLOCKED/PENDING, thuộc P013/P017 tại Gates D/E; không chấp thuận nâng cấp production.** Không mở `/var/lib/beszel-hub` hoặc database iPad thật.

## Kết quả kiểm thử và CI

Toolchain Go **1.27.1**, Bun **1.4.0**. Go suite canonical baseline pass. Bun test trên clean checkout lúc đầu thiếu Lingui catalog sinh tự động; build/catalog chuẩn bị thực tế giải quyết điều kiện này. Sau merge chạy frozen install, `bun run sync`, test và production build thật. Không dùng thư mục embed giả. Sau build đã trả các catalog nguồn sinh thêm về đúng nguồn tracked đóng băng.

| Command | Result |
| --- | --- |
| `python3 scripts/audit/validate-upstream-v021-matrix.py` | PASS (exit 0) |
| `python3 scripts/audit/validate-upstream-v021-readiness.py` | PASS (exit 0) |
| `python3 -m unittest discover -s tests/audit -p 'test_*.py' -v` | PASS (exit 0) |
| `sh tests/upstream-sync-test.sh` | PASS (exit 0) |
| `sh tests/ios-patch-sentinels.sh` | PASS (exit 0) |
| `sh tests/bootstrap-test.sh` | PASS (exit 0) |
| `sh tests/install-sh-test.sh` | PASS (exit 0) |
| `git diff --check` | PASS (exit 0) |
| `cd internal/site && sha256sum bun.lock && bun install --frozen-lockfile && sha256sum bun.lock` | PASS (exit 0) |
| `cd internal/site && bun run sync` | PASS (exit 0) |
| `cd internal/site && bun test` | PASS (exit 0) |
| `cd internal/site && bun run build` | PASS (exit 0) |
| `go test -tags "testing no_ui" ./...` | PASS (exit 0) |
| `env -u NETWORK go test -race -tags 'testing no_ui glibc' ./agent -run 'Test(NvmlSampleToFloat\|PowerFromSamples.*)' -count=1 -v` | PASS (exit 0) |

Audit unit: **20**, gồm **6 A7 fixture**. Sync/sentinels/bootstrap/installer: **10/44/54/814**. Frontend: **44 test, 149 expectation**; dist thật: **95 tệp**, 88 JavaScript asset. Linux Agent/Hub build pass. Go file selection iOS/macOS/Linux pass. Kiểm tra Windows tùy chọn bị chặn vì thiếu thư mục binary LHM sinh bởi upstream; không tuyên bố Windows build pass.

Focused suite: **936 sự kiện test/subtest pass**, không lỗi. Race suite Agent, Hub systems/WebSocket/SSH, alerts, migrations, entities: **1775 sự kiện test/subtest pass**, không lỗi. Bốn ICMP socket subtest là **SKIPPED**, không phải pass, do quyền socket cloud; package không có test cũng không được tính là test pass. P010/P017/P018 sở hữu kiểm thử ICMP native. Lần race compile đầu trùng lúc Vite sinh lại dist nên thiếu embed asset. Lần chạy asset ổn định hoàn tất 11 package nhưng alerts hết default timeout 10m khi đang tiến triển qua fixture mật khẩu. Đã chạy lại toàn bộ alerts không đổi test với `-timeout 30m` và pass. Tổng race chỉ cộng package hoàn tất, loại sự kiện alerts từ lần timeout.

- [iOS Build — 38486554442](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38486554442): **success**, SHA `fd6eb7f69bf6240f36d20c0a3ad7f397d45035fd`.
- [iOS installer lifecycle — 38486558092](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38486558092): **success**, SHA `fd6eb7f69bf6240f36d20c0a3ad7f397d45035fd`.
- [Helm charts — 38486605239](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38486605239): **success**, SHA `fd6eb7f69bf6240f36d20c0a3ad7f397d45035fd`.
- [iOS Build — 38488654850](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38488654850): **success**, SHA `15bc4226eac9f1dd8a9c5d0dd6de5a4ccd2966f5`.
- [iOS installer lifecycle — 38488657303](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38488657303): **success**, SHA `15bc4226eac9f1dd8a9c5d0dd6de5a4ccd2966f5`.
- [Helm charts — 38488659904](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38488659904): **success**, SHA `15bc4226eac9f1dd8a9c5d0dd6de5a4ccd2966f5`.

Log native xác nhận Go 1.27.1, Xcode 15.4/iPhoneOS SDK 17.5, patch nguồn A7 thực tế, Agent/Hub CGO, Mach-O ARM64/platform 2/minos 12.0, tên asset và SHA256SUMS hợp lệ. Agent SHA256 `30c34192d45cf82ceb1894d783aa98878256ecd1fc4673845452cb1d3bc6dd08`; Hub SHA256 `4436b39de019e027599fc4dbfc206f2d51f5cc532b915a4bfe6c761921931704`. release-ios/Helm publish được skip có chủ đích. Chỉ upload build artifact Actions thông thường; không release/registry publication. Tải lại artifact vào cloud bị HTTP 403 từ blob; không tuyên bố local Mach-O inspection ngoài kiểm tra hosted đã pass.

Sau evidence commit, chạy lại iOS Build/installer và PR checks cần thiết trên final feature HEAD trước merge. PR #8 và báo cáo hoàn tất ghi chính xác final feature SHA/run URL/post-merge refs; tệp này ghi source candidate và run ban đầu để tránh commit hash tự tham chiếu. Nếu build input đổi phải chạy lại CI liên quan.

Audit dependency đóng băng trả **29 advisory entry trên 9 package**: 1 critical, 19 high, 8 moderate, 1 low. JSON chứa chi tiết/URL. Critical tar nằm ở dev build path `tar@7.5.13 <- @tailwindcss/oxide@4.1.12 <- @tailwindcss/vite@4.1.12`. P015/P017 sở hữu exposure triage, Go inventory và sửa chữa tại Gates D/E; không đổi dependency để che kết quả. Cảnh báo mixed import/large chunk không làm build thất bại.

## Nghĩa vụ còn lại và quyết định Gate B

- P007/P008: platform/A7 và metrics/battery native, P018 chạy thiết bị (C/F).
- P009/P010: cặp transport cũ/mới, monitor/ICMP native, unsupported fixture với P017/P018 (C/F).
- P011/P012: package jailbreak read-only và adapter Wi-Fi RSSI native (C).
- P013/P017: consistent 0.19 backup, seeded upgrade, full restore, interruption và bất biến account/config/history (D/E; chặn nâng cấp production).
- P014/P015/P017: browser/API/security/remote-OS, dependency triage và bằng chứng candidate tái lập (D/E).
- P016/P017: installer rollback kèm dữ liệu và nghiệm thu service/CLI (E).
- P018: A7 thật/no-SIGILL, service/reboot/reactivation và chấp thuận release (F).

Gate B **PASS** nghĩa là đã tích hợp lịch sử nguồn đóng băng 178 commit, giữ hợp đồng iOS và pass kiểm tra tích hợp. Gates C–F còn pending. P006 không đổi `iOS` hoặc `main`. Không release, push tag, publish image/chart, sửa dữ liệu production hoặc triển khai. Tiếp theo là P007; chưa bắt đầu.

## Danh sách P006 chính xác

Mọi mục đều reachable; functional owner theo P003, tách biệt với quyền sở hữu lịch sử của P006.

| Full upstream SHA | Priority | Canonical functional owner |
| --- | --- | --- |
| `4a40d1772c2c0fcb1aec1fefecbbc56bb6622cbe` | P2 | P016 |
| `7748651f5cb3ee0ccd3d3e6141f315ee5d20454e` | P2 | P016 |
| `dcae7682cddf5e52ba3bd7b3aebfa0f472d8c890` | P2 | P017 |
| `36c14e903d73b226a2df5bbf7ab4fbf1943c2906` | P0 | P009 |
| `74c952b87907a36d84d404ad17a0695be4ac55df` | P1 | P011 |
| `d89f844a2a0abc7474555f0635344245456f82d6` | P1 | P010 |
| `ca40b5b38ffac6336872cdb0d5f4f053916f394a` | P2 | P014 |
| `965518e63c92cbf72e498ab6d10f14f0a78f4196` | P2 | P014 |
| `ef840e0e6d8f066419a351c25312acb2d5323af7` | P1 | P008 |
| `4fe1a0a7a42f655564e0b1577a2bc0dd74af8f58` | P2 | P008 |
| `bc2278e7e9820825ce990fb2fe7fca4b9fe9a1c3` | P1 | P011 |
| `7305eda972cb423bd51ac18f4966312b346360d0` | P0 | P009 |
| `d91f06f3c533b231346dc1d0ca5bf0bca265e462` | P0 | P009 |
| `195afc10b8924ef4d6afc5d409eb8a8e1c43b96c` | P2 | P015 |
| `e30e980854408d46b2e8837a14a0c0d771a18545` | P1 | P015 |
| `5529d2e15e142dcec0ee5a239ffadb33e3295b01` | P1 | P010 |
| `5714643ff9001e70e0fbf8ecb4f55369dcf7ae61` | P2 | P014 |
| `b11e3617990bee170122c4af1913ed79ddb4a632` | P1 | P008 |
| `017ebb091aa64596f9545c89e67f6c880f8bb121` | P1 | P010 |
| `d8b41919751da63f63745b4a9d2ce894dd61342d` | P1 | P008 |
| `13bdd05934a7363016aa9a92c0d5bef13c0e81a4` | P1 | P010 |
| `383e9c707249e88075044d2170a0f1e68411ee76` | P1 | P010 |
| `f8d3fe1817b2ac977c051243d6ffc9077a000ba3` | P0 | P009 |
| `7f63e65f22bff5d4bf5962e3555bc4107c31de6e` | P2 | P010 |
| `7f2eb90f24df180a7c0dd6c1b888aaba65b3dbb6` | P2 | P010 |
| `d324ccc0977d0347e4f94529490fefbc72e54900` | P1 | P009 |
| `4da7d867f237542aadac148eebbcff60f82d9ef9` | P1 | P010 |
| `596f96442e5b60706018e47d63c56bd5abad2a37` | P1 | P016 |
| `da51dfd5986528d7341ba488b96046603298ae0b` | P1 | P014 |
| `3d33d5eabb2bed36b5ff252057a96647dbd8c8c7` | P1 | P010 |
| `665bcb3e53d80a4d4c01e43e0b0bea0ecc85383c` | P1 | P010 |
| `a1cbdc980e05f0a03616d89e0b3d6986e0e59aa8` | P2 | P017 |
| `a1d86d120fa046eabdfc0fc1a674e34aebdc81f1` | P2 | P014 |
| `02f24ed0af859dd9a3e21ba009ed3714733a50f8` | P1 | P010 |
| `9cf75cca7af6a198c418d99a2bd6d78efb2e5d9b` | P2 | P014 |
| `32d469285382521bae4ad3f368a7045f32ff2f5e` | P1 | P010 |
| `bb23f99b3eeaced9608e11e280e7c986dba650be` | P1 | P010 |
