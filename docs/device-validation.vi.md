# Kiểm thử thiết bị Beszel iOS

Hồ sơ kiểm thử theo hướng bằng chứng cho bản port cộng đồng Beszel iOS.
File này được cập nhật mỗi khi có lượt kiểm thử hoàn thành.

## Tóm tắt trạng thái

| Hạng mục | Kết quả |
| --- | --- |
| Bộ test hồi quy tự động cho installer | **Đạt** (551 / 0; dòng tóm tắt in ra khớp số PASS) |
| Kiểm tra cú pháp `sh -n` (install.sh + tests) | **Đạt** |
| `shellcheck -s sh install.sh` | **Đạt** (sạch) |
| `shellcheck -s sh tests/install-sh-test.sh` | Ghi chú SC2329 có sẵn ở mức info trên helper mock của test harness; không có phát hiện nào ở installer |
| `git diff --check` | **Đạt** |
| Bộ test Go | Không chạy trong môi trường này (thiếu sẵn asset embed `dist/`; không liên quan installer; không đổi mã Go) |
| Ma trận end-to-end trên máy thật (gồm kiểm thử khởi động lại) | **Đạt** — xem bên dưới |

## Kiểm thử trên máy thật

**Kết quả: ĐẠT.**

Toàn bộ ma trận trên máy đã chạy trên thiết bị tham chiếu,
kết thúc bằng bài kiểm thử khởi động lại thật. Phiên bản
installer thử trên máy là `0.4.0`; khi mọi kiểm tra đều đạt, nó được
nâng lên `1.0.0` mà không đổi binary (binary vẫn là `v0.19.0-ios.1`).

Không có cụm xác nhận xóa dữ liệu (`DELETE AGENT DATA` / `DELETE HUB DATA`) nào
được gõ ở bất kỳ bước nào. Bản sao lưu an toàn bản release-candidate tại
`/var/backups/beszel-ios-rc-20260917-071749` được giữ nguyên, không xóa
hay ghi đè.

### Môi trường kiểm thử

- Thời gian: 2026-09-17 (UTC); giờ máy lúc kiểm tra sau khởi động lại là 2026-09-18 +07
- Nhánh: `ios`, HEAD `5f413619`
- Phiên bản installer thử trên máy: `0.4.0` (nâng lên `1.0.0` sau khi kiểm thử xong)
- Bản binary: `v0.19.0-ios.1` (không đổi; không sửa mã Go/binary nên không có `v0.19.0-ios.2`)
- Thiết bị tham chiếu (cấu hình duy nhất đã kiểm thử):
  - iPad mini 2 (iPad4,4 / A1489), Apple A7, arm64
  - iOS 12.5.7 (Build 16H81)
  - Môi trường Amethyst / Procursus semi-untethered
- Không tuyên bố tương thích với iOS gốc/chưa jailbreak.

### Ma trận trước khi khởi động lại

Đã kiểm thử trước khi khởi động lại:

- chẩn đoán Agent, chẩn đoán Hub
- không rò rỉ key Agent (giá trị key không bao giờ in/log/lưu)
- cấu hình lại no-op cho Agent, cấu hình lại no-op cho Hub
- sửa lành mạnh cho Agent, sửa lành mạnh cho Hub
- hủy gỡ Agent, hủy gỡ Hub
- gỡ Agent có kiểm soát, giữ nguyên `/var/lib/beszel-agent`
- cài lại Agent: plist/binary/PID/listen phục hồi, kết nối lại, chỉ số phục hồi, đo pin phục hồi
- gỡ Hub có kiểm soát, giữ nguyên `/var/lib/beszel-hub` và DB Hub
- cài lại Hub: `/api/health` 200, giữ nguyên tài khoản / systems / cấu hình / thống kê lịch sử
- Agent kết nối lại, thống kê tiếp tục tăng sau khi cài lại

### Kiểm thử sau khi khởi động lại (bằng chứng mới 2026-09-17/18)

Người vận hành khởi động lại chiếc iPad thật, để máy boot iOS bình thường,
rồi kích hoạt lại thủ công jailbreak Amethyst semi-untethered sẵn có, sau đó
xác nhận SSH key vẫn đăng nhập được. Việc phải kích hoạt lại jailbreak sau khi
khởi động lại hoàn toàn là bình thường với jailbreak semi-untethered, không tính
là lỗi của Beszel. Phiên này không khởi động lại lần thứ hai.

Định danh thiết bị (STEP 33):

- `id -u` = `1002` (user SSH không phải root)
- `uname -a` = `Darwin Nghias-iPad 18.7.0 Darwin Kernel Version 18.7.0 ... RELEASE_ARM64_S5L8960X iPad4,4 arm Darwin`
- `hw.machine` = `iPad4,4`
- `hw.cputype` = `16777228` (CPU_TYPE_ARM64)
- `kern.osrelease` = `18.7.0`, `ProductVersion` 12.5.7, `BuildVersion` 16H81

