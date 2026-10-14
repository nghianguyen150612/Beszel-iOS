# Kiểm tra build iOS native — P007

SHA integration ban đầu là
`6b14ebcdadc44641f4f84a5d7999ebf964905aa3`; đã kiểm tra ancestry Gate B và toàn bộ
178 commit upstream đóng băng trước khi sửa. Workflow gốc được ghi nhận tại SHA
này (Git blob `eae8a2f6`, hash đầy đủ trong JSON). Build native thành công của P006
là bằng chứng lịch sử, không xác minh thay đổi P007.

## Hợp đồng được củng cố

| Phạm vi | Hành vi P007 |
| --- | --- |
| Runtime | Khớp toàn bộ lệnh Go 1.27.1; YIELD có guard giá trị 0; ghi nguyên tử; bản lạ/lần gọi thứ hai thất bại và không sửa file. |
| Toolchain | Sao chép toàn bộ GOROOT riêng theo job, kiểm tra bản/cấu trúc, GOTOOLCHAIN=local, cache iOS riêng; hash runtime gốc giữ nguyên. |
| CGO | GOOS=ios, GOARCH=arm64, CGO_ENABLED=1 cho cả hai binary; baseline GOARM64=v8.0, chung wrapper clang SDK thiết bị; xóa CGO/GOFLAGS kế thừa. |
| SDK | `xcrun --sdk iphoneos` resolve clang/sysroot; yêu cầu SDK iPhoneOS thiết bị và compiler tồn tại; từ chối cờ đích xung đột; `-arch arm64 -mios-version-min=12.0`. |
| Lựa chọn file | `go list -mod=readonly -deps -json` thật cho cả hai entry point, ios/darwin/linux ARM64 CGO=1; kiểm tra chính xác nhóm implementation và dependency. |
| Frontend | Bun 1.4.0 frozen install → sync → test → build production; lock không đổi; yêu cầu index/JS/CSS thật; mọi byte asset sinh ra phải embed trong Hub. |
| Binary | file/otool native và parser Mach-O: ARM64 ALL thin MH_EXECUTE, iOS thiết bị platform 2/minos 12.0, entry point trong segment thực thi; hỗ trợ minimum iPhoneOS legacy nhưng không nhận platform khác. |
| Entry package | `go version -m` thật phải xác định đúng main package Agent/Hub, Go 1.27.1, ios/arm64/CGO=1, ARMv8.0 và Git revision sạch khớp HEAD đã test. |
| Assembly | Build kiểm tra Agent/Hub đã link riêng, không strip; kiểm tra chính xác opcode/nhánh procyield; giữ strip production. |
| Payload | Đúng Agent, Hub, SHA256SUMS; hai hash duy nhất chỉ dùng tên file; binary thường không rỗng; xác minh cả hai checksum. JSON là artifact review riêng. |
| Build lặp | Hai build mỗi binary bằng cache sạch độc lập; so sánh byte trong cùng job. |

Kiểm tra Mach-O cũ nhận mọi LC_BUILD_VERSION có minos 12.0, kể cả platform macOS
hoặc simulator. Parser mới kiểm tra platform và deployment cùng nhau, từ chối
ARM64e/CPU khác, entry point không hợp lệ, và có fixture binary/payload sai được
chủ động tạo.

Một lỗi lựa chọn test được xác định: `gpu_darwin_test.go` dùng riêng `darwin`, nên
iOS chọn các symbol parser macOS không tồn tại. Nay dùng `darwin && !ios`, khớp
collector production. Không sửa collector production hay mã đồng thời; thay đổi
công cụ và constraint test này không cần thêm gate race.

## Kết quả platform và giới hạn

- iOS chọn battery_ios.go, system_platform_ios.go; loại battery macOS, Wi-Fi
  AppleScript và GPU macOS; chọn fallback Wi-Fi/GPU không hỗ trợ. Loại
  implementation cgroup CPU/memory/network. Mỗi nhóm platform đã biết có đúng
  một implementation.
- macOS giữ battery_darwin.go, wifi_darwin.go, gpu_darwin.go; loại metadata/battery
  iOS. CI hosted chạy test parser GPU và cả hai build Darwin bằng Go gốc chưa vá.
- Linux giữ battery, Wi-Fi, CPU/memory/network-counter và file cgroup Linux;
  loại file iOS, build cả hai entry point tại workspace.
