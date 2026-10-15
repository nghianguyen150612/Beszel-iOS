# P010 — Tương thích giám sát mạng và collector không hỗ trợ trên iOS

Kiểm tra source/fixture P010 và build native source-candidate đã PASS. Quyết
định integration cuối yêu cầu CI HEAD có đầy đủ tài liệu và merge thông thường
được ghi trong PR #12. JSON đã commit ghi chính xác candidate và kết quả
workflow. Một commit không thể chứa SHA của chính nó; kết quả CI HEAD cuối và
xác minh sau merge được ghi trong phần tiến độ của PR.

## Nguồn gốc và phạm vi

Clone mới chọn nhánh `iOS` trước. Integration ban đầu:
`0c48b1378393b325e9b61ba716f832b1bf807aa6`; iOS sản phẩm:
`0c3e4cd22ac24264834c401ad412e44aeda4e4f3`; main đóng băng:
`bb23f99b3eeaced9608e11e280e7c986dba650be`. Cả ba khớp trước khi sửa.
Main đóng băng là ancestor của integration. Các validator matrix/readiness
kiểm kê đủ 178 commit upstream và 13 patch iOS logic. Nhánh
`ios/network-monitor-compatibility` chỉ nhắm vào
`ios/upstream-v021-integration`, merge bằng merge commit thông thường.

Đã đọc hợp đồng acceptance, báo cáo build/A7/chọn source P007, telemetry/battery
P008, transport P009; source/test monitor, Agent/handler/collector, entity monitor,
sync/version guard/ownership/serialization/storage của Hub. Fixture dùng thư mục
thử nghiệm tạm và loopback; không truy cập `/var/lib/beszel-hub` thật.

## Kiến trúc và hợp đồng kết quả

Hub gửi ID record ổn định qua replace/upsert/delete. MonitorManager quản lý các
task có cấu hình bất biến. Task chia sẻ một probe đang chạy cho yêu cầu tức thời
và lịch định kỳ, chỉ ghi một sample vào lịch sử raw ngắn hạn và ring 61 bucket
phút. Lỗi dùng `-1` nội bộ và các trường loss/count/sum hiện có trên wire.
Khoảng chỉ có lỗi có aggregate latency bằng 0, loss 100%; các số 0 này không
phải phép đo thành công. Kết quả task bị hủy không vào lịch sử task thay thế.
Thu thập metadata chứng chỉ độc lập với xác minh sức khỏe HTTPS. Generation
resume thay đổi sẽ loại bỏ probe cũ. Giữ nguyên CBOR integer keys, JSON và các
guard `MinVersionNetworkMonitors=0.20.0`, `MinVersionCbor=0.12.0`,
`MinVersionAgentResponse=0.13.0`. Không sửa migration PocketBase hoặc DB thật.

## Lỗi được xác lập và thay đổi

| Bề mặt | Hành vi ban đầu | P010 |
| --- | --- | --- |
| HTTP | Helper trực tiếp không có deadline nếu client không giới hạn/nil | Context tối đa 10 giây, giữ deadline ngắn hơn của caller. |
| DNS | Chưa guard IPv6, port lỗi và shortcut cancellation/config của IP literal | Chuẩn hóa IPv6 có scope; từ chối ngoặc không khớp và port lỗi. Kiểm tra config/cancellation trước literal lookup; không thay server bằng resolver mặc định; empty answer trả lỗi. |
| ICMP | Cache mode vĩnh viễn, có thể giữ mode mất quyền hoặc không còn hỗ trợ | Cache IPv4/IPv6 độc lập, bỏ mode bị từ chối và thử nhận diện lại sau 5 phút; timeout/loss endpoint không thành lỗi capability. |
| Ping iOS | Thiếu case GOOS=ios | Dùng scheme Apple network-cmds của Procursus; tra executable và xác nhận `-n -c 1` bằng probe loopback có giới hạn. Không giả định ping6 đã cài. |
| Subprocess | `cmd.Output()` giữ output không giới hạn và có thể chờ pipe kế thừa | Stdout tối đa 64 KiB, bỏ stderr, WaitDelay 250ms, context kill/reap, không shell hoặc nâng quyền. |
| Publication | Test xác định tái hiện kết quả trả về sau delete khi đang chờ cert I/O | Publication cuối đồng bộ với cancellation; task hủy trả nil. |
| Lịch sử thay thế | Copy series cho mọi thay đổi cấu hình | Giữ lịch sử khi chỉ đổi interval; reset khi đổi target/protocol/port TCP/server DNS/chính sách TLS. Không copy cert HTTPS sang protocol khác. |
| Thứ tự thời gian | Sample đến muộn có thể thay timestamp mới nhất hoặc ghi đè bucket mới | Sắp xếp sample theo timestamp; retention dựa trên sample mới nhất; bucket cũ không ghi đè slot mới. |
| Snapshot CBOR | Test WS thật giữ loss 100% sau response thành công bỏ key loss bằng 0 | Decode vào vùng nhớ mới, chỉ publish khi thành công; response lỗi giữ nguyên destination cũ. Áp dụng generic hiện đại và raw WS legacy, giữ typed fallback. |
| Collector | Có thể dò tool GPU không liên quan, smartctl, ZFS utility và SMC/IOKit macOS trên iOS | Constructor unsupported sớm; không backend pool; guard API ZFS trực tiếp; hook nhiệt độ absent riêng; không dò Docker local tự động. |

