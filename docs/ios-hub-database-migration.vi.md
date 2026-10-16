# P013 — Di chuyển cơ sở dữ liệu Hub, khôi phục và phân quyền

**Trạng thái:** Migration Hub v0.19→v0.21 thật, backup/restore đầy đủ, rollback, 5 ca recovery, kiểm tra API/bảo mật và native CI cuối đều đạt. Workflow migration [38848300763](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848300763), installer [38848300779](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848300779), và iOS Build [38848296639](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848296639) cùng chạy trên code head `db82eb3b4d2066182292d1601b5e2e763ed05f68`. Chứng cứ và artifact được ghi phía dưới và trong manifest JSON. PR #15 vẫn Draft trong khi checkpoint tài liệu và CI bắt buộc trên head đó hoàn tất.

## Quyết định P012

```text
P012 native Wi-Fi RSSI:
DEFERRED — REFERENCE DEVICE UNAVAILABLE

P012 fallback/source/CI:
ACCEPTED FOR INTEGRATION

Native RSSI measured:
NO

Normal iOS Wi-Fi RSSI:
UNAVAILABLE / NIL

P018 physical validation:
PENDING

P013:
AUTHORIZED
```

PR [#14](https://github.com/nghianguyen150612/Beszel-iOS/pull/14) đã được merge bình thường vào `ios/upstream-v021-integration` với merge SHA `d6239915e404668c10453ab2ec44f797b2f0178e`. Lý do được chủ dự án chấp thuận ghi trong [bình luận PR](https://github.com/nghianguyen150612/Beszel-iOS/pull/14#issuecomment-6206076758). RSSI native chưa được đo. `WIFI-ABI-12`, `WIFI-ASSOCIATION-12`, `WIFI-AUTH-12` và phần kiểm tra vật lý của `WIFI-BOUND-12` tiếp tục được hoãn cho P018. P016 phụ trách mọi thay đổi signing, entitlement hoặc đóng gói helper nếu có bằng chứng cần thiết. Bằng chứng vật lý của Gate C vẫn chờ; quyết định này không cho phép phát hành.

## Nguồn và fixture

- P012 feature HEAD đã xác minh: `2a16d0f46f6fd1c91f40e295190b89fb8713b8cd`; integration trước merge: `33840093bb465f0b164bb7d66aebc8cd85f86a4e`.
- Merge SHA P012 / SHA integration bắt đầu P013: `d6239915e404668c10453ab2ec44f797b2f0178e`.
- Nhánh: `ios/hub-database-migration`; base: `ios/upstream-v021-integration`.
- Nhánh production `iOS` giữ nguyên `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`; `main` đóng băng giữ nguyên `bb23f99b3eeaced9608e11e280e7c986dba650be`.
- Fixture được khởi tạo và mở lại bằng Hub v0.19 thực tế từ source production `0c3e4cd22ac24264834c401ad412e44aeda4e4f3` (Hub SHA-256 `bdcdf9457e6141575349b8a7967b133be4d866a1c7747b37be3585b0f679ec9b`, Agent SHA-256 `454f82b7cbe74b891c183a0c6a4fa3561f272db6b83d27c606fbde73186162be`, PocketBase `v0.40.2`). Không tạo cơ sở dữ liệu hiện tại rồi gắn nhãn v0.19.
- Hub v0.21 được build từ nhánh feature với Go `1.27.1`, Bun `1.4.0`, PocketBase `v0.40.4`.
- Tất cả dữ liệu nằm trong thư mục tạm biệt lập, ngoài `/var/lib/beszel-hub`. Tài khoản, mật khẩu, OAuth, Agent token và SSH key đều là dữ liệu tổng hợp; file DB và private key không được upload hay commit.

Recipe và trình tạo manifest: [`harness.py`](../tests/hub-migration/harness.py), [`backup.py`](../tests/hub-migration/backup.py), [`rollup_probe.go`](../tests/hub-migration/rollup_probe.go). Fixture gồm hai user (admin/superuser và user thường), hai dòng user settings, một system thuộc cả hai user, system details, Agent fingerprint/config, SSH identity của Hub, lịch sử system/container, container, systemd service, SMART, ZFS pool, alert và alert history. Manifest lưu ID, vai trò, quan hệ, timestamp, số lượng, hash/mode cấu hình, fingerprint khóa public, schema, migration ledger và SQLite integrity; không chứa dữ liệu bí mật.

## Migration và bảo toàn dữ liệu

Cơ sở dữ liệu v0.19 thực tế có 12 migration ledger entries. Hub v0.21 bổ sung đúng một lần, đúng thứ tự 5 migration:

1. `0_collections_snapshot_0_20_0.go`
2. `1790193183_network_monitor_cert.go`
3. `1790273174_network_monitor_server.go`
4. `1790924727_zfs_pool_status.go`
5. `1790964708_network_monitor_skip_tls_verify.go`

Ledger sau migration có 17 mục; 12 mục cũ được giữ nguyên. ID và role user, ID hai settings, quan hệ thành viên system, Agent identity/config và fingerprint SSH public key không đổi. Lịch sử system/container, alert/history vẫn truy vấn được; SQLite `integrity_check` trả `ok`. Lần khởi động v0.21 thứ hai trả health HTTP 200 và không đổi schema hash, số lượng record, ledger hoặc identity đã kiểm tra.

Migration thêm `network_monitors` và `network_monitor_stats` cùng quan hệ system/monitor và các index cấu hình/lịch sử đã chấp nhận. Field monitor gồm `certInfo` (JSON), `server` (text), `skipTlsVerify` (bool). Migration thêm `alerts.state` (JSON), thêm `NetworkMonitorLoss` vào options của `alerts.name`, siết ownership rule create/update của alert, thêm index system/name cho alert, thêm `alerts_history.monitor_name` (text) và `zfs_pools.status` (text). Fixture v0.19 thật đã có `containers.updatable`, `zfs_pools.display_name` và `zfs_pools.raw`; test xác nhận chúng giữ nguyên. Harness so sánh toàn bộ schema, chỉ cho phép các thay đổi đã quan sát, đồng thời kiểm tra rule collection monitor mới, mọi index, relation và collection cũ. Schema đầy đủ nằm trong [`ios-hub-database-validation.json`](ios-hub-database-validation.json).

## Backup, rollback và phục hồi khi bị ngắt

Quy trình cold backup dừng Agent writer và Hub, xác nhận process đã thoát, kiểm tra dung lượng rồi lưu toàn bộ data directory và checkpoint state. Archive gồm `data.db`, WAL/SHM nếu có, `auxiliary.db`, cấu hình, SSH identity tổng hợp, binary Hub/Agent v0.19, service definitions, installer state và metadata file. Quy trình tạo và kiểm tra SHA256, inventory, rồi restore vào thư mục tạm riêng. Negative test bao gồm thiếu file, writer còn chạy, thiếu dung lượng, checksum sai và tar bị ngắt.

Hosted P013 run [38848300763](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848300763) đã đạt tại source `db82eb3b4d2066182292d1601b5e2e763ed05f68`. Artifact tổng hợp [12023723159](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848300763) có SHA-256 `01793741b798c851b6ade669be3c3d568928bb04e41d80e618ed56ef57d6a5af`; manifest SHA-256 `ce7b307de85eb5d9238227a06ea9d15bc1bdf03a6c77319486a449ace4845a17`. Archive SHA-256 `acd575d1cba0a530d744899c453908bb64d5c8413f7a199459f6303345ee70a9`; inventory SHA-256 `5396858d151cbb090211407717c7d09b5d7428d4d742c686ab0e536993f744d0`. Inventory có 24 entry, gồm data DB và sidecars, auxiliary DB và sidecars, config, SSH identity, binary cũ, service, install-state. Manifest công khai ghi path/type/size/UID/GID/mode đã lọc; config và SSH key là `0600`, binary `0755`, plist/install-state `0644`. Artifact chỉ chứa manifest tổng hợp và log, không chứa DB archive hoặc byte private key.

Restore riêng bằng Hub v0.19 trả `/api/health` HTTP 200. Hai account ID và role, hai settings ID, system ID/member, metric WAL-backed `megyurmgq1eaqz7`, config hash `d3606a4014c9d50145ac52b426dc564ee149d1435dbec88d68208e8448062d7b`, fingerprint `SHA256:1hCMtN4vBwY33YpGukPPOERoXqyO+YaElR1Q26giFBE` và ledger cũ đều được giữ.

Sau migration v0.21, candidate được giữ riêng. Phục hồi cách ly candidate, xác minh archive gốc, giải nén checkpoint đầy đủ, trả lại hai executable và service/config/installer state ban đầu, sau đó chỉ mở bản restore bằng v0.19. Hub cũ trả HTTP 200; schema hash, ledger, counts, ID account/system/history, config hash và SSH fingerprint khớp. SQLite integrity là `ok`. Không coi rollback binary-only là đạt.

Năm ca lỗi có kiểm soát: trước khi gọi migration; giữa chuỗi migration tại `1790273174_network_monitor_server.go`; sau schema nhưng trước startup bình thường; readiness trả HTTP 503 trước lần health đầu; restart có kiểm soát sau khi v0.21 từng healthy. DB đã migration một phần không được mở bằng v0.19. Mỗi ca restore checkpoint đầy đủ đã xác minh rồi kiểm tra health và invariant v0.19. Artifact có log và hash từng ca.

## API, xác thực, phân quyền và lịch sử

Hub đã migrate trả `/api/health` HTTP 200. User tổng hợp đăng nhập; settings, systems, membership và history còn nguyên. Harness kiểm tra tạo/cập nhật/đọc/liệt kê monitor; tạo một dòng kết quả monitor tổng hợp qua superuser API; truy vấn kết quả bằng history list được cấp quyền; kiểm tra quan hệ/giá trị; restart Hub và xác nhận dữ liệu còn; rồi xóa monitor và kiểm tra history cascade. Alert được tạo/cập nhật/liệt kê/xóa và alert history truy vấn được. Nonmember không đọc/sửa monitor hoặc alert của user khác; query monitor/alert/history không lộ record ngoài quyền.

Hosted harness inject lỗi khi lưu superuser đầu tiên sau khi ghi regular user. Kết quả HTTP 500, không có regular user và chỉ còn placeholder `_@b.b`. Placeholder này tồn tại đúng một bản qua restart để cho phép thử lại. Hai HTTP first-user request thật chạy đồng thời; chỉ một request thành công (`200`), request kia bị từ chối (`403`). Khi setup thành công, placeholder được thay bằng owner, còn đúng một admin user và một superuser cùng email với owner; `firstRun=false`.

OAuth: user mới luôn có `role=user`; login lại giữ role do admin đặt; thiếu provider ID hoặc email không tạo user hay external-auth link. `TestOAuthDuplicateIdentityUsesExistingAccount` xác nhận provider ID trùng với email đổi vẫn trả account cũ và giữ role/email. `TestOAuthAccountLinkingPreservesExistingRole` link một readonly account đã xác minh bằng email trong khi createData yêu cầu admin; role readonly và ID account vẫn giữ nguyên, không tạo account/auth relation trùng. Hai test gọi OAuth route thật và pass thường cùng `-race` tại `db82eb3`. Trusted header hiện fail-closed: bỏ qua header identity nếu peer không thuộc `TRUSTED_PROXY_IPS` được cấu hình rõ ràng. API test gồm header disabled, allowlist unset/empty/blank, IPv4 CIDR hợp lệ, nhiều network, CIDR sai, peer không tin cậy và header identity sai định dạng. Các phép thử parser bao gồm IPv4, IPv6, CIDR, mapped IPv4, entry sai. Đây là hardening bảo mật chủ ý; cấu hình trusted header phải khai báo IP proxy.

Systemd detail response dùng `Cache-Control: private, max-age=60` và `Vary: Authorization`. Hai user tổng hợp kiểm tra request có quyền, không quyền, share system và không đăng nhập; response bị từ chối không lộ chi tiết riêng và không gọi Agent. Alert test kiểm tra CRUD ownership, membership, reassignment, subscription, delivery sau khi thu hồi quyền và history. Monitor ingestion test kiểm tra spoof monitor ID, realtime/historical write, kết nối cũ/kết nối lại và ownership.

## Retention và rollup

Chính sách v0.21 không đổi:

| Resolution | Thời hạn |
| --- | --- |
| 1 phút | 1 giờ |
| 10 phút | 12 giờ |
| 20 phút | 1 ngày |
| 120 phút | 7 ngày |
| 480 phút | 30 ngày |

Alert history giữ 200 record mới nhất mỗi user sau khi số lượng vượt 250. Test hiện có kiểm tra boundary retention cho system/container/monitor stats; alert history theo user; mẫu trùng/thiếu; response sum, sample count, loss và timestamp có trọng số; rollup monitor thưa hoặc disabled; cleanup/rollup lặp lại. Probe mới chạy lại migration ledger trên DB đã migrate rồi ngay sau đó gọi production rollup manager trên record tạm, kiểm tra aggregate system/container/monitor và chạy cleanup hai lần để xác nhận ổn định. Test cron-boundary xác nhận cleanup mỗi giờ ở phút 8 và rollup mỗi 10 phút khi startup ngay sau tick cleanup. Test dùng timestamp kiểm soát với scheduler parser; không chờ wall-clock 10 phút hoặc 1 giờ.

## Bằng chứng và trạng thái Gate

Manifest đầy đủ và hash backup/restore nằm trong [`ios-hub-database-validation.json`](ios-hub-database-validation.json); chi tiết backup trong [`ios-hub-backup-restore-evidence.md`](ios-hub-backup-restore-evidence.md); bảo mật trong [`ios-hub-security-validation.md`](ios-hub-security-validation.md). JSON validation checkpoint trước được giữ nguyên trong [`ios-hub-database-validation-prior-checkpoint.json`](ios-hub-database-validation-prior-checkpoint.json), bảo toàn các trạng thái pending/blocked P012 và P013 trước đó như chứng cứ lịch sử.

Regression checkpoint P013 trước đó `1a0fde78` đã pass audit Python (54), upstream-sync (10), patch sentinels (45), bootstrap (54), installer (814), Bun 1.4.0 install/catalog/frontend tests và production build, Linux Agent/Hub build; frontend thật có 95 file (88 JavaScript asset). Canonical Go suite pass tại code head `db82eb3b4d2066182292d1601b5e2e763ed05f68`. Race chạy mới pass cho migration, users/OAuth, bootstrap, trusted proxy, cache policy và monitor ownership/systems; test OAuth link qua email đã xác minh pass thường và `-race`. Race suite focused P013 trước đó cho alert, records/retention cũng pass. Lần chạy race Hub rộng hơn từng phát hiện race có trước trong `TestAgentWebSocketIntegration`: test đọc `connectionManager.State` không khóa trong khi writer dùng mutex. Không che giấu race không liên quan này.

Bảy acceptance P013 đã có chứng cứ migration/restore hosted tổng hợp trong run `38848300763`; trạng thái từng case ở `ios-hub-database-validation.json`. Native iOS CI cuối tại code head `db82eb3` đạt trong [run 38848296639](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848296639): Go `1.27.1`, Bun `1.4.0`, Xcode `15.4`, iPhoneOS SDK `17.5`; Agent và Hub là Mach-O ARM64 iOS platform 2, minimum iOS `12.0`, có `CGO_ENABLED=1`, và Hub nhúng 95 frontend asset. Provenance ghi đúng revision và `vcs.modified=false`; checksum hợp lệ; không có private MobileWiFi linkage; release job bị skip. Artifact binary [12024229163](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848296639), validation [12024711683](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848296639). Đây không phải bằng chứng migration hay khởi chạy trên iOS thật. Gate D còn cần P014/P015. P018 phụ trách A7/device, gồm native RSSI; không có release hay deploy được cho phép.

## Bàn giao P014

Chỉ bắt đầu P014 khi được yêu cầu sau khi P013 merge bình thường. Giữ nguyên schema/API contract đã migrate, quy tắc proxy fail-closed và phục hồi đầy đủ data cộng binary; RSSI Wi-Fi tiếp tục unavailable, nghĩa vụ vật lý thuộc P018. Không tự động chạy P014.