- gopsutil v4.26.9 có chủ đích chọn file Darwin CPU/memory/disk/net/host/load và
  sensor ARM64 trên iOS. Đã kiểm tra constraint thật. Chọn Darwin không chứng
  minh khả năng chạy trên iOS: IOKit/CoreFoundation, sensor, disk I/O, syscall và
  command vẫn cần P008/P010/P018. Guard iOS hiện có bỏ qua probe model CPU Darwin.
  Không triển khai tính năng metrics, network-monitor, package-manager hay Wi-Fi
  mới trong P007.

## Hồi quy đã thực hiện

Validator matrix/readiness PASS (178 SHA upstream, 13 patch logic). Cả 49 audit
test PASS; 35 test cho hợp đồng runtime/build P007, gồm 15 biến thể source sai và
fixture CPU/platform/deployment/checksum/asset sai. Suite shell hiện có PASS:
upstream sync 10, sentinel iOS 44, bootstrap 54, installer 814.
`git diff --check` và actionlint PASS.

Lệnh chuẩn `env -u NETWORK go test -tags 'testing no_ui' ./...` PASS; chạy toàn bộ
thêm lần nữa với `-count=1` cũng PASS, không dùng kết quả test cache. Test tập
trung Agent/battery/Wi-Fi và cả hai build Linux PASS. Bun: 44 PASS, 0 FAIL; SHA-256
lock giữ nguyên
`16643e3228f9b0fac1e5134d89782c096f49e8071489cc2a4f977f3565f02afa`.
Toàn bộ 95 file asset production local có byte khớp trong Hub Linux thật. CI
native kiểm tra độc lập asset do chính job sinh ra trong Hub iOS.

Lần Go baseline đầu thất bại ở test nhạy với scheduling có sẵn
`TestConnectionManager_ReconnectionLogic`: goroutine reconnect có thể reset
isConnecting trước khi assertion tức thời đọc được. Test và mã connection
production không đổi so với SHA ban đầu; commit cuối của test là upstream
`4ffd83677d18fe285ce1647da9d64ed4b2798f40`. Suite chuẩn và suite không cache sau đó
PASS. Ghi nhận test này cho P009/P017; không sửa hay làm yếu test để đạt PASS.

## Bằng chứng hosted

Run chính xác, commit đã test, phiên bản, hash runtime, lựa chọn source, metadata
native và checksum artifact thật nằm trong
[`ios-build-validation.json`](ios-build-validation.json). Commit triển khai và
sửa validator là `7b09aa797fbfa636e05c67e85f7df307e474e237` và
`19c705c65a45b3da1c426e3c0222636fd244f3b0`. Guard provenance cuối nằm trong
`d85eb2cc6f139d7e03c98e8f793eddf193963d4d`; kết quả hosted được kiểm tra trên HEAD
cuối ghi trong PR.

Run native đầu
[38500460499](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38500460499)
build cả hai binary, vượt so sánh cache sạch, disassembly và platform native,
nhưng thất bại vì validator frontend mới giả định URL Vite bắt đầu bằng
`/assets/`. Base Vite thật của Beszel là `./`. Bản sửa chấp nhận và test URL
`./assets/`, embed đầy đủ và asset thiếu; không tính run thất bại là PASS P007.

[PR #9](https://github.com/nghianguyen150612/Beszel-iOS/pull/9) ghi HEAD feature
cuối, các lần iOS/installer chạy lại quyết định, hash chính xác và kiểm tra merge.
File bằng chứng ghi source candidate đã test; commit cuối không thể chứa hash
của chính nó. Input build đổi thì phải chạy lại hosted.

Output giống nhau từ cache độc lập chứng minh tái lập byte cho commit/SDK/
toolchain/frontend được ghi trong môi trường hosted đó. Hash giữa commit có
metadata VCS revision nên không nhất thiết giống nhau; không bao phủ SDK, runner
khác hay phiên bản tương lai.

Không gọi lệnh ký, tạo release/tag, deploy, cài thiết bị hay sửa database
production. Artifact native chỉ để review. A7-DEVICE-01 vẫn thuộc P018 / Gate F;
hồ sơ iPad 0.19 lịch sử giữ nguyên. P007 không hoàn tất Gate C và không bắt đầu
P008.
