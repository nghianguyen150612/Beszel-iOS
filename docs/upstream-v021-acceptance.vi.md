# Kế hoạch nghiệm thu iOS v0.21

Tài liệu này xác định bằng chứng cần có sau khi tích hợp. P003 không build
Agent/Hub đã nâng cấp và không thử binary upstream mới trên iPad. Các ca
**planned** chưa phải là kết quả đạt. Hồ sơ 0.19 được giữ riêng và không thay
thế cổng 0.21.

## Thiết bị tham chiếu và quy tắc bằng chứng

Thiết bị tham chiếu: iPad mini 2, A1489 / iPad4,4, Apple A7, iOS 12.5.7,
Amethyst / Procursus. Giữ nguyên thư mục `/var/lib/beszel-hub`; tuyệt đối không
dùng cho thử nghiệm migration. Ghi output lệnh, checksum binary, metadata
phiên bản/build, log dịch vụ liên quan và invariant trước/sau có timestamp. Che
token và nội dung private key. Kiểm thử dùng cơ sở dữ liệu phải dùng dữ liệu
synthetic có thể hủy.

| Test ID | Mục tiêu | Điều kiện trước | Quy trình | Kết quả mong đợi | Bằng chứng cần có | Chủ sở hữu | Gate | Trạng thái |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| HIST-019-001 | Giữ hồ sơ 0.19.0 trên thiết bị làm bằng chứng lịch sử. | Hồ sơ P002 xác định iPad tham chiếu và binary 0.19.0. | Rà hồ sơ có health, Agent reconnect, telemetry, pin, account/config/history và reboot reactivation. | Ghi rõ hồ sơ là 0.19.0; không tính là kết quả 0.21. | `docs/device-validation.md`, `docs/ios-port-status.md`; không thêm tuyên bố 0.21. | P017 | Gate A | historically validated |
| A7-BUILD-01 | Build Agent và Hub iOS hợp lệ với target bắt buộc. | Checkout sạch; Go đã pin; runner macOS có iPhoneOS SDK và CGO toolchain. | Build cả hai với `GOOS=ios GOARCH=arm64 CGO_ENABLED=1`; kiểm tra Mach-O, CPU type, load command, minimum deployment và checksum. | Cả hai là Mach-O arm64; minimum deployment 12.0; checksum hợp lệ. | Log build, phiên bản Go/Xcode/SDK, output `file`/`otool`, SHA256SUMS. | P007 | Gate C | planned |
| A7-RUNTIME-01 | Xác nhận patch runtime A7 đúng hình dạng và fail closed. | Có mã nguồn runtime Go và patch script; không dùng cây production làm scratch. | Thử source `procyieldAsm` chuẩn; thử marker bị sửa/thiếu/trùng; kiểm tra output có vòng YIELD và không còn CNTVCT trong hàm. | Source đúng được patch một lần; mọi hình dạng lạ thoát nonzero và không ghi file; output không có CNTVCT trong hàm. | Hash fixture, output/exit code, diff trước/sau, phiên bản toolchain. | P007 | Gate C | planned |
| A7-DEVICE-01 | Xác nhận binary chạy trên A7 không SIGILL. | A7-BUILD-01 đạt; binary ứng viên và binary rollback đã xác minh; dữ liệu thiết bị được backup. | Trên iPad tham chiếu, chạy Agent/Hub qua nhiều lần poll/reconnect, kiểm tra exit/log; lặp lại sau reboot và jailbreak reactivation. | Không SIGILL/thoát bất thường; hai dịch vụ chạy, health đạt, Agent reconnect, timestamp metrics tiến lên. | Nhận dạng thiết bị, hash binary, log/launch, health, timestamp mới, hồ sơ reboot/reactivation. | P018 | Gate F | planned |
| IOS-AGENT-01 | Kiểm tra metadata iOS. | Agent iOS ứng viên; đã ghi sysctl và `SystemVersion.plist` tham chiếu. | Thu thập iPad4,4 rồi so OS, arch, hostname, kernel, cores, threads, CPU model với nguồn thiết bị. | Báo iOS/arm64 và giá trị phù hợp; A7 hiển thị Apple A7; giá trị tùy chọn thiếu thì để absent. | Payload đã làm sạch và output sysctl/plist đối chiếu. | P008 | Gate C | planned |
| IOS-AGENT-02 | Kiểm tra CPU, RAM, disk và network counters. | Agent ứng viên chạy trên thiết bị; đã ghi counter ban đầu. | Thu mẫu liên tiếp; so CPU/RAM với công cụ thiết bị khi đo được; thử filesystem và Wi-Fi/VPN/loopback; quan sát reset/wrap counter. | Metric hỗ trợ hợp lý; semantics disk/memory được ghi rõ; interface không hỗ trợ không tạo throughput giả. | Payload có timestamp và quan sát thiết bị; ghi rõ trường không đo được. | P008 | Gate C | planned |
| IOS-AGENT-03 | Kiểm tra phần trăm pin và ánh xạ trạng thái sạc. | Agent ứng viên có parser iOS; có thể kiểm tra khi rút/cắm nguồn. | So giá trị nguồn ioreg với payload ở trạng thái xả, sạc, chờ, đầy và thiếu/lỗi. | Phần trăm 0–100; trạng thái đúng; parser thiếu/lỗi tạo absent chứ không giả lập. | Output/fixture ioreg đã làm sạch, bảng ánh xạ, mẫu telemetry. | P008 | Gate C | planned |
| IOS-AGENT-04 | Kiểm tra payload network-monitor. | Agent/Hub ứng viên; chỉ cấu hình monitor trong môi trường disposable. | Thử probe HTTP/HTTPS hợp lệ, TLS mặc định, tùy chọn xác thực từng monitor, timeout, cancellation, serialization. | Payload đúng schema; mặc định xác thực TLS; ngoại lệ chỉ áp dụng đúng monitor; timeout không chặn stats. | Fixture/config monitor, record đã làm sạch, log và output test. | P010 | Gate C | planned |
| IOS-AGENT-05 | Kiểm tra reconnect sau lỗi WebSocket/SSH. | Agent ứng viên và Hub/SSH endpoint điều khiển được. | Từ chối/làm chậm handshake, ngắt transport, bật SSH fallback rồi khôi phục endpoint. | Agent retry không cần restart thủ công; listener chờ được đóng; một kết nối phục hồi và timestamp mới tiến lên. | Transcript fault-injection, trạng thái dịch vụ, log reconnect và mẫu mới. | P009 | Gate C | planned |
| IOS-AGENT-06 | Đảm bảo collector iOS không hỗ trợ vẫn absent. | Agent ứng viên và manifest chọn build tag. | Kiểm tra file được chọn cho iOS; yêu cầu GPU, SMART, ZFS, systemd, Wi-Fi RSSI và trường chỉ có trên OS khác khi chưa có adapter. | Trường không hỗ trợ absent/no-op, không chặn Agent; không hiển thị zero hoặc stale như số đo thật. | Danh sách `go list`, payload đã làm sạch, bảng kết quả từng khả năng. | P010 | Gate C | planned |
| IOS-HUB-01 | Kiểm tra health Hub và bảo toàn account/config/system/history trên bản nâng cấp disposable. | DB-BACKUP-01 và DB-MIG-01 đạt trên dữ liệu synthetic. | Chạy Hub ứng viên trên bản copy đã migrate; gọi `/api/health`; so ID/role user, settings, systems, config, SSH identity và invariant history. | Health trả 200 hợp lệ; ID và history fixture còn nguyên; không dùng dữ liệu production. | Status/body health, báo cáo invariant trước/sau, migration ledger, log.<br>P013 hosted run 38848300763; v0.19 restore và v0.21 API health HTTP 200; invariant account, settings, system, config, key và history trong `ios-hub-database-validation.json`. | P013 | Gate D | verified |
| IOS-HUB-02 | Kiểm tra migration, monitor mới và alert. | Hub ứng viên và DB migrated disposable có user/system/history fixture. | Kiểm tra ledger/collection; tạo/sửa/xóa monitor và alert; truy vấn history và retention; restart Hub một lần. | Migration đúng thứ tự/idempotent; monitor/alert hoạt động; record cũ được giữ. | Ledger/schema diff, API output, count/ID record, log lần chạy thứ hai.<br>P013 hosted run 38848300763; ledger 12→17, schema, monitor result/history, alert, rollup/retention và lần khởi động thứ hai trong manifest. | P013 | Gate D | verified |
| IOS-HUB-03 | Thực thi ranh giới phân quyền. | Ít nhất hai user synthetic có quyền system khác nhau; Hub ứng viên. | Thử bootstrap user, OAuth role, proxy trust, monitor ownership, alert CRUD/delivery và cache header nhạy cảm. | Không có truy cập chéo; OAuth mới là user; proxy theo allowlist; response riêng tư và vary theo Authorization. | Output API, assertion role/owner, header; không dùng credential thật.<br>API hosted tổng hợp và Go security test; bootstrap, OAuth, proxy fail-closed, cache header, ownership monitor/alert và từ chối cross-user trong `ios-hub-security-validation.md`. | P013 | Gate D | verified |
| IOS-HUB-04 | Kiểm tra asset frontend nhúng. | Cài Bun đã pin sạch và Hub build có `dist` mới. | Xác minh manifest asset; mở login/dashboard/alerts/monitor/localization qua Hub ứng viên. | Asset được phục vụ từ Hub; model API và field absent render đúng; không bundle thiếu/cũ. | Phiên bản/log Bun, hash asset, browser/network smoke. | P015 | Gate D | planned |
| IOS-HUB-05 | Giữ telemetry từ xa Linux/Windows/macOS. | Hub ứng viên; fixture Agent từ xa cho từng OS. | Gửi metric/monitor/alert đại diện từ Linux, Windows, macOS; truy vấn API/history và dashboard. | Hub lưu/hiển thị telemetry được hỗ trợ dù Agent iOS cục bộ bỏ collector không hỗ trợ. | Output fixture theo OS, assertion API/history, bằng chứng dashboard. | P014 | Gate D | planned |
| DB-BACKUP-01 | Tạo backup nhất quán của toàn bộ Hub state. | DB 0.19 tương thích synthetic; dừng Agent trước Hub; xác nhận Hub/writer đã dừng; đủ dung lượng ngoài staging. | Archive toàn bộ `/var/lib/beszel-hub` cùng service/config/binary/install-state cần thiết; gồm `data.db`, WAL/SHM nếu có, `auxiliary.db`, key và file khác; tạo/kiểm tra manifest SHA256. | Archive đầy đủ/đọc được, hash đạt; ghi ownership/permission; backup ngoài staging; không copy DB đang hoạt động. | Bằng chứng process dừng, inventory, listing archive, manifest, mode/owner, dung lượng còn.<br>P013 hosted run 38848300763; archive 24 entry SHA-256 `acd575d1cba0a530d744899c453908bb64d5c8413f7a199459f6303345ee70a9`; inventory SHA-256 `5396858d151cbb090211407717c7d09b5d7428d4d742c686ab0e536993f744d0`; metadata mode/owner đã lọc. | P013 | Gate D | verified |
| DB-MIG-01 | Nâng DB 0.19 synthetic bằng Hub 0.21 tương lai. | DB-BACKUP-01 đạt; fixture tạo bằng ứng dụng 0.19 được hỗ trợ; không dùng production/device DB. | Thêm users, systems, metrics, alerts, records monitor tương thích, history; clone backup; chạy Hub 0.21 ứng viên trên copy/port riêng; kiểm tra migration/invariant. | Migration không mất dữ liệu; ID, quan hệ, account, setting, history và schema kỳ vọng còn; API mới hoạt động. | Công thức/hash fixture, migration ledger, báo cáo invariant trước/sau, startup/API output.<br>P013 hosted run 38848300763; fixture thật từ v0.19, ledger 12→17 đúng thứ tự, schema/record invariant, API health và startup lần hai ổn định. | P013 | Gate D | verified |
| DB-RESTORE-01 | Chứng minh rollback đầy đủ về trạng thái 0.19 từ backup. | DB-MIG-01 chạy trên copy disposable; có backup và binary 0.19 cũ. | Dừng Agent rồi Hub; restore toàn bộ data/state và binary/service config 0.19 vào vị trí disposable riêng; chạy 0.19 và kiểm tra health/history. | 0.19 chạy trên state phục hồi; user/settings/system/history và key identity còn nguyên. | Log restore, hash, health response, invariant ID/count/timestamp.<br>P013 hosted run 38848300763; phục hồi đầy đủ data và binary/config/service v0.19; health 200, schema/ledger/ID/count/role/history/key fingerprint được giữ. | P013 | Gate D | verified |
| DB-INTERRUPT-01 | Khôi phục từ migration bị ngắt hoặc startup lỗi. | Fixture 0.19 disposable và backup đã xác minh; có khả năng ngắt/fail injection. | Ngắt candidate tại các checkpoint migration/startup; không mở DB partial bằng 0.19; restore toàn bộ backup cùng binary cũ. | Khôi phục bằng full restore; 0.19 chạy sạch với dữ liệu ban đầu; không chấp nhận chỉ rollback binary. | Điểm lỗi, log/exit code, hash restore, health và invariant phục hồi.<br>P013 hosted run 38848300763; năm điểm lỗi phục hồi bằng full backup và binary cũ; log và invariant từng ca trong artifact 12023723159. | P013 | Gate D | verified |
| IOS-INSTALL-01 | Kiểm tra install, update, repair, reconfigure, uninstall không purge. | Fixture installer cô lập; artifact xác minh; data sentinel. | Thử thao tác lifecycle, đường hủy/lỗi và uninstall giữ dữ liệu. | File/service được quản lý đúng; uninstall thường giữ dữ liệu Agent/Hub; purge cần xác nhận chính xác. | `sh tests/install-sh-test.sh` (873 đạt); installer hosted workflow 39026256211; `docs/ios-service-lifecycle.md`. | P016 | Gate E | verified |
| IOS-INSTALL-02 | Kiểm tra rollback transaction có yêu cầu restore DB. | Failure injection; binary/service cũ và backup xác minh; fixture migration disposable. | Gây lỗi trước/sau staging, migration mock, health, state commit và restore; yêu cầu phục hồi snapshot đầy đủ hoặc dừng/chờ recovery. | Binary, data, state, metadata và service state cũ cùng phục hồi; recovery chưa chắc chắn không chạy Hub cũ trên dữ liệu nghi vấn. | `docs/ios-installer-rollback-evidence.md`; genuine P013 fixture cuối 39026256183, artifact 12098566938; installer tests (873 đạt). | P016 | Gate E | verified |
| IOS-INSTALL-03 | Kiểm tra CLI status/doctor/diagnostics và thứ tự service. | CLI/installer ứng viên trên target cô lập. | Chạy lệnh chỉ đọc; start/stop/restart cả hai; gây lỗi Hub, Agent, health và reconnect. | Lệnh chỉ đọc che secret, không mutate; Hub start trước Agent; Agent stop trước Hub; authenticated connection gate trước khi commit Hub update. | `docs/ios-service-lifecycle.md`; 873 installer tests và full Go 1.27.1 regression tại source `04747f6`; installer workflow cuối 39026256211. | P016 | Gate E | verified |
| IOS-INSTALL-04 | Kiểm tra bootstrap pin và tính toàn vẹn binary. | Artifact native P015 cùng fixture lỗi, sai kiến trúc, simulator, macOS và target mới hơn. | Kiểm tra bootstrap/pin, manifest hash, executable ARM64 cơ bản, iOS device platform và deployment target; từ chối tampering trước mutation. | Chỉ executable iOS device ARM64 hợp lệ với checksum và target iOS 12.0 trở xuống mới đạt; mock signer không được xem là chữ ký iOS đáng tin. | Bootstrap tests (54 đạt); Mach-O validator (11 đạt với artifact P015 và P016); iOS Build cuối 39026267695 và metadata artifact có checksum trong `docs/ios-installer-validation.json`. | P016 | Gate E | verified |
| IOS-INSTALL-05 | Kiểm tra LaunchDaemon sau reboot và jailbreak reactivation. | iPad tham chiếu; candidate đạt; có kế hoạch reboot do operator kiểm soát và backup. | Kiểm tra load/unload/thứ tự, reboot, kích hoạt lại jailbreak semi-untethered thủ công, kiểm tra hai service. | LaunchDaemon hoạt động sau reactivation; health Hub đạt; Agent reconnect và sample mới tiến lên. | Nhận dạng thiết bị, hash plist, `launchctl`, health và timestamp mẫu. | P018 | Gate F | planned |

