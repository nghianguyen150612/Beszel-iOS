# P013 Di chuyển và khôi phục cơ sở dữ liệu Hub

**Trạng thái:** Đang thực hiện. Hiện mới xác minh fixture v0.19 thật và lần khởi động lại bằng phiên bản gốc. Di chuyển, backup, rollback, gián đoạn, quyền truy cập, lưu lịch sử, hồi quy đầy đủ và CI iOS còn chờ.

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

## Bằng chứng hiện tại

**Fixture Phase A: PASS.** Harness đã build Beszel `0.19.0` từ source SHA production `iOS` và Beszel `0.21.0` từ cây tích hợp ban đầu của P013; frontend thật được build bằng Bun 1.4.0 và frozen lockfile. Database v0.19 tổng hợp khởi tạo 20 collection; `PRAGMA integrity_check` trả về `ok`. Fixture có hai user (admin/superuser và user thường), hai user settings, một system có ownership, một agent fingerprint và mỗi loại một record system details, system history, container, container history, systemd service, SMART device, ZFS pool, alert và alert history. Fixture gốc đã mở lại bằng binary v0.19 thật, giữ nguyên ID user, role, ID/ownership system và ID các record đã tạo.

Hash source/binary, toàn bộ collection schema, SQLite schema hash, migration ledger, số lượng, ID, quan hệ, timestamp, hash/mode cấu hình và synthetic SSH public fingerprint được ghi trong [`ios-hub-database-validation.json`](ios-hub-database-validation.json). Database fixture và private key nằm trong thư mục tạm dùng một lần và đã được xóa sau lần chạy. Checkpoint này chưa khẳng định migration hay rollback thành công. Archive backup, migration v0.21, lần khởi động v0.21 thứ hai, restore, recovery gián đoạn, API/security, ownership monitor/alert, retention lịch sử, hồi quy và CI cuối còn chờ.
