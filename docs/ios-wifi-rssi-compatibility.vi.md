# P012 — Tương thích RSSI Wi-Fi trên iOS

## Quyết định hoãn phạm vi được chủ dự án phê duyệt — 2026-10-16

Chủ dự án đã phê duyệt hoãn thu thập RSSI Wi-Fi native trên iOS vì iPad tham
chiếu tạm thời không có sẵn. Quyết định này chấp nhận phạm vi fallback/source
không khả dụng để tích hợp và cho phép tiếp tục P013 sau khi hoàn tất PR #14.
Quyết định chỉ đổi phạm vi và trình tự; không đổi kết quả kiểm thử và không
chứng minh RSSI native hoạt động.

```text
P012 native Wi-Fi RSSI: DEFERRED — REFERENCE DEVICE UNAVAILABLE
P012 fallback/source/CI: ACCEPTED FOR INTEGRATION
Native RSSI measured: NO
Normal iOS Wi-Fi RSSI: UNAVAILABLE / NIL
P018 physical validation: PENDING
P013: AUTHORIZED
```

P018 phụ trách kiểm chứng Wi-Fi trên thiết bị thật. P016 phụ trách mọi thay đổi
đóng gói helper, entitlement hoặc ký mã có bằng chứng trong tương lai.
`WIFI-ABI-12`, `WIFI-ASSOCIATION-12`, `WIFI-AUTH-12` và phần kiểm chứng iOS vật
lý của `WIFI-BOUND-12` vẫn được hoãn; không mục nào được ghi là PASS. Phần
fallback/source P012 của Gate C được chấp nhận để tiếp tục, còn bằng chứng
thiết bị/vật lý vẫn chờ P018. Không có ủy quyền phát hành.

