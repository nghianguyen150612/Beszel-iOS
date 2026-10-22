# Ghi chú ứng viên Beszel-iOS `v0.21.0-ios.1` (bản nháp)

**Chưa được duyệt để đưa vào sản xuất. Chưa phát hành.** Tag đề xuất theo định dạng Beszel-iOS hiện có. Ứng viên tích hợp snapshot upstream đã ghim `b5357343550c8e2493d0e65472a06c1007396384` cùng các bản vá iOS gốc được bảo toàn. Tài liệu này không khẳng định upstream đã phát hành một bản ổn định mới hơn.

## Phạm vi ứng viên

- Tích hợp 41 commit upstream sau v0.21 trên snapshot đã ghim; đã đối soát 178 mục lịch sử upstream đóng băng, 35 commit iOS lịch sử và 13 nhóm bản vá iOS logic.
- Cung cấp bản dựng Agent và Hub ARM64 cho iOS, telemetry CPU, giám sát mạng và cập nhật cảnh báo TLS.
- Bao gồm thay đổi OAuth và phân quyền, cập nhật giao diện/địa phương hóa và migration cơ sở dữ liệu upstream.
- Dùng Go 1.27.2 với bảo vệ runtime Apple A7 đã được rà soát và bản vá bảo mật x509 chính thức; bản dựng ứng viên yêu cầu iOS 12.0 trở lên.
- Giữ trình cài đặt rootful `beszel-ios`, dữ liệu Agent/Hub riêng, vòng đời LaunchDaemon, kiểm tra checksum, cập nhật có staging và rollback dựa trên backup.

## Bảo mật và kiểm định

Ứng viên cần vượt qua quét advisory Go và JavaScript trên đúng mã nguồn, bộ kiểm thử Go và frontend, migration/khôi phục Hub lịch sử bằng dữ liệu tổng hợp thật, kiểm thử vòng đời trình cài đặt và kiểm định Agent/Hub iOS trên macOS/Xcode hosted. Không coi các kiểm định P022 là hoàn tất cho tới khi ghi nhận kết quả trên đúng PR head.

Các advisory còn mở hoặc cần xác minh gồm GO-2026-5932 và GHSA-vfj7-8cjw-p6xm; GO-2026-6809 có thể vẫn xuất hiện trong kết quả scanner trong khi bản vá x509 chính thức và các kiểm thử đã vượt qua. Cần quét lại GO-2026-6608 trên toolchain ứng viên chính xác.

## Giới hạn

- Không khẳng định đã kiểm tra tương thích hay chạy trên iPad vật lý. P018 Gate F vẫn PENDING.
- P022 không phê duyệt phát hành công khai, triển khai sản xuất hoặc dùng danh tính ký.
- Chẩn đoán Wi-Fi iOS vẫn được hoãn; ứng viên không hỗ trợ private MobileWiFi.
- Không hạ cấp cơ sở dữ liệu PocketBase đã migration chỉ bằng cách thay binary Hub; cần cold backup đã xác minh và phê duyệt khôi phục toàn bộ.
- Xác minh checksum và ký bằng `ldid` không chứng minh được độ tin cậy của chữ ký mã Apple.
