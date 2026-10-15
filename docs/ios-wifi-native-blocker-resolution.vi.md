# P012A — Giải quyết các điểm chặn Wi-Fi native

P012 vẫn **BLOCKED**, Gate C **PENDING**, PR #14 giữ Draft. Bắt đầu từ
`88dcf9b89389cd58de6f4859d4619485451dfa82`, nhánh tích hợp
`33840093bb465f0b164bb7d66aebc8cd85f86a4e`. Giữ nguyên chứng cứ P012 cũ.
Agent iOS vẫn trả nil; không có cờ bật API thử nghiệm. Bộ thu Linux/macOS/Windows
và hợp đồng JSON/CBOR không thay đổi.

## WIFI-ABI-12 — BLOCKED; đã xác minh export đúng firmware

[Manifest](ios-wifi-native-abi-manifest.json) ghi URL, revision, hash của Cykey,
Theos 12.4, Procursus và quy tắc sở hữu CF của Apple. Header lịch sử không được
coi là chứng minh ABI của iOS 12.5.7.

[CI nghiên cứu](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38741425098)
tại `6b859dd6bdf4b7be457546567ba77ef5dfc4e070` đã kiểm tra SHA256 ảnh restore
Apple, BuildManifest iPad4,4/j85ap và SystemVersion 12.5.7/16H81. MobileWiFi chỉ
nằm trong dyld cache. Trích xuất bằng công cụ cache trên máy macOS cho UUID
`FA49B5A2-2578-38A0-AE5E-5DDBB37D68EA`, SHA256
`052eef70beb4e8471471c9562772ea79437dc2aba51e0cb302f054e0972009ba`.
Cả sáu export đã được xác minh trên ảnh đúng thiết bị. Lưu
[báo cáo](evidence/ios-wifi-12.5.7/firmware-report.json) và
[disassembly giới hạn](evidence/ios-wifi-12.5.7/selected-functions.txt), không
phân phối hay chạy binary firmware. Lần nghiên cứu thứ hai lỗi thiếu SDK sysroot;
lần thứ ba đã sửa và thành công. Cache kiểm tra checksum tránh tải lại dư thừa.

Thanh ghi ARM64 củng cố prototype C pointer/int lịch sử: Create lưu 32 bit flags,
CopyProperty tuần tự hóa đối số thứ hai, CopyCurrentNetwork gọi MIG rồi tạo
WiFiNetwork, GetInterfaceName trả con trỏ mượn không retain. Tuy nhiên, kiểu opaque,
constructor/callee đầy đủ, quyền sở hữu và kiểu chuỗi còn là suy luận mạnh, chưa
đủ xác minh toàn bộ ABI. Chú thích CF-string trong bản trích xuất chưa giải mã
đúng nên không dùng làm tên khóa. CopyDevices, CopyCurrentNetwork và
GetInterfaceName dereference đối số đầu mà không kiểm tra NULL; CopyProperty và
GetSSID có kiểm tra NULL. Không được giả định mọi hàm chấp nhận NULL.

Mã sản xuất mới chỉ là parser/client chưa kích hoạt và chuyển đổi public CF;
không ép kiểu hay gọi con trỏ hàm private. Test kiểm tra hash/prototype và đường
chuyển đổi/giải phóng. Cần phân tích thêm constructor/callee trước khi viết bridge
thu nhận, và cần xác minh thiết bị trước khi bật. Có symbol không đồng nghĩa ABI.

## WIFI-ASSOCIATION-12 — BLOCKED

CopyDevices đọc dictionary cache của manager và lọc role 0; không chứng minh liên
kết station hiện tại hoặc độ mới. CurrentNetwork gọi
`wifi_device_copy_current_network`, nhưng chưa biết cache của daemon hết hiệu lực
thế nào khi mất kết nối. Đã thấy chuỗi RSSI/RSSI_CTL_AGR, chưa chứng minh đơn vị
dBm, kiểu số, sentinel hoặc độ mới. Không đổi tín hiệu chuẩn hóa thành dBm.