Lỗi stale publication, DNS literal bị hủy và stale CBOR loss đã tái hiện trước khi sửa. Các phát hiện
khác được xác lập từ semantics source và fixture âm tính/recovery, không phải
lỗi đã quan sát trên iPad thật. Giữ chức năng đúng của upstream: chia deadline
TCP, token ICMP mật mã, chia sẻ probe, TLS theo monitor, cadence cert, collector
Linux/macOS và telemetry remote của Hub.

## TCP, DNS, HTTP và chiến lược quyền ICMP

TCP resolve trong tổng ngân sách 3 giây, nhưng latency chỉ đo kết nối và
fallback. Mỗi địa chỉ được chia thời gian còn lại; một địa chỉ không phản hồi
không dùng hết deadline của các lựa chọn khác. Fixture kiểm tra IPv4/IPv6,
nhiều địa chỉ, refusal, DNS failure/empty, port 0, cancellation, chia ngân sách
và đóng connection.

DNS đo latency trong 3 giây. Server rỗng dùng resolver hệ thống; server được
chỉ định dùng Dial closure riêng, đúng port số và IPv6 có scope. Cancellation và config server lỗi bị từ chối trước shortcut literal/hosts của Go. Test có UDP IPv4/IPv6,
NXDOMAIN/empty, server/config lỗi, không phản hồi, timeout/cancellation, recovery
và resolver mặc định hoạt động nhưng không được thay custom server lỗi.
Giữ semantics Go đối với IP literal và hosts file; không tạo adapter truy vấn
DNS authoritative riêng.

HTTP giữ GET, redirect, User-Agent, status >=400 là lỗi, đóng response body,
timeout/cancellation. Server local kiểm tra redirect, lỗi status/URL, connection
đóng và latency thực. HTTPS xác minh mặc định; opt-out riêng theo monitor/client.
Fixture trusted, self-signed/unknown CA, sai hostname, expired và chuỗi
secure→insecure→secure chứng minh cách ly. TLS WebSocket độc lập giữ bộ trust/
rotation P009; không bypass TLS toàn cục.

ICMP giữ thứ tự raw socket → datagram → executable. Root không chứng minh quyền
socket và iOS không dùng CAP_NET_RAW Linux. Lỗi permission/protocol/unsupported/
not-found làm vô hiệu cache; mode không hỗ trợ bị hạn chế thử lại trong 5 phút.
IPv4/IPv6 có cache riêng. Deadline socket theo caller nếu sớm hơn; cancellation
đóng socket. Chỉ reply khớp source/type/code/ID/sequence/token tạo RTT đo được.
Giữ token mật mã 16 byte bảo vệ sequence wrap và delayed/duplicate reply.
Không dùng thời gian chạy process hoặc số RTT tự tạo khi socket thất bại.

Recipe Procursus `makefiles/network-cmds.mk` đóng gói Apple network_cmds-669
ping/ping6 là căn cứ source, không chứng minh executable có trên thiết bị.
Tra executable và kiểm chứng `-n -c 1` trên 127.0.0.1/::1 độc lập trong tối đa
1 giây, giới hạn output và yêu cầu RTT parse được. Cache kết quả unavailable;
cancellation caller không làm hỏng cache capability. Probe target thực vẫn
trong deadline ban đầu 3 giây. Không shell injection, sudo/root escalation,
entitlement hoặc sửa cấu hình thiết bị. Giữ arguments Linux/Windows/macOS/BSD;
command lỗi hoặc thiếu RTT không thể thành công.

## Lifecycle, chứng chỉ và collector

Test bao phủ sync ban đầu/lặp/rỗng, upsert/delete, đổi interval/target,
probe lịch/tức thời/chia sẻ/overlap, cancellation/publication, resync reconnect
WS/SSH, shutdown/resume, chờ certificate, giữ/reset lịch sử, chuyển đổi loss và
sample sai thứ tự. Không giữ task/history publication lock khi network I/O
blocking. Probe chậm chạy tuần tự, bỏ tick đã lỡ thay vì xếp backlog vô hạn.
Probe/certificate có deadline; cancellation giải phóng timer/socket/subprocess.
Test cert có expiration, rotation, inspection lỗi, timeout/cancellation,
retry/recovery và metadata chỉ gửi một lần. Đọc leaf không tin cậy không đổi
kết quả health HTTPS.

[Bảng capability](ios-collector-capability-matrix.md) nêu từng collector local.
Map optional không hỗ trợ absent/empty theo schema cũ; scalar default 0 không
chứng minh đo được 0. Hub giữ GPU, SMART/NVMe, ZFS/Btrfs, systemd, container,
sensor từ Agent Linux/macOS/Windows remote. P011 sở hữu telemetry Procursus/APT
read-only, P012 sở hữu Wi-Fi RSSI; chỉ giữ fallback an toàn, không thêm adapter
của hai milestone này.

