# P014 — Xây dựng lại kiểm thử giám sát Hub

Trạng thái: ĐANG THỰC HIỆN. PR mới là #17, nhánh
`ios/hub-monitor-alert-rebuild`, bắt đầu từ P013
`00c14bbf2756213cc2afef828ef7d5b9af26f1d6`. PR #16 đã đóng, chưa hợp nhất và
bị hủy bỏ. Không nhập lại mã hay sử dụng kết quả kiểm thử cũ làm bằng chứng mới.

Bốn bộ dữ liệu tổng hợp độc lập mô tả Linux, Windows, macOS và iOS. Kiểm thử
chạy trên Linux, không chứng minh Agent đã chạy trực tiếp trên Windows/macOS
hoặc iPad. Dữ liệu không có bộ thu thập tùy chọn phải giữ trạng thái không khả
dụng; giá trị đo bằng không vẫn hợp lệ.

Đã tái hiện lỗi JSON v2 từ chối mảng số `cpus` và JSON v1 từ chối chuỗi byte
base64 `bat` cũ. Bộ giải mã mới hỗ trợ cả hai dạng, kiểm tra giới hạn byte và
độ dài bộ đôi pin. Kiểm thử entity và dữ liệu nền đã đạt. Định dạng CBOR và
JSON số đầu ra không thay đổi.

Biểu đồ công suất GPU trước đây gộp thiết bị trùng tên và hiển thị dữ liệu thiếu
thành số không. Bản sửa phân biệt định danh thiết bị, giữ số không thực và để
trống khi không có phép đo. Kiểm thử Bun tập trung đã đạt.

Các bước lưu trữ, lịch sử, cảnh báo, monitor, API, race và CI iOS vẫn đang được
thực hiện. Chưa công bố P014 PASS. Gate D vẫn cần P015. Không triển khai P015.

RSSI Wi-Fi iOS gốc vẫn hoãn vì không có thiết bị tham chiếu. Không thay đổi
iPad, dữ liệu thật, nhánh iOS/main được bảo vệ, chữ ký, entitlement, LaunchDaemon,
cấu hình Wi-Fi, bản phát hành hoặc triển khai.

Xem chi tiết tại [sổ checkpoint](ios-p014-rebuild-checkpoints.md) và
[kết quả kiểm thử](ios-hub-monitoring-validation.json).