Parser thực tế yêu cầu powered/station/current_network, trạng thái associated rõ
ràng và tên interface có thật trên OS. Chỉ chấp nhận số nguyên RSSI_CTL_AGR trong
-150..-1; lịch sử int8 giữ chính sách bão hòa -128 cho giá trị yếu hợp lệ. SSID
UTF-8 tối đa 32 byte là tùy chọn; SSID ẩn/lỗi/không được cấp quyền không làm mất
RSSI hợp lệ. Không chấp nhận BSSID/password. Lỗi hoặc mất kết nối trả nil/map mới,
không giữ mẫu cũ. Test chỉ xác minh chuyển đổi dữ liệu giả lập, chưa chứng minh
lời khai liên kết hay đơn vị của nguồn native.

Có fixture không/nhiều thiết bị, thiếu metadata, tắt/ngắt Wi-Fi, dictionary/số sai,
sentinel/tràn/zero/dương/scaled, nối lại mạng cũ/mới, SSID lỗi/ẩn, lỗi sau thành
công và phục hồi qua mã chuyển đổi thật cùng JSON/CBOR. Hành vi ngắt/nối lại và
đơn vị thật vẫn bắt buộc kiểm tra trên thiết bị. Chưa bật thu nhận.

## WIFI-AUTH-12 — BLOCKED

Firmware đúng có chuỗi `com.apple.wifi.manager-access` trong MobileWiFi và wifid;
ví dụ Procursus đã ghim yêu cầu khóa này cùng nhiều quyền khác. Đây chỉ là ứng
viên, chưa chứng minh quyền đó cần hay đủ cho các lệnh chỉ đọc của helper/
LaunchDaemon dưới Amethyst. Biên dịch, root UID, dlopen hay export không chứng
minh daemon cấp quyền.

Helper sở hữu IPC sẽ cần xác minh chữ ký/quyền riêng; Agent không kế thừa quyền
của tiến trình con. Chưa biết có cần quyền bổ sung hay không. Không sao chép bộ
quyền netctl. P016 sở hữu đóng gói, ldid, đường dẫn tuyệt đối tin cậy và quyền sở
hữu executable. P012A không thêm entitlement, đổi ký production, installer hay
LaunchDaemon. Chưa chạy lời gọi daemon, chưa đo hành vi bị từ chối trên iPad.

## WIFI-BOUND-12 — PASS cho kiến trúc/fixture host; iOS còn chờ

Disassembly xác nhận MIG đồng bộ nhưng không chứng minh giới hạn thời gian hoặc
hủy. Không chọn gọi CGO trực tiếp. Client chưa kích hoạt dùng executable riêng,
đường dẫn tuyệt đối regular, argv cố định, không shell, môi trường tối thiểu,
bỏ stderr, stdout tối đa 16 KiB, một lần thu mỗi client, deadline hai giây,
kill process group Unix, WaitDelay pipe 100 ms và Wait/reap tiến trình con trực
tiếp. Close chặn lần đọc mới, hủy và chờ lần hiện tại theo deadline người gọi.
Helper thiếu/bị cấm/crash/timeout không trả snapshot; không bỏ lại goroutine mỗi
lần timeout, không thêm vòng polling riêng.

Executable lỗi thật kiểm tra output lớn/rỗng/hỏng, crash/exit lỗi, thiếu/không có
quyền/không executable/symlink, giữ pipe, timeout lặp, gọi song song, cancel,
Close và phục hồi. Test race kiểm tra tiến trình con đã biến mất và số goroutine
có giới hạn. Subreaper Linux chỉ dùng trong test cho fixture cố ý tạo orphan;
production không nhận nuôi tiến trình không liên quan. Diagnostic native không fork.

Đây là cách ly tiến trình, không phải hủy lời gọi C. Giả định helper tin cậy cùng
UID và POSIX kill/wait bình thường; không phải sandbox binary độc hại hay chứng
minh về kernel wait không ngắt được. Cần P018 kiểm tra exec/group kill/reap, độ trễ
và tài nguyên A7. PASS nguồn không tự cho phép bật thu nhận.