## Validation và giới hạn acceptance

[JSON validation](ios-network-monitors-validation.json) ghi lệnh/kết quả,
SHA/tree/blob lấy từ Git và metadata artifact native. Gate yêu cầu focused
probe/ICMP/lifecycle/TLS/collector/protocol, race liên quan, canonical Go có tag,
hồi quy P007–P009/audit/shell, Bun frozen/catalog/test/build production thật,
embed Hub thực, build Linux Agent/Hub và go-list ios/darwin/linux ARM64 CGO.
Fixture mutation từ chối source không hỗ trợ hoặc embed giả/thiếu.

Phân biệt test algorithm/permission/subprocess với socket thật. Bốn subtest
socket host raw IPv4, datagram IPv4, raw IPv6, datagram IPv6 đều SKIP vì
EPERM/EACCES. **SKIP là unavailable, không phải PASS.** Không khẳng định socket
iOS, RTT phần cứng hiện tại, thực thi A7 hoặc độ tin cậy mạng iPad hoạt động.
P018 / Gate F sở hữu ICMP/trust/DNS/mạng thực, A7 và service/reboot trên iPad4,4 /
iOS 12.5.7. Không sửa bằng chứng thiết bị 0.19 lịch sử. Build full Agent Windows
optional vẫn có giới hạn LHM assets đã biết; compile transport portable và test
schema remote không chứng minh runtime Windows.

IOS-AGENT-04 và IOS-AGENT-06 chỉ PASS phần source/fixture/native-build P010 sau
khi các check đã ghi PASS và merge thông thường đã review hoàn tất. P010 không
hoàn thành Gate C: còn P011/P012 và acceptance vật lý liên quan. Không cài thiết
bị, sửa quyền jailbreak/entitlement/LaunchDaemon, truy cập DB thật, chạy release
job, tạo tag, publish registry hoặc deploy. CI HEAD cuối và xác minh ref/tree
sau merge được ghi tại [PR #12](https://github.com/nghianguyen150612/Beszel-iOS/pull/12).

## Bằng chứng source-candidate đã thực thi

Candidate source production cuối kiểm tra local:
`dac735270cf8caf85ef8d8473e24b49e2fd7b743`. Canonical Go và race đầy đủ ở các
package liên quan (Agent, battery, delta tracker, entity, Hub WS/SSH, systems)
PASS; P010 race cuối lặp 3 lần, DNS/correlation/lifecycle 10 lần, ICMP focused 5
lần, TLS/protocol 3 lần đều PASS. Có 23 hàm test P010 portable mới và một test
ZFS riêng iOS được chọn với tag `testing`; test iOS này không được tính là đã
chạy trên host hoặc thiết bị thật. 49 audit test, matrix/readiness, upstream sync
10, iOS sentinel 45, bootstrap 54 và installer 814 assertion PASS. Bun 1.4.0
frozen/catalog, 44 test và build production thật PASS. Binary test transport
WS/SSH Windows compile; thiếu LHM assets full Agent Windows là giới hạn cũ riêng.

Native source-candidate [iOS Build 38657535072](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38657535072)
và [installer 38657538922](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38657538922)
PASS tại `14a778f1ad56bd8f7e251f9e61b196f6725dc953`. Sửa DNS/ZFS sau đó yêu cầu
CI HEAD cuối; chỉ hai run candidate này chưa đủ cho phép merge. Verifier độc lập
xác nhận hai binary ARM64 MH_EXECUTE, platform iOS 2/minimum 12.0,
CGO=1/GOARM64=v8.0, VCS sạch đúng commit, Go 1.27.1, checksum và 95 file frontend
thật embed trong Hub native. Runner dùng Bun 1.4.0, Xcode 15.4, iPhoneOS SDK 17.5,
SDK clang, GOROOT patch riêng, hash runtime gốc không đổi, YIELD có guard/không
truy cập counter ở cả hai binary linked và hai build cache độc lập giống byte.
Release job: **SKIPPED**. Agent candidate SHA256:
`a824edd2afcc8be5acd754905cef07c58987f27afd8e79ffa097827f1a8653f4`;
Hub: `f880491c82c03ecf5d5bd1b87c9d020af388f01b04b4c9d1f79398ed4a38354c`.
ZIP tải về khớp digest artifact GitHub. CLI ban đầu gặp Forbidden tại storage
redirect; tool download artifact GitHub đã kết nối hoạt động, không sửa network
policy, credential hoặc TLS verification.

IOS-AGENT-04 / IOS-AGENT-06: acceptance source/fixture **PASS**, native candidate
**PASS**. Gate integration P010 **chỉ PASS khi** CI/review/merge thông thường
HEAD cuối và xác minh sau merge trong PR #12 đều thành công. URL/hash run cuối
và merge SHA ghi tại PR để tránh commit evidence tự tham chiếu. Thủ tục phần
cứng P018 chưa chạy.