LaunchDaemon vẫn còn sau khởi động lại (STEP 34):

- `launchctl list dev.beszel.agent`: đã load, `PID = 203`, `LastExitStatus = 0`
- `launchctl list dev.beszel.hub`: đã load, `PID = 198`, `LastExitStatus = 0`
- Binary và plist đều còn:
  - `/usr/local/bin/beszel-agent` (9251952 bytes), `/usr/local/bin/beszel-hub` (30800208 bytes)
  - `/Library/LaunchDaemons/dev.beszel.agent.plist` (1022 bytes), `/Library/LaunchDaemons/dev.beszel.hub.plist` (938 bytes)

Cổng (STEP 35):

- `*.45876 LISTEN` cùng cặp ESTABLISHED `127.0.0.1:45876 <-> 127.0.0.1:49179` giữa Agent/Hub
- `*.8090 LISTEN` cùng các kết nối ESTABLISHED của Hub

Tình trạng Hub (STEP 36):

- `curl -fsS http://127.0.0.1:8090/api/health` = `{"message":"API is healthy.","code":200,"data":{}}`, exit 0

Agent kết nối lại / số liệu mới (STEP 37):

- Dòng `systems`: `iPadServer|127.0.0.1|45876|up` (host/port không đổi)
- Hệ thứ hai `MyLaptop|100.121.124.78|45876|up` vẫn còn
- Số liệu iPad mới theo từng phút vẫn về sau khởi động lại, timestamp tăng dần:
  - `2026-09-17 23:33:26.141Z`, rồi `23:34:26.083Z`, rồi `23:35:26.083Z` (đồng hồ máy 23:35:34 UTC ở mẫu cuối)
- Không lộ private key khi kiểm tra (key Hub chỉ liệt kê theo đường dẫn; giá trị key Agent không bao giờ in).

Dữ liệu / tài khoản / cấu hình được giữ (STEP 38):

- `/var/lib/beszel-agent` còn nguyên (`fingerprint`, 48 bytes, từ Sep 13)
- `/var/lib/beszel-hub` còn nguyên (`data.db` từ Sep 13, `data.db-wal` vừa ghi sau khởi động lại; `auxiliary.db`, `id_ed25519` còn đủ, nội dung key không bao giờ dump)
- `/var/lib/beszel-ios/install-state` còn nguyên với `AGENT_RELEASE=v0.19.0-ios.1` và `HUB_RELEASE=v0.19.0-ios.1`
- DB Hub không bị reset: `users` = 1 (tạo 2026-09-13), `_superusers` = 1, `systems` = 2, `user_settings` = 1, `system_details` = 2
- Thống kê lịch sử từ trước khi gỡ/cài lại Hub và khởi động lại vẫn còn (bản ghi iPad sớm nhất `2026-09-13 07:00:09.063Z`; iPad `1m` đếm 106) và số liệu mới vẫn tiếp tục sau khởi động lại.

Đo pin (STEP 39):

- Nguồn `/usr/sbin/ioreg -r -c AppleARMPMUCharger -l` (không `-a`) vẫn chạy sau khởi động lại:
  - `CurrentCapacity = 42`, `MaxCapacity = 100`, `AppleRawMaxCapacity = 3989`, `AppleRawCurrentCapacity = 1654`
  - `ExternalConnected = Yes`, `IsCharging = Yes`, `FullyCharged = No`, `BatteryInstalled = Yes`
  - (Lần đọc ngay sau khởi động lại của người vận hành là `CurrentCapacity=40`, `ExternalConnected=No`, `IsCharging=No`; sau đó máy được cắm sạc trong phiên này. Cả hai lần đọc đều chứng minh nguồn còn chạy; Beszel theo được đà tăng 40 -> 41 -> 43.)
- Beszel vẫn nhận đo pin sau khởi động lại: các bản ghi `1m` mới nhất của iPad mang `bat:[41,3]` / `bats:{"Primary":41}`, rồi tăng lên `{"Primary":43}` ở phút tiếp theo.

### Quan sát tài nguyên (chỉ quan sát, không tinh chỉnh)

- Binary Agent 9251952 bytes; binary Hub 30800208 bytes
- `/var/lib/beszel-hub` 6.2M; `/var/lib/beszel-agent` 4.0K
- `hw.memsize` / `hw.physmem` = 1019215872 (~973 MB); ảnh `vm_stat`: 1744 free pages, 95814 active, 93262 inactive, 2523 speculative, 38863 wired (page 4096-byte)
- Không quan sát được RSS từng tiến trình từ phiên SSH `uid=1002` (`ps` báo 0 cho daemon của root và không có `sudo` không mật khẩu); ghi ở đây là không quan sát được chứ không ước lượng.

## Rà soát cứng hóa tĩnh (không cần máy)

Đã rà soát bảo mật/tính đúng đắn của `install.sh` bao gồm:

- Tương tác `set -e` và mã trả về khi quay lại
- Gài/gỡ trap và tái nhập (`update_begin` / `update_end`,
  `cleanup_work_dir`)