## Diagnostic native riêng và kiểm tra quyền sở hữu

`tools/ios-wifi-probe` là executable C riêng, bị vô hiệu hóa thu nhận. Snapshot
luôn unavailable. Identity/inventory không ép kiểu hay gọi hàm private. dlopen/
dlclose framework cố định là tùy chọn; framework/symbol thiếu chỉ thành boolean.
Không đọc SSID/BSSID/password. Chỉ có artifact riêng để review, không vào payload
Agent/Hub, installer hay ký production. Validator chặn ioshelper lọt vào dependency
Agent/Hub bình thường.

Fixture public CF chạy chuyển đổi và cleanup container có sở hữu dưới sanitizer
address/undefined, allocator đếm cấp phát qua các vòng thành công/lỗi. Không tuyên
bố LeakSanitizer trên macOS hay sở hữu opaque private đã được chứng minh. Fixture
symbol thiếu/một/đủ cố ý có prototype không tương thích và abort nếu bị gọi;
PASS inventory chỉ chứng minh không gọi hàm, không chứng minh ABI. Cross-build
native chỉ là chứng cứ build.

## Quy trình P018 chỉ đọc — CHƯA THỰC HIỆN

1. Cần người dùng cho phép rõ ràng và có quyền truy cập bản diagnostic đã review;
   P016 phê duyệt vị trí tin cậy, checksum và chính sách ký trước. Không tự deploy/
   ký. Chỉ lưu JSON đã khử thông tin nhạy cảm và metadata.
2. Kiểm tra SHA256 riêng và revision `--build-info-v1`. Chạy `--identity-v1` dưới
   supervisor có deadline, xác nhận iPad4,4/16H81; người dùng cung cấp phiên bản
   Amethyst/Procursus. Không cài/gọi refresh gói hoặc cấu hình lại jailbreak.
3. Chạy `--symbols-v1` với deadline hai giây, output giới hạn và reap trực tiếp.
   Ghi framework/sáu symbol, exit và độ trễ. Đây chưa phải kiểm tra ABI/quyền.
4. Chạy `--snapshot-v1`: revision này bắt buộc unavailable. Có thể kiểm tra client
   lặp/song song/timeout mà không gọi private; ghi PID/reap/độ trễ thật.
5. **DỪNG trước thu nhận.** Tool chưa có chế độ association/RSSI. Phải review đủ
   ABI/opaque ownership, station có thẩm quyền và dBm raw trước; sau đó viết/review
   revision chỉ đọc có giới hạn và xin phép rõ ràng cho revision đó.
6. Chỉ sau review, nếu được phép, so sánh chữ ký bị từ chối với ứng viên quyền tối
   thiểu manager-access được phê duyệt riêng. Ghi bộ quyền chính xác, thành công/
   từ chối và độ trễ từng hàm. Không suy từ root, thêm quyền netctl rộng, sửa
   wifid/kernel/trust cache hay tắt bảo vệ.
7. Với revision thu nhận tương lai, chỉ người dùng đổi Wi-Fi bằng điều khiển bình
   thường: kết nối → ngắt/tắt → nối lại mạng cũ/mới. Yêu cầu không có số đo sau
   ngắt/từ chối, phục hồi mới, interface thật, xác minh dBm, không SIGILL, dịch vụ
   ổn định và tài nguyên giới hạn. Chỉ lưu nhãn chuyển trạng thái, RSSI/interface,
   status/latency, firmware/build; xóa mọi SSID/BSSID/password và mạng khác.

Bước 5–7 là yêu cầu cho mã tương lai đã review, chưa phải lệnh có thể chạy ở
revision này hoặc test đã hoàn tất. Nếu không chứng minh an toàn, giữ unavailable.

## Cổng nghiệm thu

