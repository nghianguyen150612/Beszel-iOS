# Beszel-iOS

Chạy Beszel Agent và Hub trực tiếp trên iPhone hoặc iPad đã jailbreak — biến thiết bị iOS cũ thành một nút giám sát gọn nhẹ hoặc một máy chủ tự quản.

[![License: MIT](https://img.shields.io/github/license/nghianguyen150612/Beszel-iOS)](LICENSE)
![Beszel base 0.19.0](https://img.shields.io/badge/Beszel%20base-0.19.0-blue)
![Tested on iPad mini 2 · iOS 12.5.7](https://img.shields.io/badge/tested-iPad%20mini%202%20%C2%B7%20iOS%2012.5.7-green)

[Beszel](https://github.com/henrygd/beszel) là một nền tảng giám sát máy chủ nhẹ, tự quản. Bạn cài một **Agent** nhỏ trên mỗi máy cần theo dõi, còn **Hub** sẽ thu thập số liệu và hiển thị trên bảng điều khiển web — CPU, bộ nhớ, ổ đĩa, mạng và nhiều thông số khác. Dự án này là **bản port cộng đồng không chính thức**, chạy cả Agent lẫn Hub trực tiếp trên iOS đã jailbreak, không cần máy ảo Linux hay container.

> Dự án gốc: **Beszel của henrygd** — <https://github.com/henrygd/beszel>
>
> Đây là bản port cộng đồng, không liên kết và không được dự án gốc chứng thực.

## Vì sao dùng Beszel-iOS?

Bạn còn một chiếc iPhone hay iPad cũ nằm trong ngăn kéo? Nếu máy đã jailbreak, nó vẫn còn dùng được.

Beszel-iOS giúp bạn tận dụng thiết bị đó như một máy nhỏ luôn bật: chạy Agent giám sát trên máy, xem số liệu từ bảng điều khiển Beszel quen thuộc, và — nếu muốn — host luôn bảng điều khiển ngay trên chính chiếc iPhone/iPad đó. Mọi thứ đều là binary arm64 chạy trực tiếp trên iOS và sẽ tự khởi động cùng hệ thống khi môi trường jailbreak đã hoạt động.

Nói ngắn gọn: iPad cũ vào, biểu đồ hệ thống trực tiếp ra.

## Tính năng

- **Agent Beszel chạy trực tiếp trên iOS** — báo cáo CPU, bộ nhớ, ổ đĩa, mạng, tải hệ thống và thông tin thiết bị về bất kỳ Hub Beszel nào.
- **Hub Beszel chạy trực tiếp trên iOS** — phục vụ bảng điều khiển web quen thuộc của Beszel và lưu lịch sử ngay trên thiết bị.
- **Theo dõi pin** — hiển thị phần trăm pin và trạng thái sạc ngay cạnh các thông số máy chủ thông thường.
- **Tự khởi động** — Agent và Hub chạy dưới dạng dịch vụ hệ thống và sẽ quay lại sau khi bạn kích hoạt jailbreak.
- **Hub + Agent trên cùng một máy** — một chiếc iPhone/iPad vừa tự giám sát chính nó, vừa host bảng điều khiển của mình.
- **Bộ cài đặt một lệnh** — cài đặt, cập nhật, sửa lỗi, cấu hình lại và gỡ bỏ từ một menu đơn giản. Các bản tải về đều được kiểm tra checksum, bản cập nhật luôn giữ bản sao lưu với khả năng tự quay lại, dữ liệu của bạn được giữ nguyên.
- **An toàn theo mặc định** — gỡ cài đặt thông thường chỉ gỡ ứng dụng nhưng giữ lại dữ liệu; muốn xóa dữ liệu luôn phải xác nhận rõ ràng.
- **Bám sát bản gốc** — giữ nguyên thiết kế và giao thức Agent + Hub của Beszel, dựa trên bản gốc 0.19.0.

## Bắt đầu nhanh

> **Trước khi bắt đầu:** hãy chắc thiết bị của bạn đáp ứng [Điều kiện](#điều-kiện) và [Khả năng tương thích](#khả-năng-tương-thích) bên dưới. Quan trọng nhất là máy phải đã jailbreak và bạn có quyền truy cập SSH hoặc terminal.

SSH vào chiếc iPhone/iPad đã jailbreak, rồi chạy:

```sh
curl -fsSL https://raw.githubusercontent.com/nghianguyen150612/Beszel-iOS/iOS/install.sh | sudo sh
```

Chỉ một lệnh, không cần clone hay tải thêm gì khác. Sau đó:

1. Chọn **Agent**, **Hub** hoặc **Agent + Hub** trong menu.
2. Làm theo hướng dẫn (với Agent bạn cần public key của Hub; với Hub bạn chọn một cổng).
3. Nếu bạn cài Hub, hãy mở địa chỉ web của nó để tạo tài khoản. Nếu bạn cài Agent, hãy thêm nó vào Hub như bình thường.

Về sau muốn làm gì — cập nhật, kiểm tra trạng thái, sửa lỗi, cấu hình lại hay gỡ bỏ — bạn chỉ cần chạy lại đúng lệnh trên rồi chọn mục muốn dùng.

## Điều kiện

- Một chiếc iPhone, iPad hoặc iPod touch **arm64** đã jailbreak.
- Môi trường jailbreak còn hoạt động với các công cụ Unix chuẩn (`curl`, `launchctl`), có mạng, và có công cụ kiểm tra SHA-256 (`sha256sum`, `shasum` hoặc `openssl`).
- Có terminal ngay trên máy, hoặc truy cập SSH vào máy.
- Có quyền chạy lệnh dưới quyền root (`sudo`) để cài đặt.

Cấu hình đã được kiểm thử là môi trường **Amethyst + Procursus** (xem [Khả năng tương thích](#khả-năng-tương-thích)). Các môi trường jailbreak khác có thể vẫn chạy được, nhưng chưa được kiểm thử thực tế ở mức tương đương. Nếu thiếu `ldid` (công cụ ký binary iOS), bộ cài đặt sẽ hỏi bạn có muốn cài thêm hay không.

## Khả năng tương thích

Mục này dùng ba mức trạng thái khác nhau, bạn đừng nhầm lẫn:

- **Đã kiểm thử thực tế** — dự án đã thử trực tiếp trên thiết bị thật.
- **Kỳ vọng tương thích (chưa kiểm thử)** — về mặt kỹ thuật có khả năng chạy được dựa trên cách build và cách viết bộ cài đặt, nhưng chưa thử trên máy thật.
- **Không hỗ trợ** — với binary và bộ cài đặt hiện tại thì không chạy được.

### Phần cứng đã kiểm thử

Chỉ có một cấu hình đã hoàn thành toàn bộ bài kiểm thử trên máy thật (cài Agent, cài Hub, cài Agent + Hub cùng lúc, chẩn đoán, sửa lỗi, cấu hình lại, gỡ an toàn, cài lại giữ nguyên dữ liệu, kết nối lại, đo pin, khởi động lại, kích hoạt jailbreak lại, dịch vụ tự quay lại):

| Thiết bị | Model | SoC | Hệ điều hành | Jailbreak / Bootstrap | Trạng thái |
| --- | --- | --- | --- | --- | --- |
| iPad mini 2 | iPad4,4 / A1489 | Apple A7 | iOS 12.5.7 | Amethyst + Procursus (semi-untethered) | **Đã kiểm thử thực tế** |

Bằng chứng đầy đủ được ghi trong [Kiểm thử thiết bị thực tế](docs/device-validation.vi.md).

### Kiến trúc và phiên bản iOS mục tiêu

Các binary hiện tại được build cho:

- Kiến trúc: **arm64** (`GOOS=ios`, `GOARCH=arm64`).
- Phiên bản iOS tối thiểu khi build: **iOS 12.0** (lúc build truyền `-mios-version-min=12.0` và có bước kiểm tra metadata iOS trong binary).

Nói dễ hiểu: binary được *build cho* arm64 với mốc tối thiểu iOS 12.0. Các thiết bị arm64 đã jailbreak khác, đáp ứng được mốc đó, *có thể sẽ tương thích*, nhưng **chưa** được kiểm thử thực tế ở mức như trên. Mốc deployment target tối thiểu không phải là lời hứa hỗ trợ mọi bản iOS về sau hay mọi chip mới — nó chỉ cho biết binary được build dựa trên mốc nào.

Xem [Ghi chú build iOS](docs/ios-build-notes.vi.md) để biết cấu hình build chính xác.

### Máy nào thì thử được?

#### Đã kiểm thử

- Đúng cấu hình iPad mini 2 trong bảng trên. Nếu máy của bạn giống hệt như vậy, bạn đang đi trên con đường đã được kiểm thử.

#### Kỳ vọng tương thích (chưa kiểm thử thực tế)

Dựa trên những gì bản build và bộ cài đặt thực sự yêu cầu, các máy sau *có thể* chạy được nhưng **chưa được kiểm thử thực tế**:

- iPhone, iPad hoặc iPod touch arm64 đã jailbreak.
- Đang chạy iOS bằng hoặc cao hơn mốc tối thiểu của binary (iOS 12.0).
- Chạy được binary arm64 ký bằng `ldid`, ngoài sandbox của App Store.
- Cho phép truy cập root / `sudo` khi cài đặt.
- Hỗ trợ `launchd` / LaunchDaemon thông qua `launchctl`.
- Có môi trường jailbreak ghi được tại các đường dẫn truyền thống mà bộ cài đặt dùng (xem bên dưới).
- Có `curl`, có mạng tới GitHub releases, có công cụ SHA-256 và có `ldid` (hoặc có nguồn gói jailbreak để cài `ldid`).

Ngoài chiếc iPad mini 2 đã kiểm thử, tài liệu này không nêu tên model iPhone/iPad cụ thể nào là "được hỗ trợ", vì chưa có model nào khác hoàn thành bài kiểm thử của dự án. Nếu bạn thử trên máy khác, hãy [báo lại kết quả](#đóng-góp) — đó là cách mở rộng danh sách. Dự án rất hoan nghênh các bạn thử trên máy cộng đồng.

#### Không hỗ trợ

- **iOS gốc / chưa jailbreak.** Không có jailbreak thì không thể cài binary chạy trực tiếp, LaunchDaemon hay dịch vụ hệ thống.
- **Thiết bị không chạy được binary arm64 iOS** mà dự án phát hành.
- **Môi trường không cấp được quyền root** (quá trình cài đặt phải ghi đường dẫn hệ thống và quản lý dịch vụ hệ thống).
- **Cấu hình thiếu các thành phần jailbreak mà bộ cài đặt phụ thuộc**, ví dụ không hỗ trợ LaunchDaemon hoặc không chạy được binary ký ngoài / ký tạm.
- **Các kiểu bố cục rootless không có các đường dẫn truyền thống** bên dưới, trừ khi sau này có tài liệu kiểm thử ghi rõ. Bộ cài đặt hiện tại vẫn yêu cầu bố cục filesystem truyền thống (xem [Rootful / rootless](#rootful--rootless)).

Các chip mới hơn (kể cả thiết bị arm64e) và các thế hệ iOS mới hơn **không được coi là đã hỗ trợ** chỉ vì deployment target là iOS 12.0. Trong trường hợp tốt nhất, chúng vẫn thuộc nhóm "kỳ vọng tương thích, chưa kiểm chứng" cho tới khi có kết quả trên máy thật.

### Khả năng tương thích jailbreak

#### Môi trường jailbreak đã kiểm thử

- **Amethyst + Procursus**
- **Semi-untethered** (sau khi khởi động lại hoàn toàn, jailbreak sẽ tạm ngưng và cần kích hoạt lại)
- **iOS 12.5.7**
- Đã thử trên **iPad mini 2** nêu trên, bao gồm cả bài khởi động lại hoàn toàn rồi kích hoạt jailbreak thủ công, sau đó Agent và Hub tự quay lại.

#### Các jailbreak khác

Các jailbreak / bootstrap khác, nếu cung cấp đủ môi trường root / bootstrap yêu cầu, *có thể* chạy được, nhưng hiện tại **chưa kiểm chứng** cho tới khi có kết quả thử trên máy thật.

Cụ thể, một jailbreak / bootstrap khác chỉ có cơ hội chạy hợp lý khi nó đáp ứng đủ các điều kiện sau — đây chính là những gì `install.sh` thực sự dùng tới:

- Chạy được lệnh arm64 trực tiếp (chạy được binary Agent / Hub đã tải về).
- Có quyền root hoặc `sudo` trong lúc cài đặt.
- Ghi được vào các vị trí cài đặt đã kiểm thử (`/usr/local/bin`, `/var/lib`, `/Library/LaunchDaemons`, `/var/log`).
- Quản lý `launchd` / LaunchDaemon qua `launchctl` (load, unload, list).
- Có `ldid` (hoặc có nguồn `apt` để cài `ldid`) để ký binary ngay trên máy.
- Có `curl` và mạng tới GitHub releases, cùng công cụ SHA-256 để kiểm tra checksum.
- Có filesystem / bootstrap jailbreak ghi được, với hành vi Unix chuẩn.

Tài liệu này không liệt kê thêm tên jailbreak (hay bootstrap) nào là "được hỗ trợ", vì chưa có bằng chứng trong repo hay kết quả máy thật nào chứng minh. "Về lý thuyết có thể chạy" không giống với "đã được dự án kiểm thử".

### Rootful / rootless

"Rootful" ở đây chỉ là cách gọi ngắn gọn cho kiểu jailbreak với bố cục filesystem cổ điển, nơi các đường dẫn hệ thống truyền thống như `/usr/local/bin`, `/Library/LaunchDaemons` và `/var/lib` ghi được trực tiếp. Jailbreak "rootless" thì ánh xạ lại hoặc hạn chế các đường dẫn đó.

Bộ cài đặt đã kiểm thử hiện dùng **các đường dẫn hệ thống truyền thống**:

- Binary: `/usr/local/bin/beszel-agent`, `/usr/local/bin/beszel-hub`
- Định nghĩa dịch vụ: `/Library/LaunchDaemons/dev.beszel.agent.plist`, `/Library/LaunchDaemons/dev.beszel.hub.plist`
- Dữ liệu: `/var/lib/beszel-agent`, `/var/lib/beszel-hub`
- Trạng thái bộ cài đặt: `/var/lib/beszel-ios/install-state`

Nếu các thư mục đó chưa có, bộ cài đặt sẽ thử tạo; nếu không ghi được, nó sẽ dừng và báo rõ là chưa hỗ trợ kiểu bố cục rootless. Nói ngắn gọn: cấu hình Procursus đã kiểm thử thì chạy được, còn **môi trường jailbreak rootless chưa được kiểm thử tương đương** và hiện nên coi là chưa hỗ trợ. Đợt viết tài liệu này không thiết kế lại đường dẫn bộ cài đặt.

## Agent, Hub hay cả hai?

Chưa biết nên chọn mục nào trong bộ cài đặt? Hiểu đơn giản như sau:

- **Agent** — "theo dõi chiếc máy này." Nó lặng lẽ đo đạc chiếc iPhone/iPad rồi gửi số liệu về một Hub Beszel đang chạy ở nơi khác.
- **Hub** — "bảng điều khiển." Nó thu thập số liệu từ các Agent, lưu lịch sử và hiển thị giao diện web để bạn đăng nhập.
- **Cả hai** — "tự đủ." Thiết bị iOS vừa tự theo dõi chính nó, vừa host luôn bảng điều khiển, bạn chỉ cần mở trình duyệt trỏ vào chiếc iPad là xem được.

Phần lớn các bạn thêm iPad cũ vào hệ thống sẵn có thì chỉ cần **Agent**. Chọn **Cả hai** nếu muốn chiếc iPad chạy độc lập.

Cổng mặc định: Agent `45876`, Hub `8090`. Bạn có thể kiểm tra Hub ngay trên máy tại `http://127.0.0.1:8090/api/health`.

## Giám sát pin

Vì đây là iOS, Agent còn báo cáo thêm thông tin pin — phần trăm và trạng thái sạc — ngay cạnh CPU và bộ nhớ trên bảng điều khiển. Không cần thiết lập gì thêm.

## Sau khi khởi động lại

> **Lưu ý:** jailbreak đã kiểm thử thuộc loại **semi-untethered**. Khi bạn khởi động lại hoàn toàn, môi trường jailbreak sẽ tạm ngưng — đó là hành vi bình thường của jailbreak, không phải lỗi của Beszel.

Cụ thể:

- File và dữ liệu của Beszel vẫn nằm nguyên trên máy sau khi khởi động lại. Khởi động lại **không** làm mất Beszel.
- Sau khi khởi động lại, bạn kích hoạt jailbreak như bình thường (với cấu hình đã kiểm thử thì bạn kích hoạt lại Amethyst thủ công).
- Bạn **không** cần cài lại Beszel.
- Khi jailbreak đã hoạt động trở lại, các dịch vụ Agent và Hub của Beszel sẽ tự chạy lại.

Chuỗi khởi động lại rồi kích hoạt lại này đã nằm trong bài kiểm thử máy thật: sau khi khởi động lại và kích hoạt jailbreak thủ công, cả hai LaunchDaemon đều quay lại, Hub trả lời khỏe mạnh, Agent kết nối lại với số liệu mới, dữ liệu và lịch sử đều còn nguyên.

## Cập nhật

Chạy lại đúng lệnh cài đặt:

```sh
curl -fsSL https://raw.githubusercontent.com/nghianguyen150612/Beszel-iOS/iOS/install.sh | sudo sh
```

Chọn **Update**. Cài đặt và dữ liệu Hub của bạn được giữ nguyên, binary mới được kiểm tra trước khi thay thế, bản cũ được giữ làm sao lưu và sẽ tự quay lại nếu bản mới khởi động thất bại.

## Sửa lỗi và cấu hình lại

Cùng menu bộ cài đặt còn có:

- **Diagnostics (Chẩn đoán)** — kiểm tra trạng thái chỉ đọc cho Agent và Hub (file đã cài, trạng thái dịch vụ, cổng, tình trạng hoạt động). Chẩn đoán không bao giờ in ra giá trị key của Agent.
- **Repair (Sửa lỗi)** — sửa nhẹ nhàng như khởi động lại dịch vụ tại chỗ, khôi phục binary bị thiếu, hoặc tạo lại cấu hình đã hỏng khi được xác nhận, kèm khả năng tự quay về cấu hình cũ nếu sửa thất bại.
- **Reconfigure (Cấu hình lại)** — đổi cổng/key của Agent hoặc cổng của Hub thông qua một giao dịch có kiểm tra, có sao lưu cấu hình hiện tại trước.

## Gỡ cài đặt

Chạy lại lệnh cài đặt rồi chọn **Uninstall**, sau đó chọn Agent, Hub hoặc cả hai.

Gỡ thông thường sẽ xóa ứng dụng và dịch vụ nhưng **giữ lại dữ liệu**, nên sau này cài lại bạn không mất lịch sử. Xóa dữ liệu là một bước riêng, rõ ràng: bộ cài đặt chỉ xóa dữ liệu khi bạn gõ đúng cụm xác nhận mà nó hiển thị trên màn hình. Bất kỳ câu trả lời nào khác đều giữ dữ liệu an toàn.

Gỡ cài đặt chạy được ngoại tuyến, không cần tải hay ký gì thêm.

## Vị trí dữ liệu

Tiện cho việc sao lưu:

- Dữ liệu Agent: `/var/lib/beszel-agent`
- Dữ liệu Hub (cơ sở dữ liệu, tài khoản, lịch sử): `/var/lib/beszel-hub`
- Trạng thái bản phát hành của bộ cài đặt: `/var/lib/beszel-ios/install-state`

Đừng tự xóa các thư mục này bằng tay — hãy dùng menu của bộ cài đặt. Đặc biệt thư mục Hub chứa tài khoản và lịch sử của bạn.

## Câu hỏi thường gặp

**Có cần jailbreak không?**
Có. Beszel-iOS cài binary chạy trực tiếp và dịch vụ hệ thống, iOS gốc không cho phép việc đó.

**Máy iOS gốc (chưa jailbreak) có cài được không?**
Không. iOS gốc không được hỗ trợ.

**Chỉ cài mỗi Agent được không?**
Được. Đó là cách dùng phổ biến nhất: chiếc iPad báo cáo về một Hub đang chạy ở nơi khác.

**iPad có chạy luôn Hub được không?**
Được. Hub chạy trực tiếp trên chiếc iPad đã kiểm thử và phục vụ bảng điều khiển Beszel bình thường. Bạn cũng có thể chạy Agent + Hub cùng lúc trên một máy để nó tự giám sát chính nó.

**Khởi động lại máy có mất Beszel không?**
Không. File và dữ liệu vẫn ở nguyên trên máy. Vì jailbreak đã kiểm thử thuộc loại semi-untethered, bạn chỉ cần kích hoạt jailbreak lại sau khi khởi động; Agent và Hub sẽ tự chạy lại.

**Thử trên iPhone/iPad khác được không?**
Thử được nếu máy đó đáp ứng các điều kiện trong mục [Kỳ vọng tương thích](#kỳ-vọng-tương-thích-chưa-kiểm-thử-thực-tế), nhưng hãy coi đó là chưa kiểm chứng và nhớ báo lại kết quả. Chỉ cấu hình iPad mini 2 nêu trên là đã kiểm thử.

**Đây có phải phần mềm Beszel chính thức không?**
Không. Đây là bản port cộng đồng không chính thức. Dự án Beszel gốc không phát hành hay hỗ trợ bản iOS.

## Hạn chế đã biết

- Kiểm thử máy thật hiện chỉ bao phủ cấu hình iPad mini 2 / A7 / iOS 12.5.7 / Amethyst + Procursus nêu trên. Các phần cứng và jailbreak khác chưa được kiểm chứng.
- Máy phải đã jailbreak và có quyền root. iOS gốc và kiểu bố cục rootless chưa được bộ cài đặt hiện tại hỗ trợ.
- Sau khi khởi động lại hoàn toàn, bạn phải kích hoạt lại jailbreak semi-untethered thì dịch vụ Beszel mới chạy lại được (sau đó chúng tự quay lại).
- Đây là bản port cộng đồng không chính thức, không phải bản iOS do dự án gốc hỗ trợ.

## Bản phát hành

Bản binary iOS hiện tại: **v0.19.0-ios.1**

Phiên bản có hai phần: `0.19.0` là phiên bản Beszel gốc mà bản port dựa trên, còn `ios.1` là số revision của bản port iOS cho base đó. Bản thân bộ cài đặt được đánh phiên bản riêng (hiện là `1.0.0`).

Mỗi bản phát hành gồm ba file: binary Agent, binary Hub và file checksum để bộ cài đặt kiểm tra trước khi cài. Bạn có thể xem chúng trong [GitHub Releases](https://github.com/nghianguyen150612/Beszel-iOS/releases).

## Tài liệu

Chi tiết hơn ngoài trang này:

- [Tình trạng bản port iOS](docs/ios-port-status.vi.md) — cái gì chạy được và đã thử những gì
- [Kiểm thử thiết bị thực tế](docs/device-validation.vi.md) — kết quả đầy đủ trên chiếc iPad tham chiếu
- [Kiến trúc bản port](docs/architecture.vi.md) — các thành phần ghép với nhau trên iOS ra sao
- [Ghi chú build iOS](docs/ios-build-notes.vi.md) — các bản phát hành được build thế nào
- [Đồng bộ với bản gốc](docs/upstream-sync.vi.md) — bản port bám sát Beszel gốc ra sao

### Bản gốc tiếng Anh

Bản tiếng Anh gốc: [README.md](README.md)

## Build từ mã nguồn

Đa số người dùng không bao giờ cần bước này — bộ cài đặt đã cho sẵn binary.

Nếu bạn muốn tự build binary iOS, bạn cần macOS có Xcode iPhoneOS SDK, cùng Go và Bun, làm theo đúng workflow build của repo. Xem [Ghi chú build iOS](docs/ios-build-notes.vi.md) để biết đầy đủ các bước.

## Đóng góp

Đóng góp hữu ích nhất lúc này là thử trên thêm phần cứng. Nếu bạn thử Beszel-iOS trên một chiếc iPhone/iPad đã jailbreak khác, hãy mở issue hoặc discussion kèm:

- model máy (ví dụ iPad mini 2)
- giá trị `hw.machine` (ví dụ `iPad4,4`)
- phiên bản iOS
- jailbreak và bootstrap (ví dụ Amethyst + Procursus)
- bạn đã chạy Agent, Hub hay cả hai
- cái gì chạy được, và log liên quan nếu có lỗi

Các bản sửa build, bản sửa giữ tương thích với bản gốc, và cải thiện tài liệu cũng rất được hoan nghênh. Hãy nhắm vào nhánh `iOS`.

> Đừng đưa private key, mật khẩu, token hay bản sao cơ sở dữ liệu Hub vào báo cáo.

## Dự án gốc và ghi công

Beszel là của [henrygd](https://github.com/henrygd/beszel). Repo này là bản port iOS cộng đồng không chính thức, duy trì trên nhánh `iOS` — các câu hỏi chung về Beszel xin gửi về dự án gốc.

## Giấy phép

Giấy phép MIT — xem [LICENSE](LICENSE). Giữ nguyên bản quyền gốc:

> Copyright (c) 2024 henrygd
