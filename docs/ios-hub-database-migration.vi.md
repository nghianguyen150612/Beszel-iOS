# P013 Di chuyển và khôi phục cơ sở dữ liệu Hub

**Trạng thái:** Đang thực hiện. Bằng chứng tổng hợp trên CI đã pass cho fixture v0.19 thật, backup toàn bộ state khi Hub dừng và khôi phục riêng bằng v0.19, migration v0.19→v0.21 và tính ổn định ở lần khởi động thứ hai. Rollback sau migration, khôi phục sau gián đoạn, API/security/ownership, retention, hồi quy đầy đủ và native CI cuối vẫn đang chờ.

## Quyết định phạm vi P012

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

PR [#14](https://github.com/nghianguyen150612/Beszel-iOS/pull/14) đã được merge bình thường vào `ios/upstream-v021-integration` với merge commit `d6239915e404668c10453ab2ec44f797b2f0178e`. iOS Build trên đầu cuối đã pass ở [run 38837362864](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38837362864); installer lifecycle pass ở [run 38837362750](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38837362750). Quyết định đã duyệt và giới hạn bằng chứng nằm trong PR #14 cùng các comment liên quan. Native Wi-Fi RSSI chưa được đo. `WIFI-ABI-12`, `WIFI-ASSOCIATION-12`, `WIFI-AUTH-12` và phần iOS vật lý của `WIFI-BOUND-12` vẫn hoãn cho P018. P016 sở hữu mọi thay đổi ký, entitlement hoặc đóng gói helper nếu sau này có bằng chứng phù hợp. Chưa có quyền phát hành.

## Mã nguồn P013 ban đầu

SHA tích hợp ban đầu thực tế của P013 là `d6239915e404668c10453ab2ec44f797b2f0178e`, được xác định sau khi merge P012. P013 bắt đầu từ cây mã này. Ref production `iOS` vẫn là `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`; `main` đã đóng băng vẫn là `bb23f99b3eeaced9608e11e280e7c986dba650be`.

## Công thức tạo fixture v0.19 thật

[`tests/hub-migration/harness.py`](../tests/hub-migration/harness.py) archive mã nguồn production v0.19 thật tại `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`, build frontend bằng dependencies khóa ở Bun 1.4.0 và build Hub binary thật. Harness build riêng Hub/frontend hiện tại. Harness chạy v0.19 trong thư mục tạm bên ngoài `/var/lib/beszel-hub`; không tạo database hiện đại rồi gắn nhãn v0.19.

Fixture chỉ dùng thông tin đăng nhập tổng hợp; nó tạo dữ liệu qua endpoint tạo user đầu tiên được hỗ trợ, PocketBase record API với superuser tổng hợp tạm thời và cấu hình hệ thống `config.yml` v0.19 được hỗ trợ. Fixture gồm administrator/superuser, user thường, cài đặt từng user, một system có cả hai user, system details, agent fingerprint/configuration, Hub SSH identity, lịch sử system/container, container, systemd service, SMART device, ZFS pool, alert và alert history. Harness khởi động lại database gốc bằng binary v0.19 thật và kiểm tra ID, role, quyền sở hữu system cùng ID record trước khi bắt đầu thử migration.

Manifest tạo ra lưu hash source/binary, application version, collection schema, SQLite schema hash và integrity, migration ledger, số lượng record và quan hệ, ID account/system, timestamp, hash và mode cấu hình, cùng fingerprint public key SSH. Database, thông tin đăng nhập tổng hợp và private key vẫn nằm trong state tạm, không được commit.

## Checkpoint bằng chứng trên CI

Feature head `415cf3914c0347be53fe02796a6f063f1bbbced0` pass ở [P013 workflow run 38840950368](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38840950368). Artifact manifest tổng hợp là `12020648066` (SHA-256 `a18e3974f2fde420ffcb539dd11fa672140792b325c1026cc5cfb03253fa4209`); SHA-256 của manifest là `f93cf69ab8432ca8ae8f2157ef184ce00a18eed3da86883299ce5ea9bde8a396`. Artifact hết hạn ngày 2026-10-30. Artifact chỉ có JSON manifest, không có database hoặc private key.

**Phase A, B và C: PASS cho lần chạy harness tổng hợp trên CI.** Runner build Beszel `0.19.0` từ production source `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`, và Beszel `0.21.0` từ feature source `415cf3914c0347be53fe02796a6f063f1bbbced0`, với Go 1.27.1 và Bun 1.4.0. Startup/bootstrap v0.19 thật khởi tạo database 20 collection; `PRAGMA integrity_check` trả về `ok`. Fixture có hai account (administrator/superuser và user thường), settings từng user, một system dùng chung, agent identity/configuration, lịch sử system/container, container, systemd, SMART, ZFS, alert và alert history.

Cold archive được tạo sau khi Hub dừng và xác nhận không còn Agent writer. Inventory 24 mục có toàn bộ Hub data tree, `data.db`, `data.db-wal`, `data.db-shm`, `auxiliary.db`, binary Hub và Agent, LaunchDaemon definitions, installer state, mode, owner và timestamp. SHA-256 archive: `41877eaabae8bba97e24a8d2dc50aa6c2b41f1a067f99759455360d8a302ee58`; inventory: `fc3d1063defbc53ff8eeaa25b28f0b2cf47fbb8a8be61f3f3a17f47db687c821`. Backup helper cũng pass test thiếu file, writer còn sống, thiếu dung lượng, checksum sai, `tar` bị ngắt và so khớp inventory restore.

Bản giải nén riêng mở bằng binary v0.19 thật: health HTTP 200; giữ nguyên ID/role account, ID settings, ID/membership system, metric đã commit trong WAL và fingerprint SSH public key. Binary v0.21 sau đó migrate clone mới từ backup và health HTTP 200. Toàn bộ entry migration cũ v0.19 vẫn còn; năm migration schema v0.20/v0.21 chạy đúng một lần và đúng thứ tự. Legacy collection và rule owner của alert còn nguyên; collection monitor mới, index/relation monitor, field certificate/server/TLS, container `updatable` và ZFS `display_name`/`raw`/`status` đã được kiểm tra. Invariant account/system/history/config/key và SQLite integrity đều pass. Lần startup v0.21 thứ hai cũng health HTTP 200, không đổi schema hash, số record, migration ledger hay identity đã kiểm tra.

Manifest đầy đủ gồm schema, ledger migration, ID/count record, timestamp/quan hệ tổng hợp, hash và toolchain nằm trong [`ios-hub-database-validation.json`](ios-hub-database-validation.json). Chi tiết archive/restore ở [`ios-hub-backup-restore-evidence.md`](ios-hub-backup-restore-evidence.md). Kết quả này **chưa** chứng minh `DB-RESTORE-01` (rollback từ database đã migrate) hoặc `DB-INTERRUPT-01` (khôi phục sau lỗi migration/startup). Các case này, API monitor/alert, security/ownership, retention, hồi quy đầy đủ và native CI cuối vẫn chưa hoàn tất. Không dùng dữ liệu thật hoặc service production; chưa có quyền phát hành.