SHA, CI, digest và kết quả cloud cuối cùng nằm trong `p012a` của
[ios-wifi-rssi-validation.json](ios-wifi-rssi-validation.json);
[ledger](ios-wifi-rssi-checkpoints.md) giữ toàn bộ chứng cứ cũ. Fixture/native build
không chứng minh quyền daemon hoặc số đo RSSI thật. P012 BLOCKED, Gate C PENDING,
P018/Gate F còn chờ. PR #14 Draft, không merge hoặc bắt đầu P013. Không đụng thiết
bị, DB production, release, nhánh iOS/main. Việc tiếp theo: hoàn tất review ABI/
association/đơn vị daemon rồi thực hiện P018 từng bước khi được cho phép.

Chính sách canonical trong upstream-v021-acceptance.md cho phép hoãn khả năng không khả thi với owner và fallback rõ ràng. P012A yêu cầu chứng minh thu RSSI native; chưa có phê duyệt thay đổi phạm vi nên không dùng chính sách này để đổi cổng thành PASS.

## Checkpoint cloud đã chạy và candidate cuối

Nguồn `aac215abdb89c5fa6bc307a550216d044f0906a9`:
[iOS Build PASS](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38742936483),
[installer PASS](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38742936311).
Fixture public CF thật trên macOS PASS sanitizer address/undefined và 1.000 vòng
chuyển đổi/cleanup. LeakSanitizer không được hỗ trợ/đã tắt, không ghi PASS. Snapshot
unavailable từ executable C thật đi qua client Go giới hạn PASS ba lần race.
Executable lỗi, inventory và regression P007–P011 hosted PASS. Không gọi constructor
opaque hay daemon private.

Đã tải artifact và kiểm lại checksum/provenance/Mach-O độc lập:
Agent `1aa5218684cb96eeb058c4b0a2c32bc306d6c30b30d3582ed5f0ae47dbd1e981`,
Hub `f6a3605e18e9c022c893fdeed23d8a97b50c0ce8f1d92381e09706949972e684`,
diagnostic `835700069fbba7736854b0fc6a5ff3835fbe0c4c11af0eee129cb064f96d79ff`.
Cả ba ARM64/platform 2/tối thiểu 12.0. Agent/Hub dùng Go 1.27.1 ghim, VCS sạch,
runtime A7 đã vá, 95 asset frontend thật; C diagnostic có revision/hash nguồn
riêng. Không hard import private, hai cache build byte-identical, SHA256SUMS đúng,
release skipped. Hash chỉ áp dụng checkpoint này, không gán cho bản build sau.

Regression nguồn ổn định local PASS: 54 audit Python; matrix/readiness, sync 10,
sentinel 45, bootstrap 54, installer 814; canonical Go; race đầy đủ các package
liên quan (Hub systems chậm dùng timeout Go chuẩn); ba vòng race tập trung
P008–P012/Procursus; mười vòng nil-adapter; chọn Wi-Fi năm nền tảng; build Linux
Agent/Hub; compile Wi-Fi/client Windows/FreeBSD; Bun 1.4.0 frozen install/catalog,
44 test/build production, 95 asset, lockfile nguyên. Không chạy binary cross-build.

Lần tiếp tục này bổ sung file framework tồn tại nhưng không hợp lệ và xác minh
unload tường minh trước khi tiến trình kết thúc. Loader sản xuất PASS 100 vòng mỗi
ca missing/invalid/partial/all trên Linux dưới ASan/UBSan; mutant riêng cố ý bỏ
dlclose bị fixture từ chối. CF thêm số phân số/NaN/vô hạn và cleanup property có
sở hữu khi thiếu đích output. Có test NULL từ bước trước; **chưa** cưỡng bức lỗi
cấp phát manager/device private vì chưa có bridge an toàn. Không ghi là đã chạy
allocator fault injection. Kết quả hosted bổ sung, HEAD cuối và digest artifact
cuối ghi tại checkpoint mới nhất của PR #14.

Dừng nghiên cứu tĩnh ở chứng cứ đã commit, không tải lại firmware. Disassembly
hiện có không giải quyết cache daemon, đơn vị raw, quyền từng hàm hoặc kill/wait
trên thiết bị. Đây vẫn là nghĩa vụ P018 sau review ABI theo quy trình; không suy
ra từ fixture. P012 BLOCKED / Gate C PENDING / PR Draft giữ nguyên.
