# Tích hợp Beszel upstream v0.21.0 — P005

**Kết quả: PASS cho tích hợp mã nguồn P005.** P005 tích hợp lịch sử upstream v0.21.0 đã ghim vào nhánh tích hợp được phê duyệt. Đây chưa phải nghiệm thu iOS cuối cùng hay phê duyệt phát hành. Gate B vẫn chờ P006.

## Lịch sử Git và phạm vi commit

| Tham chiếu | SHA |
| --- | --- |
| Nhánh tích hợp ban đầu | `11432ec410a6cb03707789b7a871e7985e18d6fe` |
| `main` sản phẩm đã đóng băng | `bb23f99b3eeaced9608e11e280e7c986dba650be` |
| Mốc v0.20.0 của P004 | `a0dc19eacf6902d2f6934a3efd7ea7277198c40b` |
| Mốc v0.21.0 của P005 | `ec96c134660c998f7eb044e349e3715b65733fe1` |
| Merge commit upstream P005 | `dbe5ce029555ef4ca209092689e3dad9ab5fc7ea` |
| Mã nguồn ứng viên dùng cho các lượt CI hosted đầu tiên | `65167c09f9612d8256e0fd39830e5b1d935b8d36` |

Lệnh merge là `git merge --no-ff --no-commit ec96c134660c998f7eb044e349e3715b65733fe1`. Merge commit có đúng hai parent theo thứ tự: nhánh tích hợp `11432ec410a6cb03707789b7a871e7985e18d6fe`, sau đó là upstream v0.21.0 `ec96c134660c998f7eb044e349e3715b65733fe1`.

Bộ SHA P005 chính xác trong ma trận chuẩn khớp với toàn bộ 116 commit của `v0.20.0..v0.21.0`; hai mốc đều là tổ tiên của mã ứng viên. 25 commit P004 vẫn còn trong lịch sử. Tổng cộng đã tích hợp **141/178** commit; 37 commit từ v0.21.0 tới mốc cuối đã đóng băng được dành cho P006. Không có commit P006 nào được đưa vào. JSON lưu đủ 116 SHA và ảnh chụp trước merge gắn với SHA nhánh tích hợp ban đầu.

Các báo cáo tích hợp và preservation record của P004 được giữ nguyên.

Merge upstream thay đổi 256 đường dẫn. Hai commit theo sau là `1082869c54866a8020c04d33e5d992df6caa48b8` (đối chiếu build tag collector iOS và README bản địa hóa) và `65167c09f9612d8256e0fd39830e5b1d935b8d36` (bài kiểm tra hồi quy khóa status và thiết lập `localStorage` chỉ dành cho môi trường test Bun). Thiết lập Bun sửa môi trường test, không đổi mã chạy sản phẩm hoặc làm yếu assertion.

## Xung đột merge và cách xử lý

1. `internal/entities/systemd/systemd_test.go`: test upstream mới dùng `testing/synctest`; fixture từ P004 cũng tính đến số CPU của máy chạy. Giữ bản upstream sau khi xác nhận fixture thời gian vẫn nhân theo `runtime.NumCPU()` và các assertion vẫn còn.
2. `readme.md`: README chữ thường của upstream xung đột với các tệp README tiếng Anh/tiếng Việt hiện hành của sản phẩm. Đã xóa tệp chữ thường bị trùng. Hai tệp bản địa hóa giữ nguyên các tuyên bố iOS và bổ sung mô tả lịch sử monitor upstream chỉ như kết quả từ Agent từ xa được hỗ trợ; không khẳng định iOS tự dò mạng.

Không dùng cách giải quyết hàng loạt “ours/theirs”. Đã rà soát 12 đường dẫn mã nguồn P0 sau merge; không làm yếu kiểm soát bảo mật nào.

## Các thay đổi upstream được giữ lại

Merge mang theo collector và model v0.21 cho Wi-Fi theo từng giao diện, cập nhật gói, dữ liệu GPU và NVMe SMART, ZFS, systemd, Docker và bộ đếm mạng. Hub nhận các thay đổi v0.21 về monitor, alert, lịch sử và phân quyền; đồng bộ vòng đời WebSocket và quyền sở hữu SSH; giải mã response mới; xử lý hủy và timeout; cùng các cập nhật dashboard, biểu đồ, cập nhật gói, log, bản địa hóa và giao diện alert. Các cập nhật dependency và Helm upstream cũng được giữ.

Cây migration giữ nguyên định danh và thứ tự upstream. P005 sửa `internal/migrations/0_collections_snapshot_0_20_0.go` và thêm lần lượt `1790193183_network_monitor_cert.go`, `1790273174_network_monitor_server.go`. Bộ Go đầy đủ chạy trên các cơ sở dữ liệu test dùng một lần, nhưng chưa chạy quy trình riêng để nâng cấp dữ liệu 0.19 lên 0.21 có seed, sao lưu, khôi phục và mô phỏng gián đoạn. Không mở hoặc sửa cơ sở dữ liệu thật tại `/var/lib/beszel-hub`.

