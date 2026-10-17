# P014 — Tương thích giám sát Hub, cảnh báo và telemetry

Trạng thái: **kiểm thử đạt; đang chờ hợp nhất thông thường**. Ứng viên mã nguồn
hiện tại là `bf9c4d4b3c8910ab4e2546fbbf5d4e75c93864b6` trên nhánh
`ios/hub-monitor-alert-rebuild`, PR [#17](https://github.com/nghianguyen150612/Beszel-iOS/pull/17),
nhắm tới `ios/upstream-v021-integration`. Nhánh bắt đầu từ P013
`00c14bbf2756213cc2afef828ef7d5b9af26f1d6`. PR #16 đã đóng, chưa hợp nhất và
không có commit nào của PR đó trong nhánh này.

## Luồng dữ liệu từ Agent từ xa

Các fixture tổng hợp, được tạo riêng từ cấu trúc mã nguồn, đại diện Linux,
Windows, macOS và iOS khi collector tùy chọn không khả dụng. Peer WebSocket và
SSH tổng hợp chạy qua bộ giải mã, polling và xử lý Hub thật. Kiểm thử lưu dữ liệu
vào PocketBase dùng một lần, truy vấn chi tiết và lịch sử bằng API có xác thực,
kiểm tra từ chối người dùng khác, kết nối lại và xóa hệ thống. Đây không phải là
bằng chứng Agent Windows/macOS hoặc iPad đã chạy native.

Hub vẫn giữ các trường riêng của Linux, Windows và macOS dù Agent iOS cục bộ bỏ
qua collector không hỗ trợ. JSON v1/v2, mảng byte CBOR cũ và các hành động riêng
cho SMART, ZFS, systemd và package vẫn tương thích. Chi tiết có trong
[ma trận telemetry](ios-remote-agent-telemetry-matrix.md).

## Lỗi đã tái hiện và sửa

- Bộ giải mã Hub từ chối mảng CPU hiện tại và tuple pin base64 cũ. Bộ giải mã mới
  hỗ trợ cả hai dạng, từ chối giá trị tràn và tuple sai, đồng thời giữ nguyên định
  dạng JSON/CBOR hiện tại.
- Rollup lịch sử chia telemetry thiết bị tùy chọn cho mọi bản ghi. Bản sửa dùng số
  quan sát riêng cho từng trường/thiết bị, giữ số không đo được và bỏ qua filesystem
  null. Công suất GPU và công suất package có trạng thái hiện diện riêng.
- Thiếu GPU, nhiệt độ hoặc filesystem có thể làm cảnh báo hồi phục giả hoặc tạo số
  liệu disk. Cảnh báo đợi mẫu tùy chọn hợp lệ; mẫu mới bằng không vẫn có thể hồi phục.
- Kết quả monitor cũ hoặc chậm có thể ghi đè dữ liệu mới hay tạo lại bản ghi đã xóa.
  Bộ đếm mẫu, kiểm tra cấu hình và quyền sở hữu trong transaction ngăn kết quả cũ,
  đồng thời vẫn chấp nhận đồng hồ Agent quay ngược và peer cũ không có bộ đếm.
- Poll bị hủy hoặc thay thế có thể ghi sau khi hệ thống cũ đã nghỉ. Kiểm tra context
  và chờ updater đảm bảo snapshot cũ không được commit. Không đổi schema/migration.
- Frontend để trống chỗ dữ liệu không khả dụng, giữ số không thực, hỗ trợ đơn vị
  MiB/s cũ và chịu được GPU, disk, network, filesystem, pool hoặc Wi-Fi thiếu/null.

## Kết quả xác thực

Canonical Go suite đạt trên mã backend `3f153dca940651ab394663b9d85d14cbde64b957`.
Commit tiếp theo `ca7108b7` chỉ định dạng TypeScript/TSX đã sửa, không thay đổi Go.
Các nhóm race trên Agent, Wi-Fi, entity, Hub, systems, transport, records và alerts
đều đạt; các regression P014 trọng yếu được chạy lặp ba lần. Race cũ của
`TestAgentWebSocketIntegration` được tái hiện trên P013, nguyên nhân là test đọc
connection state không dùng mutex. Test mới dùng read-lock sẵn có và đạt ba lần;
transport production không bị đổi.

Bun 1.4.0 frozen install, tạo/biên dịch catalog, TypeScript và production build đạt.
55 frontend tests với 267 assertion đạt; build có 95 asset nhúng. Ma trận/readiness,
54 Python audit tests, upstream sync, patch sentinel, bootstrap và installer đều đạt.
Linux Agent/Hub build và biên dịch test Hub Windows/transport FreeBSD đều đạt. Biên
dịch chéo không được coi là chạy native.

Native iOS CI đạt trên source `3f153dca940651ab394663b9d85d14cbde64b957`:
[iOS Build](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38941248597),
[installer lifecycle](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38941249967),
và [P013 Hub migration fixture](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38941249970).
Sau commit định dạng frontend, Bun tests/build đã chạy lại và Hub native được kiểm
tra độc lập với chính bộ asset frontend hiện tại. Hai binary là Mach-O arm64,
iOS platform 2, minimum iOS 12.0; kiểm thử Go 1.27.1 A7, asset nhúng, SHA256SUMS
và liên kết sản xuất đều đạt. Hash và provenance nằm trong
[bằng chứng cuối](evidence/p014/final-validation.md) và JSON xác thực.

## Phạm vi còn lại

`IOS-HUB-05`, `R-REMOTE-OS-TELEMETRY` và phần data-model P014 của
`R-IOS-FRONTEND-API` đã được xác minh. `IOS-HUB-04` kiểm thử route và localization
thuộc P015, nên Gate D vẫn mở. RSSI Wi-Fi iOS native vẫn
**HOÃN — THIẾU THIẾT BỊ THAM CHIẾU**. P018 tiếp tục phụ trách chạy trên iPad/A7.
Không thay đổi thiết bị, cơ sở dữ liệu production, signing, entitlement, release,
tag hoặc deployment.