Bằng chứng cloud được chấp nhận gắn với P012 source HEAD cuối cùng
`2a16d0f46f6fd1c91f40e295190b89fb8713b8cd`: iOS Build và installer trên hosted
CI đều pass, release bị bỏ qua; các kiểm tra Go, race, audit, frontend, build
Linux, sanitizer, artifact, checksum, provenance và fallback nil đã được ghi
nhận là pass. Bằng chứng này chỉ xác nhận fallback nil an toàn và hành vi
source/build; không đo RSSI và không kiểm thử iPad thật. Xem [bằng chứng HEAD
cuối](https://github.com/nghianguyen150612/Beszel-iOS/pull/14#issuecomment-6204677937).

## Trạng thái kỹ thuật lịch sử trước quyết định hoãn

Phần dưới đây ghi lại trạng thái nghiệm thu kỹ thuật và kết quả kiểm thử trước
khi chủ dự án phê duyệt hoãn. Giữ nguyên chúng làm bằng chứng lịch sử; quyết
định phạm vi không biến blocker native thành PASS.

P012 **BLOCKED đối với thu thập RSSI native**. Adapter iOS đã triển khai trả về
unavailable an toàn. Đây là checkpoint source/fixture đã công bố, không phải
khẳng định MobileWiFi hoạt động trên thiết bị tham chiếu. PR giữ Draft, không
merge khi hợp đồng native còn chưa xác minh. Gate C **PENDING**, không hạ tiêu
chí để đạt PASS. Chưa bắt đầu P013.

## Nguồn gốc và tính khả thi

Integration ban đầu: `33840093bb465f0b164bb7d66aebc8cd85f86a4e`.
Production iOS: `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`.
Frozen main: `bb23f99b3eeaced9608e11e280e7c986dba650be`.
Nhánh: `ios/wifi-rssi-compatibility`.
[Draft PR #14](https://github.com/nghianguyen150612/Beszel-iOS/pull/14).
Đã fetch và xác minh chính xác baseline; dùng lại checkout sạch hiện có, không
xóa tiến độ. Đủ 178 commit upstream vẫn là tổ tiên, đủ 13 patch logic được kiểm kê.

[Hồ sơ khả thi](ios-wifi-provider-feasibility.md) ghim header Cykey lịch sử,
export arm64 SDK iOS 12.4, source Procursus netctl và tài liệu API/quyền sở hữu CF
của Apple. Hồ sơ có chữ ký hàm, đối tượng mượn/sở hữu và các khoảng trống bằng
chứng. MobileWiFi là ứng viên chính nhưng chưa được chọn để thực thi.
Apple80211 chưa có hợp đồng iOS 12 được xác minh; CaptiveNetwork chỉ có metadata;
NEHotspotNetwork trả tín hiệu chuẩn hóa 0–1, không phải dBm, còn fetchCurrent yêu
cầu iOS 14+. Không thay bằng CoreWLAN/JXA, nl80211, chất lượng, vạch sóng, tốc độ
hay lưu lượng.

Stub SDK 12.4 export sáu hàm manager/device/network trong hồ sơ khả thi. Danh
sách export không chứng minh ABI tham số, kiểm tra đối tượng hay chính sách daemon
trên iOS 12.5.7. Ví dụ Procursus release dictionary trước khi đọc số mượn và in
metric đã scale với nhãn dBm; không sao chép các mẫu này. Chưa xác lập đủ tin cậy
hợp đồng private API đúng firmware và giới hạn thời gian để phát hành lời gọi.

## Triển khai và hợp đồng dữ liệu

`agent/wifi/wifi_ios.go` dùng build constraint `ios`, trả nil. Fallback chung loại
iOS; macOS tiếp tục CoreWLAN qua osascript, Linux giữ nl80211, Windows giữ WLAN
native. Không sửa file collector của ba nền tảng. Validator yêu cầu đúng file
iOS, từ chối collector Wi-Fi sai nền tảng hoặc trùng lặp.

Không load framework, resolve/gọi symbol, thêm cầu CGO, dependency private lúc
link hay entitlement. Các trạng thái thiếu framework/symbol, manager/danh sách,
quyền, association, RSSI hoặc tên interface đều trả unavailable mà không gọi API
chưa xác minh. Đây không phải kiểm thử lỗi thực tế của API đó. Không có đối tượng
CF/con trỏ giữ lại cần release, worker, cache native, retry hay khởi tạo native.
Adapter trả đồng bộ cả với context đang hoạt động lẫn bị hủy. Giữ deadline chung
hai giây; context Go không thể hủy C IPC đồng bộ. Provider tương lai cần lời gọi
có thời gian chặn đã xác minh hoặc thiết kế process isolation/reaping đã review.

Giữ nguyên `Info.WiFi`, `Stats.WiFi`, map theo interface, field JSON và khóa số
CBOR. Không tự đặt tên interface iOS. SSID tùy chọn; sanitizer UTF-8 hiện tại bỏ
byte không hợp lệ mà không loại RSSI hợp lệ. SSID thiếu/ẩn/bị hạn chế riêng tư
không được thay bằng BSSID. Không xuất MAC, mật khẩu, danh sách mạng đã lưu hoặc
yêu cầu quét mạng xung quanh.

Lỗi chung đã xác nhận: đổi tín hiệu không hợp lệ sang int8 không kiểm tra có thể
biến số dương/quá lớn/không hữu hạn thành lịch sử có vẻ hợp lý. `Signals` nay loại
NaN và số ngoài khoảng dBm -150..-1 hiện có trước khi chuyển. Loại cả giá trị
chuẩn hóa 0..1 và phân số âm gần zero; kiểm tra khoảng không tự chứng minh đơn vị.
Provider phải chứng minh nguồn là dBm thô. Giữ cách làm tròn/bão hòa lịch sử cũ:
-150..-129 thành -128, không wrap; Info hợp lệ giữ độ chính xác nguồn. Fallback
này không xuất bất kỳ số RSSI iOS nào.

Tách khối cập nhật Wi-Fi hiện có của Agent thành hàm nhỏ nhận collector để kiểm
thử quyết định. Chỉ poll interval mặc định mới lấy dữ liệu; mọi kết quả, cả nil
và rỗng, thay snapshot hiện tại. Realtime dùng lại snapshot, không phải phép đo
mới. Poll mặc định lỗi xóa RSSI cũ. Giữ SKIP_WIFI và dạng biến môi trường có
prefix; không thêm polling nền độc lập.

## Entitlement và bảo mật

Source đóng gói Procursus ghim có `com.apple.wifi.manager-access` cùng nhiều quyền
rộng hơn. Đây chỉ là bằng chứng quyền ứng viên, chưa chứng minh cần/đủ hay bộ quyền
tối thiểu cho truy vấn chỉ đọc trên iOS 12.5.7. Root/jailbreak không phải bằng
chứng được cấp quyền. Không sửa entitlement, ký production, LaunchDaemon hay ldid.
P016 phải review quyền tối thiểu và phạm vi ký; P018 kiểm chứng CLI/LaunchDaemon
bị từ chối và được cấp quyền. Không sửa wifid/kernel/cấu hình, nguồn Wi-Fi,
join/disconnect hoặc cơ chế bảo mật.

## Kiểm chứng và giới hạn nghiệm thu

[Bằng chứng JSON](ios-wifi-rssi-validation.json) có SHA thực sự đã test, lệnh,
kết quả và URL hosted. [Ledger checkpoint](ios-wifi-rssi-checkpoints.md) ghi từng
bước bền vững và việc còn lại. Go test bằng danh sách file tường minh chạy đúng
adapter iOS production trên cloud, không chọn collector host; không phải test
phần cứng hay mô phỏng quyền MobileWiFi.

Coverage gồm unavailable lặp/đồng thời, SKIP_WIFI, tín hiệu hữu hạn/khoảng/overflow/
biên, UTF-8/CBOR của SSID, JSON/CBOR roundtrip, số lần acquisition default/realtime,
thay snapshot theo lifecycle, fixture wire khóa số cũ với tên interface Linux/
macOS/Windows, decoder Hub và disconnect/reconnect của record tổng hợp. DB Hub
kiểm thử là tạm và tổng hợp. **Chưa test** parser CF native, lỗi resolve symbol,
cân bằng CFRelease, quyền daemon thật hay interface station thực tế vì không
triển khai private integration.

Kết quả regression bắt buộc, Bun frozen install/catalog/test/build thật, build
Linux Agent/Hub, source selection, build iOS hosted, runtime A7, metadata Mach-O,
import, checksum, provenance và asset nhúng nằm trong JSON. Build thành công chỉ
xác nhận adapter unavailable; không chứng minh provider RSSI native. Không tính
CI P011 cũ hay hồ sơ thiết bị iOS 0.19 thành kiểm chứng P012 mới.

Nghiệm thu native P012: **BLOCKED** (`WIFI-ABI-12`, `WIFI-ASSOCIATION-12`,
`WIFI-AUTH-12`, `WIFI-BOUND-12`). Kiểm tra fallback/source/fixture có thể pass riêng,
không thay thế acquisition native còn thiếu. Acceptance chuẩn cho phép deferred
khả thi có owner và fallback absent/no-op; PR này không tuyên bố Gate C PASS mới
hay diễn giải lại acceptance case. Gate C PENDING đến khi giải quyết hoặc chấp
nhận rõ ràng deferral này. Chưa hoàn tất Gate D/F hoặc acceptance vật lý.

P018 / Gate F phải xác minh framework/symbol 12.5.7 thật, ABI/hình dạng đối tượng,
quyền tối thiểu, RSSI dBm thô đối chiếu nguồn native, danh tính interface hiện tại,
ngắt/tắt Wi-Fi/bị từ chối, reconnect/đổi SSID, giới hạn thời gian/resource cleanup,
không SIGILL và service ổn định trên iPad mini 2 A1489 / iPad4,4 / Apple A7 /
iOS 12.5.7 / Amethyst/Procursus. Không khẳng định số đo RSSI, pin, RAM hay quyền
thực tế trên thiết bị.

Bước cụ thể tiếp theo: xác minh hợp đồng firmware ghim và cách giới hạn lời gọi
trước khi thay adapter; quyền đóng gói tối thiểu thuộc P016, nghiệm thu thiết bị
thuộc P018. Resume từ nhánh và ledger đã push. P013 — tương thích migration
PocketBase, backup/restore database và authentication/security của Hub — là
milestone tiếp theo trong roadmap, chưa bắt đầu. Không sửa thiết bị/DB production,
không release/tag/deploy, không đổi production iOS hoặc frozen main.

## Bằng chứng build native của source đã hoàn tất

Source SHA: `d3f33c83c8d6164717a26749826408359fed84ea`.
[iOS Build PASS](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38724163923),
[installer PASS](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38724168872).
macOS 14.8.9 arm64, Xcode 15.4 / SDK iPhoneOS 17.5, Go 1.27.1, Bun 1.4.0.
Build Agent/Hub lặp bằng cache tách biệt cho kết quả giống từng byte; patch runtime
A7 cô lập và kiểm tra instruction pass. Cả hai là arm64 MH_EXECUTE, iOS platform 2,
minos 12.0, CGO=1, GOARM64=v8.0, VCS sạch đúng SHA source trên. Hub thật nhúng 95
asset. Đã tải artifact và kiểm chứng độc lập hash ZIP, metadata Mach-O, toàn bộ
SHA256SUMS, provenance source và asset frontend production cục bộ trong cloud.

| Payload | SHA256 |
| --- | --- |
| Agent | `993c82e71cf570ee490c731ee1ea2a6299b7e6573515b6b1790f707d962f7f6f` |
| Hub | `b3ea5f9327e5024ee84f2870088dc01a243ea011df522f1b4bf53b7054cb10b1` |

`otool -L` / `nm -u` xác nhận không có import MobileWiFi, CoreWLAN, Apple80211 hay
WiFi private. Dependency gồm CoreFoundation, Security, libresolv và libSystem.
Job release đã skip. Thay đổi tài liệu/filter nhánh workflow cuối nhận lượt CI
push mới trên feature HEAD; commit cuối và URL CI cuối ghi tại PR #14, không coi
run cũ là test cho SHA mới. Build này chỉ chứng minh adapter unavailable; RSSI
native P012 vẫn BLOCKED, Gate C vẫn PENDING.

## Tiếp tục P012A

Xem [giải quyết điểm chặn native](ios-wifi-native-blocker-resolution.vi.md) và phần
`p012a` của JSON. Đã xác minh export đúng firmware, thêm chuyển đổi chặt chẽ,
client helper giới hạn chưa kích hoạt và diagnostic native riêng luôn unavailable.
Không bật RSSI. Chứng cứ cũ giữ nguyên SHA ban đầu. P012 BLOCKED / Gate C PENDING.