## Các cổng giai đoạn

| Gate | Tiêu chí nghiệm thu đo được |
| --- | --- |
| A — Sẵn sàng P003 | Có owner chuẩn cho 178 commit upstream và 13 patch iOS; cả 21 SHA P0 nối với kiểm soát cụ thể; tình huống tương thích, migration/backup, bằng chứng acceptance, rủi ro/owner/blocker được ghi; không còn rủi ro bảo mật/mất dữ liệu nghiêm trọng thiếu mitigation và gate chặn. Gate này chỉ duyệt tài liệu readiness. |
| B — Lịch sử P004–P006 | Đã tích hợp lịch sử upstream cần thiết; giữ mọi patch iOS; review diff schema/hành vi iOS; có quyết định rõ cho conflict; không deploy production. |
| C — Nền tảng và Agent P007–P012 | Agent/Hub build iOS/arm64 với CGO và metadata iOS 12; giữ giới hạn runtime A7; metric/networking Agent đạt; collector không hỗ trợ absent/no-op; hành vi remote không regression. |
| D — Hub/frontend P013–P015 | Security/authorization và migration test đạt; record hiện có sống qua upgrade disposable; API/frontend khớp; build asset nhúng đạt; triage dependency. |
| E — Installer/candidate P016–P017 | Installer/CLI, CI, checksum, backup/restore và test candidate tái lập đạt; ghi artifact release-candidate; không publish release. |
| F — Thiết bị và duyệt release P018 | Test A7 trên thiết bị thật đạt; migration/restore được chứng minh an toàn trên dữ liệu disposable; service phục hồi sau reboot/reactivation; có phê duyệt phát hành rõ ràng. |