- Xử lý INT / TERM / HUP và ngữ nghĩa exit code 130
- Đọc `/dev/tty` (`read_tty`, `ask_tty`, `wait_for_enter`) và hành vi
  khi không có terminal
- Quote đường dẫn xuyên suốt
- Kiểm tra symlink trên binary và plist đang dùng trước khi sửa
- Chặn purge theo đường dẫn chính xác (`validate_purge_path`) và không xóa
  phá hoại bằng wildcard
- Không `rm -rf` ngoài `WORK_DIR` và các đường dẫn purge được chặn
- Không `chown` / `chmod` đệ quy
- Ẩn KEY của Agent (giá trị không bao giờ in, log hay lưu vào state)
- Tính mơ hồ của parser plist (`<key>`/`<string>` một dòng; fail-closed khi
  trùng lặp hay gán sai)
- Escape XML (`xml_escape`) và giải mã (`xml_decode`)
- Chống injection cho parser state (đọc từng key bằng grep / parameter
  expansion; không bao giờ source)
- Chống injection tag bản phát hành (allowlist `tag_from_latest_url`,
  regex `valid_release_tag`)
- Hành vi ghim bản phát hành (mọi asset của một giao dịch từ cùng một tag)
- Ép checksum (tải -> `validate_sums_file` -> từng asset
  `sums_hash_for` -> `verify_file` trước khi cài)
- Ký `ldid` trước khi chạy
- Xử lý `.new` / `.bak` / `.rollback` / `.restore` còn sót
- Dọn state khi gỡ (đường dẫn chính xác, chỉ `rmdir` khi rỗng)
- Gỡ ngoại tuyến (không cần mạng hay `ldid`)
- Cài lại giữ dữ liệu (không đụng thư mục dữ liệu)

Không phát hiện lỗi cụ thể. Khi ma trận trên máy đã đạt, installer
được nâng lên `1.0.0`.

## Ma trận kiểm thử

Chú thích: **Pass** = đã quan sát và đạt.

| Kịch bản | Kết quả |
| --- | --- |
| Bộ hồi quy tự động (551 tests) | Pass |
| `sh -n` install.sh / tests | Pass |
| `shellcheck -s sh` install.sh | Pass |
| `git diff --check` | Pass |
| Hash script thô khớp repo | Pass (kiểm tra sau push; xem bên dưới) |
| Ảnh baseline thiết bị | Pass |
| Chẩn đoán (Agent + Hub, không sửa) | Pass |
| Cấu hình lại no-op (Agent) | Pass |
| Cấu hình lại no-op (Hub) | Pass |
| Sửa / khởi động lại no-op | Pass |
| Hủy gỡ (Agent + Hub) | Pass |
| Từ chối purge (gõ sai; không gõ purge) | Pass |
| Gỡ Agent (giữ dữ liệu) | Pass |
| Cài lại Agent + kết nối lại + pin | Pass |
| Gỡ Hub (giữ dữ liệu) | Pass |
| Cài lại Hub dùng DB cũ | Pass |
| Giữ tài khoản / cấu hình / lịch sử Hub | Pass |
| Bền sau khởi động lại (daemon, cổng, health, kết nối lại, dữ liệu, pin) | Pass |
| Đo pin (`ioreg` + Beszel) | Pass |
| Quan sát tài nguyên | Pass (RSS không quan sát được dưới mobile; còn lại đã ghi) |

## Ghi chú restage

File `/usr/local/sbin/beszel-ios-installer-rc` thuộc root trên
thiết bị tham chiếu vẫn chứa installer `0.4.0` và đã cố tình không
ghi đè trong lúc kiểm thử. Trước lần gọi cuối trên máy với installer
`1.0.0`, hãy restage rồi xác nhận: hash khớp, sở hữu root,
`/bin/sh -n`, Diagnose Agent, Diagnose Hub, Hub health 200, Agent
up/listening, và DB/tài khoản/cấu hình/lịch sử còn nguyên.

## Hạn chế đã biết

- Chỉ thiết bị tham chiếu trên là đã kiểm thử. Mọi iPhone/iPad khác, SoC
  (A8+), phiên bản iOS và jailbreak rootless khác đều chưa thử; đừng cho rằng
  bản vá runtime A7 là cần thiết hay vô hại ở nơi khác.
- Sau khi khởi động lại hoàn toàn, phải kích hoạt lại thủ công jailbreak
  semi-untethered thì LaunchDaemon tùy chỉnh mới chạy được. Sau khi kích hoạt,
  Agent và Hub đã được kiểm chứng là tự quay lại.
- Không quan sát được RSS từng tiến trình từ phiên SSH không root trên
  máy này; các con số tài nguyên trên chỉ để quan sát.
- Bộ test Go không chạy được ở đây vì thiếu asset embed frontend `dist/`
  trong checkout; đây là vấn đề môi trường, không phải lỗi mã, và không có
  mã Go nào bị sửa.