## Rà soát bảo mật và tính đúng đắn P0

Cả 12 commit P0 của P005 đều hiện diện. Các thay đổi mã nguồn tương ứng và test tập trung đều đạt; các package P0 cũng đạt lượt chạy race.

| SHA upstream | Điều kiện bắt buộc | Kiểm tra và test tập trung |
| --- | --- | --- |
| `8bf6917` | Collector Agent chậm không gây vòng reconnect | Làm mới cache nền và giới hạn thời gian thu thập WebSocket; `TestStaleUtilityCachesRefreshInBackground`, `TestFetchDataTimeoutKeepsWebSocketOpen` — PASS |
| `24ec18a` | Mỗi response Agent được giải mã vào dữ liệu mới | Trường bị thiếu không dùng lại giá trị cũ; `TestFetchDataViaWebSocketDoesNotRetainOmittedFields` — PASS |
| `3f20eca` | Handshake WebSocket bất đồng bộ lỗi vẫn cho phép reconnect | Ticker reconnect sống sau disconnect cũ; `TestConnectionManager_TickerSurvivesStaleDisconnect` — PASS |
| `b684ac9` | Handshake SSH có deadline giới hạn | Timeout khi peer im lặng; `TestConnectHandshakeTimeout` — PASS |
| `09a277f` | Đọc/ghi status hệ thống dùng chung có đồng bộ | Rà soát khóa getter/swap; `TestSystemStatusAccessSynchronized` — PASS với race detector |
| `921ded1` | Truy cập vòng đời WebSocket được đồng bộ | Vòng đời connection atomic và đóng đồng thời; `TestWsConnConcurrentClose` — PASS với race detector |
| `a77ed34` | Chỉ hệ thống sở hữu mới được ghi kết quả monitor | Kiểm tra quyền sở hữu; `TestNetworkMonitorResultOwnership` — PASS |
| `0484c54` | Quyền truy cập alert, subscription và gửi thông báo dùng phân quyền hiện hành | Test record, gửi tới hệ thống khác, subscription, quyền bị thu hồi và monitor — PASS |
| `24881fb` | Chi tiết systemd qua proxy không thể dùng chung cache bỏ qua xác thực | Chính sách cache riêng và `Vary: Authorization`; `TestSystemdInfoCachePolicy` — PASS |
| `97db8bd` | Hủy SSH đóng đúng request/session đang chờ | `TestSSHRequestCancellation` và hủy connection dùng chung — PASS với race detector |
| `1997984` | Mỗi hệ thống có một owner SSH transport điều phối | Khởi tạo/quyền sở hữu connection và request đồng thời — PASS với race detector |
| `4ffd836` | Fallback WebSocket/SSH và shutdown của Agent không bị kẹt | Test vòng đời fallback, reconnect và shutdown — PASS |

Kết quả test P0 và race đầy đủ được ghi trong `upstream-v021-p005-preservation.json`.

## Bảo toàn patch iOS và hành vi theo nền tảng

Cả 13 patch iOS logic đều được theo dõi. Patch runtime A7, workflow build iOS, parser pin iOS, vòng đời installer, định tuyến release, kiểm tra đồng bộ upstream và thiết lập build riêng của sản phẩm vẫn còn. Thay đổi vào `agent/system.go` vẫn giữ việc bỏ qua CPU probe trên iOS và hook `adjustPlatformSystemDetails()`; `system_platform_ios.go`, collector `AppleARMPMUCharger` và nguồn pin riêng cho macOS có build tag Darwin phù hợp.

Collector Wi-Fi macOS của P005 gọi `/usr/bin/osascript`, còn collector GPU dùng công cụ macOS. Commit sửa bổ sung giới hạn hai collector này cho macOS và chọn no-op an toàn trên iOS. `GOOS=ios GOARCH=arm64 go list` chọn `wifi_unsupported.go` và `gpu_darwin_unsupported.go`; Darwin chọn collector upstream. Khi iOS không có RSSI, dữ liệu vẫn vắng mặt thay vì hiện thành số 0. Kết quả/lịch sử monitor từ xa dùng được qua model Hub chung. Adapter RSSI iOS thuộc P012. Quyền ICMP native và monitor probe trên iOS chưa được xác thực và không được tuyên bố là đã hỗ trợ.

Workflow release và Docker tổng quát tiếp tục loại trừ `v*-ios.*`; đường dẫn installer và hash engine đã ghim không đổi. Không tạo tag, release hay triển khai nào.