## Chính sách chặn

Migration chưa rõ, backup/restore chưa xác minh, lỗi authorization, thiếu bằng
chứng runtime A7 hoặc rủi ro nghiêm trọng chưa xử lý sẽ chặn gate tương ứng.
Câu hỏi khả thi như Wi-Fi RSSI native có thể hoãn nếu có owner và fallback
absent/no-op rõ ràng. Chỉ bắt đầu P004 sau khi Gate A hoàn tất về cấu trúc; quyết
định đó không cho phép nâng cấp production.

## Hoãn P012 được chủ dự án phê duyệt — 2026-10-16

Quyết định của chủ dự án trong [PR #14](https://github.com/nghianguyen150612/Beszel-iOS/pull/14#issuecomment-6206076758)
hoãn RSSI Wi-Fi native trên iOS vì iPad tham chiếu chưa có sẵn. Quyết định chấp
nhận fallback unavailable/nil an toàn, kiểm tra source và cloud CI đã ghi nhận
của P012 để tích hợp, đồng thời cho phép P013 sau khi hoàn tất PR #14 theo cách
merge thường. Đây không phải bằng chứng RSSI đã đo hoặc native acquisition hoạt
động. P018 phụ trách kiểm chứng vật lý; P016 phụ trách thay đổi ký mã, entitlement
hoặc đóng gói helper trong tương lai nếu có bằng chứng. Phần thiết bị/vật lý P012
của Gate C vẫn chờ P018; quyết định này không cho phép phát hành.


## Bằng chứng P013 hosted — 2026-10-16

Workflow migration hosted [38848300763](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848300763) đạt tại feature source `db82eb3b4d2066182292d1601b5e2e763ed05f68`. Fixture được tạo/mở bằng Hub v0.19 thật, backup lạnh toàn bộ 24 entry có checksum/inventory/metadata mode, migration v0.19→v0.21, API monitor-history có record, probe rollup/retention sau migration, startup lần hai, rollback đầy đủ và 5 ca recovery đều đạt. Artifact [12023723159](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848300763) có SHA-256 `01793741b798c851b6ade669be3c3d568928bb04e41d80e618ed56ef57d6a5af`; archive hash `acd575d1cba0a530d744899c453908bb64d5c8413f7a199459f6303345ee70a9`; inventory hash `5396858d151cbb090211407717c7d09b5d7428d4d742c686ab0e536993f744d0`. Bảy hàng acceptance P013 phía trên là `verified` bằng chứng synthetic hosted và API/security; không chứng minh migration trên iOS.

Canonical Go suite pass tại source `db82eb3b4d2066182292d1601b5e2e763ed05f68`; audit Python (54), upstream-sync/patch-sentinels/bootstrap/installer (10/45/54/814), Bun frontend tests/build và Linux Agent/Hub build pass tại source `1a0fde78`. Race suite P013 pass: migration, bootstrap, trusted proxy, OAuth (gồm duplicate-identity), cache, monitor ownership, alert, records/retention và Agent reconnect. iOS Build cuối [38848296639](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848296639) đang chạy tại `db82eb3`. PR #15 còn Draft; Gate D còn cần P014/P015. P018 phụ trách kiểm tra vật lý; RSSI native vẫn DEFERRED, chưa đo và không có release authorization.


## Bằng chứng nguồn, fixture và native build của P008

[Báo cáo tương thích P008](ios-system-metrics-compatibility.vi.md) và
[JSON xác minh](ios-system-metrics-validation.json) ghi sửa nguồn, fixture
xác định/lỗi/phục hồi, hồi quy dùng chung và native build. Chúng không thay thế
quy trình vật lý ở trên:

| Trường hợp | Phần P008 xác minh được trên cloud | Phần phụ thuộc thiết bị |
| --- | --- | --- |
| `IOS-AGENT-01` | Mapping A7 chính xác, fixture sysctl/plist/count thiếu hoặc chưa biết, source selection và native linkage | P018 đối chiếu sysctl/plist iPad thật với payload |
| `IOS-AGENT-02` | Baseline/reset/lỗi CPU, phép tính RAM Mach, chọn usage filesystem, I/O vật lý không có, fixture interface động/rate/reset/cache | P018 quan sát CPU/RAM iOS thật, mount root/data và counter byte Wi-Fi/VPN |
| `IOS-AGENT-03` | Parser ioreg production, unknown state, cleanup subprocess có giới hạn, cache/lỗi/phục hồi và fixture payload | P018 đối chiếu output ioreg hiện tại và chuyển trạng thái sạc thật |

Các hàng nghiệm thu vật lý gốc vẫn **planned** đến khi thực hiện đối chiếu.
Mặc định scalar bắt buộc không xác nhận số 0 đo được khi nguồn không có;
field pin, swap và I/O đĩa vật lý tùy chọn vắng theo protocol hiện có. P008 có
thể đạt gate integration khi ghi rõ I/O vật lý không có; không khẳng định
collector đó được hỗ trợ. Gate C còn chờ P009–P012 và bằng chứng thiết bị áp
dụng; Gate F vẫn là P018. Không đổi bản ghi thiết bị 0.19 lịch sử.
