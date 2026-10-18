# Đánh giá bảo mật bộ cài P016

## Phạm vi

Bộ cài rootful chỉ quản lý các đường dẫn iOS hiện có trong lifecycle engine: `/usr/local/bin`, `/var/lib/beszel-ios`, `/var/lib/beszel-agent`, `/var/lib/beszel-hub`, `/Library/LaunchDaemons` và thư mục log Beszel đã cấu hình. Bộ cài từ chối bố cục rootless. Đánh giá này bao gồm hành vi của bộ cài và dịch vụ; fixture trên cloud không chứng minh chữ ký iOS đáng tin cậy hoặc hành vi trên thiết bị.

## Tính toàn vẹn của bootstrap và artifact

`install.sh` chỉ tải lifecycle engine từ URL HTTPS chuẩn trên nhánh `iOS`, lưu tệp trong thư mục tạm riêng tư, kiểm tra SHA-256 gồm 64 ký tự rồi mới chạy bản sao cục bộ đó. Engine và pin tương ứng được cập nhật cùng nhau. Nếu tải, chuyển hướng, checksum hoặc ghi cục bộ thất bại, tệp tải xuống sẽ không được chạy. Manager đã cài được lưu từ engine cục bộ đã xác minh.

Pin bảo đảm bootstrap đã tải khớp từng byte với engine. Đây không phải chữ ký độc lập cho bootstrap: người có quyền thay bootstrap và pin ở nguồn có thể thay cả hai. Artifact phát hành được chọn từ một tag bất biến đã phân giải và đối chiếu với `SHA256SUMS` của release đó. Cách này phát hiện payload bị hỏng hoặc trộn lẫn, nhưng manifest không phải chữ ký phát hành độc lập.

Trước khi cài hoặc cập nhật, engine kiểm tra Mach-O mỏng, little-endian, ARM64 cơ bản, executable, có load command iOS device và deployment target không mới hơn iOS 12.0. Engine kiểm tra lại bản staging sau `ldid`. Đây là kiểm tra định dạng và target; nó không chứng minh `ldid` tạo chữ ký được iOS tin cậy hoặc iOS sẽ chạy binary. Artifact P015 dùng làm fixture dương lấy từ hosted run [39014286778](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39014286778), source `b83e179bd84b2177cd7fc8b5b0f5460d5f2411ba`, artifact `12093514420`. Checksum của cả hai artifact đều khớp; hosted validation ghi nhận Go 1.27.1, `GOOS=ios`, `GOARCH=arm64`, `CGO_ENABLED=1`, iOS platform 2, minimum iOS 12.0 và VCS metadata sạch.

## Đường dẫn rootful và kiểm soát filesystem

Thao tác thay đổi cần quyền root và bố cục rootful chuẩn. Trước khi dùng, engine kiểm tra các thành phần thư mục cha và thư mục quản lý để phát hiện symlink, thư mục bị thay thế bằng loại khác, quyền ghi cho group/world và owner bất thường. Snapshot dữ liệu từ chối symlink, special file, hardlink ngoài dự kiến, mode không an toàn và đường dẫn ngoài allow-list cố định. Thư mục staging tạm và backup giao dịch có quyền riêng tư. Lock nguyên tử tuần tự hóa các thao tác lifecycle; lock lỗi định dạng hoặc là symlink sẽ fail closed. Cleanup chỉ xóa thư mục công việc do chính lần chạy đó tạo.

Install state được ghi qua tệp tạm cùng thư mục rồi rename. State chỉ chứa release tag và SHA-256, không chứa SSH key hoặc thông tin xác thực. Gỡ cài thông thường giữ nguyên dữ liệu Agent và Hub. Purge dùng cơ chế xác nhận chính xác hiện có cho từng thành phần; bỏ trống, hủy hoặc lỗi không cho phép xóa.

Không thao tác nào đổi ownership đệ quy database của người dùng hoặc khóa Hub. Archive giao dịch ghi và xác minh owner, group, mode, loại, kích thước và hash tệp, sau đó chỉ khôi phục các đường dẫn trong allow-list.

## Lệnh chỉ đọc và bề mặt mạng

`status`, `diagnostics` và `doctor` không dùng lock thay đổi và không tải xuống, thao tác package, sửa database, đổi trạng thái dịch vụ hoặc ghi install state. Trạng thái public key của Agent chỉ được báo là đã cấu hình/chưa có; key không được in ra. Giá trị release state lỗi không bị echo. Giao dịch Hub đang chờ hoặc không hợp lệ được báo cáo mà không tự khôi phục; doctor trả lỗi khi cần recovery.

Hub và Agent tiếp tục dùng LaunchDaemon rootful và các cổng lắng nghe đã cấu hình. Kiểm tra request hiện có của Hub không thay đổi. Hub TLS production tiếp tục tắt đàm phán HTTP/2 và giữ giới hạn 16 byte ranges. Không thêm dependency hoặc nâng major toolchain.

## Signing và rủi ro còn lại

Bộ cài dùng thao tác `ldid -S` hiện có cho binary staging. Test cloud mock bước này và không khẳng định chữ ký iOS đáng tin cậy. P016 không thêm entitlement, thay đổi trust cache jailbreak, quyền Wi-Fi private hoặc liên kết MobileWiFi. Quyền runtime, chữ ký thật, khả năng duy trì LaunchDaemon sau khi kích hoạt jailbreak lại và hành vi reboot vẫn thuộc P018.

Hai rủi ro mức cao vẫn được giữ mở: advisory multipart Go có thể tới được `GO-2026-6608` trên toolchain Go 1.27.1 đang pin và `GHSA-vfj7-8cjw-p6xm` của `braces` chỉ dùng lúc build. Giảm thiểu HTTP/2 và Range hiện có không đổi. P016 không xóa các phát hiện này và không đánh dấu Gate E đạt; P017 vẫn phụ trách kiểm tra release candidate.
