# P015 frontend assets và khả năng tương thích

## Kết quả

P015 đã được merge tại integration commit `50546d7f7468f7b07af347eb157139208b04fcbc`. Follow-up source commit `60df517928b74ac3f0230dbe0d72100bbd952d94` xử lý thêm advisory dependency và hoàn thiện kiểm tra login/catalog/browser. Kiểm tra local cho follow-up đã đạt; hosted CI và artifact verification cho commit này đang chờ. Artifact hosted trước đó chỉ áp dụng cho source đã merge trước follow-up.

Ứng viên dùng Go 1.27.1 và Bun 1.4.0. Hai chu kỳ build sạch đã xóa node_modules và dist bị ignore trước khi cài bằng Bun lock cố định. Cả hai chạy Lingui extract/compile, TypeScript và Vite production build, tạo ra manifest giống hệt nhau.

- 95 tệp không rỗng; tổng cộng 3.357.403 byte.
- SHA-256 manifest: 63dc41212f5029b2cb07cfbda5f3ddaa4ea71b72f32e39f0032f870c06286df4.
- SHA-256 Bun lock: e2d090fa8b5bcffd8e2de003262cfe6c506c297a676279fbb08d1c2db3bc6dbf; cài đặt frozen không sửa lock.
- SHA-256 npm lock được giữ lại: 6fbd1027efc8368462a78a0ff87dae23b758eee759ba56172eb862f70df1b3d5; không thay đổi. CI dùng Bun, không chạy npm ci.
- Cả 32 module locale đã đăng ký đều biên dịch và có trong asset được tạo.
- Lingui báo cáo 537 chuỗi nguồn. Tiếng Việt có 529 chuỗi đã dịch và 8 chuỗi còn thiếu; không sửa nội dung dịch.

Manifest đầy đủ nằm trong `ios-frontend-assets-manifest.sha256`. Bộ kiểm tra xác thực đầu ra không rỗng, index, tham chiếu JS/CSS, module và chunk locale, đường dẫn tương đối an toàn, cùng manifest cũ hoặc thiếu. Bộ kiểm tra đã đạt với dist của follow-up. Kiểm tra Go embed cũng đạt với dist và manifest đó.

## Hub ứng viên

Một Hub Linux được khởi chạy với cơ sở dữ liệu PocketBase dùng một lần. Hub thứ hai được kiểm tra tại tiền tố /beszel.

- 94/95 tệp khớp SHA-256. Hub chủ động sửa index.html để chèn phiên bản và cấu hình base path.
- Kiểm tra MIME đạt: 88 JavaScript, 1 CSS, 2 font WOFF2, 1 PNG, 1 SVG và 1 JSON.
- Chín route SPA được hỗ trợ trả về 200 tại root và /beszel.
- Cả hai base path chèn đúng BASE_PATH.
- Đường dẫn không hỗ trợ như .env, .git/config, go.mod và asset thiếu trả về 404.
- Các collection chưa xác thực trong DB mới trả về danh sách rỗng.
- Yêu cầu có 17 byte range trả về 400; tối đa 16 được chấp nhận.
- Static asset trả về cache policy công khai đúng yêu cầu.

Browser harness hiện tại đạt ở cả Hub root và `/beszel` qua loopback reverse proxy. Chỉ dùng tài khoản tổng hợp và dữ liệu dùng một lần. Không dùng iPad vật lý. Chi tiết route và auth nằm trong `ios-frontend-route-matrix.md`.

## Sửa đổi tập trung

Frontend chỉ làm mới xác thực khi có token, xóa trạng thái đăng nhập hết hạn hoặc bị từ chối và tránh in response body hoặc token vào log. Chỉ tạo user settings sau phản hồi 404 thực sự; phản hồi 401/403 được xử lý mà không ghi dữ liệu nhạy cảm. Lỗi lấy thông tin Hub được bắt và chỉ ghi status.

Hub tắt HTTP/2 server ở TLS production và tắt mặc định HTTP/2 client Go 1.27.1 trong executable Hub và Agent. Kiểm tra TLS chào mời h2 và HTTP/1.1 với GODEBUG=http2server=1, rồi xác nhận HTTP/1.1 được chọn. Middleware Hub giới hạn byte range ở 16; unit test và HTTP probe xác nhận ranh giới.

Các cảnh báo Vite hiện có về dynamic/static import chồng lấn và chunk WASM/vendor lớn vẫn còn. Build thành công.

## Trạng thái Gate

Trạng thái Gate D của follow-up: **đang chờ hosted CI và xác minh artifact trên source hiện tại**. Source P015 trước follow-up đã qua hosted checks; các kết quả đó không bao gồm commit `60df517`. Rủi ro dependency mức cao vẫn được ghi trong `ios-dependency-advisory-review.md`. Kiểm thử vật lý P018 vẫn chờ; chưa cho phép phát hành.
