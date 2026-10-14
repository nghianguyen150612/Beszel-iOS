# P009 — Tương thích truyền tải mạng

**Cổng source/fixture/native-build P009: PASS** tại `7472f3e86ab8b5825d64cf5ef461c8dae98491ba`.
PR #11 ghi kiểm tra HEAD cuối có tài liệu và quyết định merge integration.
Kiểm tra mã nguồn, loopback, chèn lỗi và race tại cloud đã đạt.
[PR #11](https://github.com/nghianguyen150612/Beszel-iOS/pull/11) ghi SHA kiểm thử
cuối và kết quả xác minh merge, tránh vòng tự tham chiếu trong commit bằng chứng.
Gate C vẫn chờ P010–P012. Thực thi trên A7 thuộc P018 / Gate F; migration/restore
DB thuộc P013/P017. P009 không triển khai P010.

## Nguồn gốc và bảo toàn

Integration ban đầu: `212227f53d1e14ae164e0b5be77b88680b3566e0`.
Nhánh sản phẩm `iOS`: `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`.
`main` đóng băng: `bb23f99b3eeaced9608e11e280e7c986dba650be`.
Clone mới bắt đầu từ nhánh `iOS` đúng chữ hoa/thường; checkout sẵn có được giữ
nguyên. Cả ba ref khớp, không có commit integration phát sinh lúc bắt đầu.
Nhánh làm việc là `ios/network-transport-compatibility`, chỉ nhắm
`ios/upstream-v021-integration`, dùng commit thông thường và merge commit.
**178/178** commit upstream vẫn reachable; **13/13** bản vá iOS logic vẫn được
kiểm kê. Bằng chứng P001–P008 không đổi. JSON ghi hash tập SHA và danh sách patch.

## Kiến trúc và hành vi trước sửa

Giữ connection manager hiện có, ưu tiên WebSocket outbound. Sau TCP và HTTP
upgrade, Hub gửi challenge fingerprint bằng CBOR. Agent kiểm tra chữ ký SSH của
Hub trên token, gửi fingerprint rồi mới sẵn sàng. Handler registry từ chối yêu
cầu được bảo vệ trước xác thực. Giao thức dùng khóa số CBOR, request ID tùy chọn,
`AgentResponse.Data` tổng quát và các trường có kiểu cũ. Request/response phải
thuộc đúng kết nối ban đầu.

Khi WS không khả dụng, Agent mở SSH listener nếu không có `DISABLE_SSH=true`.
Không đặt `HUB_URL` là chế độ SSH-only được hỗ trợ. Mỗi kết nối TCP SSH đã xác
thực chỉ được đếm một lần dù có nhiều session. Theo chính sách upstream, SSH
khỏe sẽ dừng thử WS tiếp; WS đang kết nối có thể hoàn tất xác thực và tiếp quản.
WS sẵn sàng sẽ đóng server SSH cũ. Mất SSH giữ/khởi động lại listener và thử WS.

Trước P009, callback auth/close gửi đồng bộ vào event queue một phần tử; thao tác
kết nối khi startup/ticker chặn event loop, không hủy setup/retry khi shutdown.
Proxy dialer không có timeout TCP tường minh. `wss://` bị đổi thành `ws://`.
Trạng thái sẵn sàng được công bố trước khi gửi fingerprint thành công. Tra cứu,
gửi và xóa pending request tách rời; response lỗi/không ID được đưa tới request
cũ nhất. Remote close chưa hủy ngay request manager. Hủy request SSH đã đóng
đúng client được giữ lại, nhưng `Close()` bỏ sót handshake chưa publish client.

## Lỗi xác nhận và bản sửa

| Phát hiện | Sửa và regression |
| --- | --- |
| WSS bị hạ xuống plaintext; chấp nhận scheme không hợp lệ | Giữ WS/WSS, đổi HTTP/HTTPS đúng; từ chối scheme khác và không in bí mật trong URL. |
| Callback auth/close kẹt khi event loop không nhận | Notification WS gộp, không blocking; đọc lại owner/trạng thái hiện tại. Đã tái hiện queue đầy trước sửa. |
| Setup chặn shutdown; connect xếp hàng bỏ qua hủy | Thử kết nối bất đồng bộ và join khi dừng; setup/proxy và slot có thể hủy; TCP 5s, setup tổng 10s; hủy chờ retry; dừng health/retry ticker. |
| Gửi fingerprint lỗi vẫn báo sẵn sàng | Chỉ publish xác thực sau khi gửi thành công và kiểm tra lại identity. Đã tái hiện broken write. |
| Handler cũ mượn trạng thái xác thực của kết nối mới | Kiểm tra đúng kết nối mang request; từ chối auth/close đến muộn. Test xác nhận từ chối handler và chữ ký stale. |
| Response trùng có thể block, message bị bỏ giữ bộ nhớ pool | Claim pending nguyên tử, giao không blocking, drain đồng bộ với hủy; giải phóng message lỗi/không route được; hủy pending khi close; không nhận request sau shutdown. |
| Chọn request cũ nhất không an toàn với response không ID | Serialize peer dưới `MinVersionAgentResponse`; đóng kết nối khi bỏ request để reply muộn không thỏa request kế tiếp. Peer hiện đại vẫn dùng ID đồng thời. |
| SSH close bỏ sót setup; nhiều caller bắt tay dư thừa | Serialize setup có thể hủy; Close hủy setup; kiểm tra hủy trước publish; giữ single owner và cleanup theo captured client. |
| Control whitespace bị trim trước validation header | Từ chối control không hợp lệ trước chuẩn hóa; giới hạn 64 KiB. Giữ dấu phân cách comma/CR/LF, duplicate, colon trong value và reserved header. |
| Fixture SSH dùng port cố định, leak listener và che lỗi negative-key | Port động, bind đồng bộ, pin host key tạm, join cleanup và dùng key thực sự không được phép. Không nới xác thực production. |

JSON ghi hash log tái hiện trước sửa. Không thay kiến trúc manager, thuật toán
mật mã hay các sửa P004–P006; không thay dependency, entity schema hoặc migration.

## Chèn lỗi, xác thực và ownership request

Harness dùng manager/handler Agent và `WsConn`/request manager/SSH transport thật.
Endpoint loopback chèn upgrade rejection, close tức thì, không gửi challenge,
chữ ký sai, CBOR lỗi, text frame, broken write, deadline idle, ngắt và khôi phục
nhiều lần. DNS/TCP timeout/refused được chèn tại biên dial một cách xác định;
không phải kiểm tra DNS Internet. Test quan sát retry 10s production sau lỗi
upgrade bất đồng bộ. Có concurrent connect/close, shutdown lúc setup và callback
kết nối cũ. Process Linux Agent thật trả hai mẫu không dùng cache; JSON ghi thời
điểm Hub nhận. Bộ đếm mẫu tổng hợp chứng minh mỗi lần phục hồi nhận dữ liệu mới.

Test request có 64 response đồng thời đảo thứ tự; race response/hủy; reply muộn,
trùng, wrap ID khi ID còn pending, frame lỗi, send failure, close/shutdown và
legacy serialize. Hai kết nối cố ý có ID số bằng nhau không nhận nhầm kết quả.
Pending map về zero; message bị bỏ được giải phóng. Timeout response hiện đại
không bỏ kết nối còn khỏe; timeout/hủy legacy đóng kết nối không thể phân biệt
reply. Hủy lúc write bị chặn đóng đúng socket để giải phóng I/O.

Suite Hub thật kiểm tra header/token/fingerprint và authorization, gồm token
sai/thiếu, version lỗi. `TOKEN` ưu tiên hơn `TOKEN_FILE`; file bỏ dòng trống/comment,
chỉ cho một token và từ chối nhiều token mà không log giá trị. `X-Token`/`X-Beszel`
vẫn do Beszel đặt sau chuẩn hóa case, kể cả override lặp. Lỗi header không chứa
value. Giữ hành vi file token rỗng cũ; Hub từ chối token bắt buộc bị rỗng.

## SSH, deadline và fallback

Giữ test banner/handshake im lặng và timeout, timeout tạo session, shell/write/
read/exit bị chặn, context cancellation, session đồng thời, lỗi shared connection,
khôi phục, cleanup client cũ và hủy muộn sau thay thế. Test mới kiểm tra Close
trong setup, từ chối pinned host key sai rồi kết nối lại thành công. Agent thật
được kiểm tra chấp nhận/từ chối public key. Hủy request đóng shared SSH client đã
capture; session khác kết thúc lỗi có thể dự đoán và request sau reconnect.
Version vẫn được khởi tạo trong owner lock trước publish.

**Giới hạn chính sách upstream:** `createSSHClientConfig()` của Hub hiện dùng
`ssh.InsecureIgnoreHostKey()`; Agent tự sinh host key. P009 giữ mặc định upstream
này, không tuyên bố có production host-key pinning. Callback kiểm tra nghiêm ngặt
được cung cấp vẫn được thực thi và kiểm thử. Provision/persist host key sẽ đổi
hợp đồng cấu hình/tương thích nên không thuộc các bản sửa này. Agent vẫn yêu cầu
public key Hub đã cấu hình. Smoke subprocess chỉ pin key loopback tạm quan sát
đầu tiên của process đó; đây không chứng minh trust identity server production.

## TLS, CA tùy chỉnh và trust iOS

Mặc định bật verification. Không có `CA_CERT_FILE` thì dùng trust mặc định Go.
File CA mở rộng `x509.SystemCertPool()`, không bật `InsecureSkipVerify`. File
thiếu/không đọc được, PEM sai, system pool lỗi/nil đều fail rõ ràng. CA và leaf
đều sinh tạm: IP/DNS đúng, nhiều CA, hết hạn, issuer lạ, sai hostname, sai EKU,
rotation và sửa cấu hình trust. Ngoại lệ TLS riêng monitor không cấu hình WS TLS.

Giữ HTTP/HTTPS/WS/WSS và `BESZEL_AGENT_ALL_PROXY`; dùng proxy selection cũ với
TCP/setup có timeout và hủy được. Không retry đường mạng trực tiếp sau lỗi proxy/
TLS. Clone TLS state mỗi attempt. Không dùng credentials/chứng thư production.
Không kiểm tra chứng thư Internet do public CA cấp: test root sinh tại local
xác minh cơ chế trust; public trust mặc định do Go cung cấp.

Go 1.27.1 chọn x509/Security framework họ Darwin cho iOS. Native link không chứng
minh API/trust store thực tế của process CLI trên iOS 12 jailbreak. Public/default
trust, quyền đọc CA và khả dụng API vẫn thuộc P018 trên máy thật. Giữ hỗ trợ
`CA_CERT_FILE`; không thêm private Apple API.

## Slow polling và hội tụ cấu hình monitor

Giữ timeout upstream: request stats WS 30s, deadline im lặng Hub 100s, Agent 120s,
stale-peer cutoff 3 phút. Test poll đã scale chứng minh collection chậm/timeout
request vẫn giữ kết nối khỏe, ping/pong idle hoạt động, peer im lặng lâu bị hết
hạn và dữ liệu sau được decode vào struct mới. Chỉ fixture không có SSH manager
được tắt SSH để cô lập test; không đổi fallback production. Không thêm retry
vòng lặp nhanh hoặc ping peer chết vô hạn.

Giữ full monitor replacement khi WS connect/SSH reconnect, retry sync lỗi sau
stats thành công. Test Hub có initial, update, delete/disable, empty và first-sync
failure. Test Agent SSH → WS thật kiểm tra duplicate giữ một task, SSH reconnect,
update và empty replacement qua WS reconnect. Probe inert/injected; native probe
và quyền ICMP vẫn thuộc P010. Không gửi monitor action cho Agent dưới 0.20.0.

## Bằng chứng tám cặp P003

CSV canonical P003 không đổi; bảng dưới là bằng chứng bổ sung theo đúng ID.
Không nâng các cặp thành pass iOS vật lý.

| Scenario | Bằng chứng P009 | Acceptance chính xác còn lại |
| --- | --- | --- |
| COMP-PAIR-001, iOS 0.19 ↔ iOS 0.19 | Giữ ghi nhận thiết bị lịch sử | Chỉ tham chiếu lịch sử, không chạy lại |
| COMP-PAIR-002, iOS 0.19 → Hub nâng cấp | Agent 0.19 đúng product commit build/chạy Linux với WS/SSH transport hiện tại | Chạy iOS/native Hub P017/P018 |
| COMP-PAIR-003, Agent mới → iOS Hub 0.19 | Linux Agent mới với mã WS/SSH Hub 0.19 nguyên trạng | iOS Hub/device P017/P018 |
| COMP-PAIR-004, cặp iOS mới | Linux binary/transport thật; cả hai binary iOS đã compile và xác minh | Cặp iOS thật và DB/API P013/P017/P018 |
| COMP-PAIR-005, Linux 0.21 → iOS Hub | Linux Agent thật và shared Hub transport | iOS Hub API/history/UI P014/P017/P018 |
| COMP-PAIR-006, Windows 0.21 → iOS Hub | Shared source/schema, cross-compile test WS/SSH Windows | Runtime Windows/iOS Hub P014/P017/P018; full binary bị chặn bởi LHM asset cũ thiếu |
| COMP-PAIR-007, macOS 0.21 → iOS Hub | Shared source và hosted macOS transport/race và build Agent/Hub đã pass | Remote macOS → iOS Hub vật lý P014/P017/P018 |
| COMP-PAIR-008, Agent remote cũ → Hub mới | Fixture no-ID, SSH one-shot, unknown/malformed version, typed/generic response | Phiên bản remote tối thiểu cụ thể do P017 xác nhận |

Giữ `MinVersionCbor=0.12.0`, `MinVersionAgentResponse=0.13.0`,
`MinVersionNetworkMonitors=0.20.0`. SSH legacy chỉ gửi một response JSON/CBOR;
response hiện đại giữ generic và typed field; partial reply mới không giữ field
stale. Regression JSON v1/v2 zero/absent và remote payload P008 vẫn pass. Linux
không chứng minh trust/kernel iOS hoặc đủ tám cặp native.

## Validation và giới hạn tài nguyên

Canonical Go và cả bốn package race transport pass với `env -u NETWORK`, tag
`testing no_ui`. Fault suite lặp 10 lần, gồm ba upgrade lỗi liên tiếp trước phục hồi và từ chối
response malformed/empty/trùng; queued-connect/SSH fixture repair lặp
riêng 10 lần; stale signature/cross-system assertion lặp 20 lần dưới race.
Audit/readiness/matrix, 49 Python tests, sync/sentinels/bootstrap và 814 assertion
installer đều pass. P007 target/A7 source fixture/source selection và P008
telemetry/battery pass. Linux Agent/Hub build với VCS sạch. Bun 1.4.0 frozen
install, catalog, 44 tests, build thật 95 asset đều pass; lock hash không đổi.

Mỗi manager chỉ có một connection attempt, một owner WS; mỗi SSH transport chỉ
có một setup serialize và một shared client. Giữ ticker retry 10s, attempt tối
thiểu 5s; setup tổng 10s/TCP 5s; shutdown hủy chờ retry. Dừng ticker health/retry.
Mỗi request có tối đa một message chờ, abandoned pending bị xóa. Stress fixture
có 12 chu kỳ recovery mỗi lần lặp, 64 request đồng thời và 200 race response/hủy
mỗi lần lặp. Assertion fatal của fixture nằm ngoài monitor lock; cleanup listener có hạn
chờ và socket TLS được đăng ký cleanup cả khi test lỗi. Không quan sát leak/race
tồn tại dai dẳng trong phạm vi này. Đây là
giới hạn nguồn/test, không phải đo CPU/power/RAM/reconnect storm trên A7 thật.

## Native hosted và quyết định cổng

Native source-candidate:
[38559144606](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38559144606),
SHA `7472f3e86ab8b5825d64cf5ef461c8dae98491ba`.
Installer [38559149068](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38559149068): PASS.
Xác minh artifact native: **PASS**. Cả hai ZIP tải về khớp upload digest GitHub.
Validator độc lập trên Linux xác nhận ARM64 MH_EXECUTE, iOS platform 2/minimum
12.0, Go 1.27.1, CGO=1, VCS source-candidate sạch, checksum và đủ 95 asset frontend
nhúng. Hai lượt build native Agent/Hub byte-identical; kiểm tra instruction ở
binary không strip tìm thấy YIELD có zero guard, không đọc counter register.
Toolchain nguồn không đổi; runner dùng GOROOT patch riêng và iPhoneOS SDK.
Release job bị skip. JSON ghi hash payload và báo cáo runtime/source cụ thể. SHA feature cuối và URL CI trên HEAD cuối nằm tại PR #11
và báo cáo cuối.

IOS-AGENT-05 đã có source, fixture và portable integration cloud pass. Native Agent/Hub đã pass ở cấp source-candidate.
Normal merge yêu cầu final-head CI và review được ghi tại PR #11; SHA cuối nằm
ngoài commit bằng chứng của chính nó. Kết nối iPad, mạng jailbreak gián đoạn,
DNS/socket/trust-store iOS và reboot/reactivation vẫn là A7-DEVICE-01 / P018.
Không truy cập/sửa DB production hoặc thiết bị. P009 không thực hiện release,
tag, deployment hay thay nhánh production.