| Patch iOS logic | Chủ sở hữu | Trạng thái P005 |
| --- | --- | --- |
| `IOS-A7-RUNTIME` | P007 | PRESERVED; fixture Go 1.27.1 đúng và fail-closed đều đạt |
| `IOS-AGENT-BATTERY` | P008 | PRESERVED; parser iOS và loại trừ Darwin được giữ |
| `IOS-AGENT-BUILD-PROBE` | P007 | PRESERVED; build Agent/Hub hosted đạt |
| `IOS-AGENT-SYSTEM-HOOK` | P008 | RECONCILED; thay đổi `system.go` upstream vẫn giữ CPU guard và hook iOS |
| `IOS-FRONTEND-BUILD` | P015 | UPSTREAM UPDATE VALIDATED; frozen install Bun 1.4.0 và build thật đạt |
| `IOS-HUB-BUILD-PIPELINE` | P015 | PRESERVED; embed frontend và hosted build đạt |
| `IOS-INSTALLER-ARCH-DETECTION` | P016 | PRESERVED; installer checks đạt |
| `IOS-INSTALLER-BOOTSTRAP-INTEGRITY` | P016 | PRESERVED; kiểm tra bootstrap ghim SHA đạt |
| `IOS-INSTALLER-LIFECYCLE` | P016 | PRESERVED; 814 lifecycle checks đạt |
| `IOS-INSTALLER-VALIDATION` | P017 | PRESERVED; bộ installer local và hosted đạt |
| `IOS-PORT-DOCUMENTATION` | P017 | RECONCILED; README bản địa hóa được giữ khi xử lý xung đột |
| `IOS-RELEASE-ISOLATION` | P016 | PRESERVED; loại trừ tag iOS khỏi workflow release/Docker còn nguyên |
| `IOS-UPSTREAM-SYNC-CI` | P017 | PRESERVED; sync tests và patch sentinel đạt |

## Xác minh

| Kiểm tra | Kết quả |
| --- | --- |
| Validator ma trận và readiness | PASS; bộ P005 chính xác 116/116, map đủ 13 patch logic |
| Unit test audit | PASS; 14 test |
| Bộ Go chuẩn, `env -u NETWORK go test -tags 'testing no_ui' ./...` | PASS |
| Test P0 tập trung và Go race | PASS trên Agent, Hub systems, WebSocket, SSH transport và alerts |
| Build Agent và Hub Linux | PASS |
| Chọn Go build tag iOS/macOS | PASS; iOS fallback và collector Darwin được chọn đúng |
| Fixture patch runtime A7 | PASS; shape Go 1.27.1 đã biết được patch, bốn shape lạ bị từ chối và không sửa fixture |
| Bun frozen install | PASS với Bun 1.4.0; install không đổi lockfile |
| Frontend tests | PASS; 44 test, 149 expectation |
| Frontend production build | PASS; tạo asset thật; có cảnh báo import hỗn hợp và chunk lớn |
| Upstream sync / iOS patch sentinel | PASS; 10 / 44 kiểm tra |
| Bootstrap / installer tests | PASS; 54 / 814 kiểm tra |
| `git diff --check` | PASS |

Các kiểm tra hosted chạy trên mã ứng viên `65167c09f9612d8256e0fd39830e5b1d935b8d36`:

- [iOS Build — run 38421092219](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38421092219): **success**. Go 1.27.1, Xcode 15.4/iPhoneOS SDK 17.5; artifact Agent và Hub là Mach-O arm64, metadata minimum iOS 12.0; checksum đã xác minh. Job `release-ios` được skip cho nhánh feature này.
- [iOS installer lifecycle — run 38421092500](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38421092500): **success**. Đã đạt kiểm tra checksum bootstrap, vòng đời installer, patch sentinel và sync.

Build hosted xác nhận kiến trúc và metadata triển khai; không chứng minh binary chạy không SIGILL trên thiết bị A7 thật.

## Nghĩa vụ còn lại và trạng thái gate

| Công việc | Chủ sở hữu | Trạng thái |
| --- | --- | --- |
| Tích hợp 37 commit upstream đã đóng băng còn lại | P006 | Chờ; chưa bắt đầu P006 |
| Kiểm chứng metrics, pin, runtime iOS native và SIGILL trên A7 | P007, P008, P018 | Chưa làm; không có iPad vật lý để thử |
| Tương thích transport iOS và kiểm chứng monitor/ICMP native | P009, P010, P017, P018 | Chưa làm; chưa xác nhận tổ hợp phiên bản cũ/mới và quyền ICMP native |
| Package update Procursus và adapter Wi-Fi RSSI native | P011, P012 | Chưa làm; fallback iOS hiện tại vẫn an toàn |
| Nâng cấp DB có seed 0.19→0.21, backup nhất quán, restore đầy đủ và khôi phục khi gián đoạn | P013, P017 | Chưa làm; không đụng cơ sở dữ liệu thật |
| API/browser, login, hành vi quyền; phân loại cảnh báo dependency | P014, P015, P017 | Chưa hoàn tất ngoài unit test và production build |
| Kiểm chứng dịch vụ trên thiết bị, reboot và phê duyệt phát hành | P018 | Chưa làm |

**Gate tích hợp P005: PASS.** **Gate B: đã tích hợp 141/178 commit; chờ P006.** Đây chỉ là tích hợp mã nguồn. Kiểm chứng thiết bị, khôi phục DB và runtime vẫn chặn phát hành. Nhánh sản phẩm `iOS` và `main` đã đóng băng không bị sửa; không có release hay triển khai.
