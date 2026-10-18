# P017 — Release candidate Beszel-iOS v0.21.0

**Gate E: PASS đối với kiểm thử trên cloud.** P016 đã merge bình thường tại `3ed79af25a03381f4c1b8800fd7cb93e3480403d`. P017 chỉ sửa workflow CI và tài liệu; không thay đổi mã sản phẩm hoặc installer. Source thực sự tạo binary là `5a7351f752376e206a3e1809b3698bb03cc87b98`.

Các workflow [P017 regression](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39095687534), [iOS Build](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39095687398), [installer](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39095687704) đều PASS; release job SKIPPED. CI tạo frontend thật với 95 assets. Đã tải các ZIP artifact và đối chiếu digest GitHub; SHA256SUMS của Agent/Hub PASS; Mach-O ARM64, platform iOS, minOS 12.0; kiểm tra runtime A7 không đọc counter gây SIGILL; không liên kết private MobileWiFi.

[P016 genuine P013 fixture](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39028505022) PASS trên source P016 chưa bị P017 thay đổi mã sản phẩm. Chưa chạy lại fixture đó trên commit chỉ sửa tài liệu/workflow P017.

Hai advisory HIGH **vẫn chưa được vá**: `GO-2026-6608` (multipart MIME, Go 1.27.1) và `GHSA-vfj7-8cjw-p6xm` (braces trong công cụ build). Giữ chính sách tắt HTTP/2, giới hạn Range và chỉ triển khai trong mạng đáng tin cậy; phải chấp thuận rủi ro rõ ràng trước khi phát hành.

**Gate F / P018: PENDING** cho iPad mini 2 A1489, iOS 12.5.7 Amethyst/Procursus rootful. Chưa thử chạy native, LaunchDaemon/reboot, metrics, kết nối hay rollback thực tế của bản v0.21.0. Native Wi-Fi RSSI vẫn DEFERRED. Chưa tạo tag/release hoặc cài lên thiết bị.

Theo yêu cầu mới của chủ repo, P017 có ngoại lệ owner-directed cho quy trình independent review; không ghi thành human approval. Phải dùng merge commit bình thường và tuân thủ branch protection thực tế.
