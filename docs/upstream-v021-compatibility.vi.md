# Ma trận tương thích upstream v0.21

Dữ liệu tình huống chuẩn nằm trong
[`upstream-v021-compatibility-matrix.csv`](upstream-v021-compatibility-matrix.csv).
Tệp ghi lại tổ hợp phiên bản, hành vi, phụ thuộc giao thức, môi trường kiểm
thử, bằng chứng, trạng thái, chủ sở hữu và điều kiện chặn. Trạng thái `planned`
là yêu cầu kiểm thử, không phải bằng chứng cổng iOS 0.21 hoạt động. P003 chưa
biên dịch Agent hoặc Hub iOS đã nâng cấp.

## Bằng chứng và trạng thái

Mã nguồn upstream đã đóng băng là
`bb23f99b3eeaced9608e11e280e7c986dba650be`; bản audit P002 là chỉ mục theo tệp.
Mã nguồn iOS hiện tại là baseline đã xác minh
`7e29c185458f79be274bb09ca57a0be9fc1b63f7`. Bằng chứng thiết bị thực chỉ áp
dụng cho binary iOS 0.19.0.

* **historically validated** — hồ sơ P002/thiết bị cho thấy hành vi 0.19.0 trên
  iPad tham chiếu; không phải kết quả 0.21.
* **planned** — mã nguồn chỉ ra hợp đồng dự kiến nhưng chưa chạy binary hoặc
  phép thử nâng cấp.
* **unsupported** — chưa tuyên bố có triển khai cho iOS; phải để trống số đo
  hoặc dùng no-op rõ ràng cho đến khi chủ sở hữu cung cấp adapter.
* **blocked** — điều kiện tiên quyết chưa cho phép kiểm thử có ý nghĩa.
* **verified** — chỉ dùng khi đã hoàn tất kiểm thử đúng cặp phiên bản.

## Hợp đồng phiên bản Agent và Hub

Upstream v0.21 khai báo `MinVersionNetworkMonitors = 0.20.0` trong
`beszel.go`. Chuỗi phiên bản này không đảm bảo mọi khả năng tương thích
Agent/Hub. `internal/hub/systems/network_monitors.go` kiểm tra phiên bản Agent
đang kết nối trước khi đồng bộ monitor; nếu Agent cũ hơn 0.20.0, Hub bỏ qua
thao tác đồng bộ. Đường thống kê thường có các kiểm tra riêng:
`MinVersionAgentResponse = 0.13.0` và `MinVersionCbor = 0.12.0`. Các mốc này
chi phối xử lý phản hồi/yêu cầu và encoding, không xác nhận hỗ trợ network
monitor. Cần kiểm thử phiên bản không rõ hoặc không phân tích được qua handshake
và fallback thực tế.

Ma trận bắt buộc bao gồm Agent/Hub iOS 0.19, Agent iOS 0.19 với Hub upstream
0.21, Agent iOS mới với Hub 0.19 hiện hữu, cặp iOS 0.21, Agent Linux/Windows/macOS
0.21 với Hub iOS 0.21, và Agent cũ từ xa với Hub iOS đã nâng cấp. Cho đến khi
P007–P018 tạo binary và chạy kiểm thử, mọi tổ hợp binary mới vẫn ở trạng thái
planned.

## Giao thức và mạng

CSV theo dõi WebSocket handshake và header xác thực, SSH fallback, kết nối lại
sau lỗi kết nối/handshake, keepalive và giới hạn peer im lặng, quyền sở hữu và
khóa kết nối, cancellation, deadline, đàm phán phiên bản, đồng bộ lại monitor,
CBOR, JSON v2 và phân biệt trường bị thiếu với giá trị zero. `agent/client.go`
kiểm tra `HUB_URL`, đọc một token từ `TOKEN` hoặc `TOKEN_FILE`, phân tích
`HUB_HEADERS`, bỏ qua `X-Token` và `X-Beszel` do người dùng truyền vào, đồng
thời bổ sung CA từ `CA_CERT_FILE` vào trust pool hệ thống. Mặc định vẫn bật xác
thực TLS. Tùy chọn `SkipTLSVerify` cho từng monitor là ngoại lệ riêng, chỉ được
bật rõ ràng và phải kiểm thử riêng.

Thay đổi `omitzero` trong entity upstream thuộc hợp đồng JSON v2. Kiểm thử phải
so sánh JSON v1/v2 và CBOR với trường thiếu, zero rõ ràng, false và dữ liệu rỗng.
Struct phản hồi mới phải thay thế dữ liệu cache để trường bị bỏ qua không làm
lộ giá trị khác không từ lần trước.

## Giới hạn nền tảng iOS

P007 phải giữ `GOOS=ios`, `GOARCH=arm64`, `CGO_ENABLED=1`, iPhoneOS SDK và metadata
deployment tối thiểu iOS 12.0. Bản vá A7 chỉ sửa đúng hình dạng đã biết của
`runtime.procyieldAsm` ARM64 trong Go; hình dạng lạ phải bị từ chối. Chỉ kiểm
tra mã nguồn hoặc link thành công không chứng minh runtime an toàn trên A7.

P008 phụ trách metadata iOS và kiểm thử CPU/RAM/disk/battery/network counters.
Parser pin hiện tại đọc `AppleARMPMUCharger`/`ioreg -l` của iOS 12 và ánh xạ
trạng thái kết nối nguồn, sạc, đầy và cạn. Không thay bằng cách đọc
`AppleSmartBattery -a` của macOS. Có bằng chứng lịch sử cho 0.19 trên iPad tham
chiếu; thu thập dữ liệu 0.21 vẫn là planned cho đến khi được build và chạy.

P010 phải loại trừ hoặc cung cấp no-op/adapter được hỗ trợ cho collector
Linux/macOS/Windows không có trên iOS, gồm GPU, SMART, ZFS, systemd và probe
network-monitor theo nền tảng. Số đo không có phải giữ absent; không giả lập
zero. P011 phụ trách adapter cập nhật gói Procursus/APT chỉ đọc, có timeout giới
hạn, không refresh danh sách gói hoặc cài đặt. P012 phụ trách fallback Wi-Fi
RSSI. Collector Wi-Fi macOS upstream gọi `/usr/bin/osascript` với JXA/CoreWLAN;
đây không phải triển khai iOS. Phải loại iOS khỏi đường build đó cho đến khi
adapter native được xác minh.

iOS Hub cũng phục vụ Agent Linux, Windows và macOS từ xa được hỗ trợ. Việc loại
collector cục bộ của iOS không được loại bỏ khả năng Hub phân tích, lưu và hiển
thị telemetry từ xa.

## Kiểm thử bắt buộc và cổng phát hành

Mỗi dòng liên kết đến acceptance case trong
[`upstream-v021-acceptance.md`](upstream-v021-acceptance.md), hoặc fixture kiểm
thử giao thức/Hub/Agent của P017. Các dòng migration vẫn chặn nâng cấp production
cho đến khi phép thử migration và restore dùng dữ liệu disposable từ 0.19 lên
0.21 đạt yêu cầu. Kiểm thử runtime A7 và dịch vụ trên thiết bị thực thuộc P018.
Không dòng nào của P003 tuyên bố đã có build hoặc kết quả thiết bị mới.
