# Tương thích runtime Apple A7 — P007

P007 kiểm tra tương thích đích build và bản vá Go runtime. P007 không chứng minh
hai binary 0.21 chạy trên Apple A7 thật mà không có SIGILL. Hồ sơ iPad mini 2 /
Beszel 0.19 trong `device-validation.md` được giữ nguyên. A7-DEVICE-01 thuộc P018 /
Gate F.

## Baseline và lỗi phát hiện

Integration ban đầu: `6b14ebcdadc44641f4f84a5d7999ebf964905aa3`.
Nhánh sản phẩm `iOS`: `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`.
`main` đóng băng: `bb23f99b3eeaced9608e11e280e7c986dba650be`.
Cả 178 commit upstream chuẩn đều nằm trong ancestry. Nhánh làm việc là
`ios/a7-runtime-compatibility`; PR base là `ios/upstream-v021-integration`.

Bản vá cũ chỉ kiểm tra các chuỗi con theo thứ tự, không kiểm tra toàn bộ mã thực
thi của hàm. Lệnh NOP được thêm vào hoặc phép tính delay bị đổi vẫn có thể vượt
qua guard. Ghi trực tiếp có thể làm mất một phần file runtime khi thao tác ghi bị
ngắt. Vòng lặp thay thế thiếu guard cho giá trị đầu vào bằng 0: phép giảm có thể
tràn và quay 2^32 lần. Đây là các lỗi cụ thể của công cụ.

## Source được hỗ trợ và mã thay thế

Chỉ hỗ trợ **Go 1.27.1**, khớp `go.mod`. Hàm thật trong
`src/runtime/asm_arm64.s` được đối chiếu với fixture Go 1.27.1 đã có, gồm cả ranh
giới comment của hàm tiếp theo: giống nhau từng byte.
[Nguồn fixture](../tests/audit/fixtures/go-runtime-provenance.md) ghi nguồn
upstream, giấy phép BSD và SHA-256.

Script dùng `go env -json GOROOT GOVERSION`, yêu cầu GOROOT tuyệt đối hợp lệ và
file runtime thường, không phải symlink. Chỉ được có một marker bắt đầu và một
marker kết thúc. Mọi dòng thực thi, toán hạng, nhãn và directive phải khớp hàm đã
review; chỉ chuẩn hóa comment và khoảng trắng. Hàm thêm vào, lệnh lạ, phép tính
hay chuỗi truy cập counter bị đổi đều bị từ chối. Phần trước và sau hàm giữ nguyên
từng byte; không thay toàn bộ file assembly bằng template.

Mã thay thế đọc đối số delay 32-bit, trả về ngay khi bằng 0; với giá trị dương,
thực hiện YIELD, giảm một lần và lặp đến 0. Không còn truy cập CNTFRQ_EL0 hoặc
CNTVCT_EL0. Cách này giữ vòng spin hữu hạn theo phương án tương thích A7 lịch sử.
Vòng counter gốc của Go 1.27.1 đo nanosecond; số lần YIELD không phải đồng hồ
nanosecond đã hiệu chuẩn, nên không tuyên bố thời gian hay hiệu năng giống nhau.
Polling, tranh chấp và độ bền dịch vụ trên thiết bị thật vẫn thuộc P018.

Toàn bộ xác minh hoàn tất trước khi tạo file tạm cùng thư mục. Script flush và
fsync dữ liệu mới, giữ quyền file, kiểm tra input chưa đổi rồi rename nguyên tử.
Lỗi trước rename giữ nguyên input và dọn file tạm. Không bảo đảm chống writer
đồng thời bất kỳ sau kiểm tra cuối; CI sở hữu riêng bản sao theo job. Lần gọi thứ
hai cố ý thoát khác 0 và không ghi. Diagnostic phiên bản không hỗ trợ ghi rõ bản
mong đợi và bản quan sát được.

## Cách ly toolchain

CI sao chép toàn bộ GOROOT từ setup-go vào `RUNNER_TEMP/go-ios-a7`, giữ `bin`,
`pkg`, `src` và cấu trúc đúng. Toolchain sao chép phải báo đúng GOROOT đó và bản Go
được module pin. PATH, GOROOT và GOTOOLCHAIN=local chọn bản sao cho **cả hai** build
iOS lẫn build kiểm tra. GOCACHE iOS riêng tránh dùng runtime object của host.
Không dùng sudo và không sửa cài đặt setup-go. Hash source runtime gốc được kiểm
tra sau patch và sau cả hai build native. Build hồi quy macOS dùng toolchain chưa
patch và chạy trước bước cách ly.

Mọi test âm dùng fake GOROOT tạm. Một bản sao tạm riêng của toolchain Linux thật
cũng vượt qua patch, từ chối lần gọi thứ hai và bảo toàn source gốc. macOS hosted
vẫn là bằng chứng quyết định cho link iOS native bằng toolchain sao chép.

## Bằng chứng test và binary

13 runtime unit test kiểm tra fixture chuẩn, toàn bộ output và guard giá trị 0;
15 biến thể source sai; thiếu/sai GOROOT, source hoặc lệnh go; go env thất bại,
JSON sai và bản Go không hỗ trợ; từ chối symlink; gọi lặp lại; lỗi flush/rename
được chủ động tạo. Lần thất bại giữ nguyên byte input, file không liên quan và
không để lại output tạm.

`verify-ios-runtime-binary.py` kiểm tra đúng một symbol
`runtime.procyieldAsm[.abi0]` trong các build Agent/Hub kiểm tra không strip. Mọi
word lệnh ARM64 và độ dời nhánh đều phải đúng; sau RET chỉ cho phép zero căn chỉnh.
Symbol thiếu/đã strip, lệnh thêm và biến thể đọc counter đều thất bại. Cờ
production `-s -w` được giữ. Kiểm tra source, fixture đã assemble và binary đã
link là ba loại bằng chứng riêng; không loại nào thay thế chạy trên A7 thật.

URL hosted, SHA source đã test, hash runtime và kết quả nằm trong
[`ios-build-validation.json`](ios-build-validation.json) và
[PR #9](https://github.com/nghianguyen150612/Beszel-iOS/pull/9). PR ghi HEAD cuối có
cả tài liệu và lần CI chạy lại, tránh hash commit tự tham chiếu trong file này.

P007 không hoàn tất Gate C: P008–P012 vẫn phụ trách telemetry, transport,
collector hỗ trợ, package update và Wi-Fi. P018 phụ trách thiết bị thật.
