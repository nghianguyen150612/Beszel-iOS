# Kế hoạch backup, nâng cấp và rollback cơ sở dữ liệu Hub

**Dữ liệu production là dữ liệu bền vững.** `/var/lib/beszel-hub` chứa cơ sở
dữ liệu PocketBase và trạng thái liên quan. Thư mục này không bao giờ là nơi
thử nghiệm disposable. P003 không chạy migration và không tuyên bố binary v0.21
đã migrate cơ sở dữ liệu.

## Kiểm tra mã nguồn và hành vi migration

Đã kiểm tra baseline iOS 0.19 tại `origin/iOS`
(`7e29c185458f79be274bb09ca57a0be9fc1b63f7`) và target upstream đã đóng băng tại
`upstream/main` (`bb23f99b3eeaced9608e11e280e7c986dba650be`). Cả hai entry point
của Hub đều import `internal/migrations`, cấu hình PocketBase với
`DefaultDataDir`, đăng ký `migratecmd` với thư mục migration, tạo `hub.NewHub`,
rồi gọi `StartHub`/`pb.Start`. PocketBase quản lý migration ledger và chạy
migration đã đăng ký khi khởi động. Hook `onAfterBootstrapAndMigrations` của
Hub ghi rõ việc dời khởi tạo Hub sang `OnServe`, vì thao tác khởi tạo phải diễn
ra sau migration cơ sở dữ liệu. Sau đó Hub cập nhật settings và quyền collection
trước khi đăng ký route, chạy server, polling và cron.

Bộ migration 0.19 có collection snapshot 0.19 cùng các migration riêng cho
field pool/container. Upstream v0.21 thay snapshot bằng
`0_collections_snapshot_0_20_0.go`, xóa migration pool cũ và thêm/đổi tên
migration cho certificate/server/TLS của network monitor và field ZFS pool.
Snapshot v0.20 định nghĩa `network_monitors` và `network_monitor_stats`, kèm
index theo system/type/created, monitor/type/created và type/created. Snapshot
cũng gồm field `updatable` của container và `display_name`/`raw` của ZFS.
Migration tiếp theo thêm `network_monitors.certInfo` (JSON), `server` (text),
`skipTlsVerify` (bool) và `zfs_pools.status` (text). Migration thêm field và
đường down xóa field; dữ liệu mới ghi vào field mới không thể mặc định còn sau
khi downgrade. Snapshot cũng chứa schema, index và rules. Hub áp dụng lại rules
authorization của alert khi khởi động. Không coi down migration hoặc rollback
chỉ binary là cách phục hồi an toàn DB 0.19.

Hành vi history thay đổi ngoài file migration. `records.go` upstream thêm rollup
per-monitor từ 1m sang 10m, 20m, 120m và 480m; Hub chạy job này mỗi 10 phút.
`records_deletion.go` giữ mẫu 1m trong 1 giờ, 10m trong 12 giờ, 20m trong 1
ngày, 120m trong 7 ngày và 480m trong 30 ngày cho system/container/monitor
statistics. Job xóa hàng giờ cũng giữ tối đa 200 alert-history mới nhất mỗi
user sau khi vượt 250 record. Đây là retention có chủ đích nhưng lần chạy đầu
sau nâng cấp có thể xóa dữ liệu. Fixture disposable phải kiểm tra boundary với
clock kiểm soát, đồng thời xác minh monitor rollup và system history riêng.

Thư mục trên thiết bị có `data.db`, `data.db-wal` được ghi gần đây,
`auxiliary.db` và `id_ed25519`; cấu hình Hub cũng nằm trong PocketBase data
directory. PocketBase/SQLite có thể dùng `data.db-shm` và sidecar khác. Vì vậy
backup toàn bộ thư mục, gồm file ẩn và mọi sidecar hiện có, thay vì chọn một vài
tên file. Binary Hub/Agent, LaunchDaemon plist và
`/var/lib/beszel-ios/install-state` được lưu trong checkpoint riêng.

## Phương thức backup và điều kiện trước

Dùng archive filesystem sau khi dừng service. Không copy `data.db` khi Hub đang
ghi. SQLite WAL có thể chứa transaction đã commit nhưng chưa checkpoint; chỉ
copy DB chính có thể thiếu dữ liệu. Không tự xóa WAL/SHM. Trên iPad tham chiếu,
chạy bằng root và xác nhận đường dẫn/service đúng với hệ thống đã cài.

1. Lên lịch bảo trì và đo kích thước toàn bộ data directory (`D`) cùng
   checkpoint set (`C`). Trước khi archive, cần ít nhất `D + C + 20%` dung
   lượng trống trên nơi lưu backup, `D + 20%` trên volume dữ liệu để quarantine/
   restore, và `3 × D + C + 20%` trên scratch storage cho fixture, bản migration
   và bản restore. Nếu thiết bị không đáp ứng được, dừng và bố trí storage ngoài
   được bảo vệ; không xóa application data để lấy chỗ. Lưu backup tại đường dẫn được bảo vệ,
   ví dụ `/var/backups/beszel-hub/<UTC timestamp>/`, ngoài staging ứng dụng và
   ngoài `/var/lib/beszel-ios`.
2. Ghi phiên bản Agent/Hub, checksum binary, owner/mode, trạng thái LaunchDaemon,
   installer state, danh sách file DB và dung lượng còn. Không ghi nội dung file
   hoặc secret.
3. Dừng Agent trước để ngừng stats mới, sau đó dừng Hub để các writer
   PocketBase/SQLite và cron thoát. Trên iPad, unload
   `dev.beszel.agent` rồi `dev.beszel.hub`:

   ```sh
   launchctl unload /Library/LaunchDaemons/dev.beszel.agent.plist
   launchctl unload /Library/LaunchDaemons/dev.beszel.hub.plist
   ```

   Xác nhận cả hai process đã thoát. Nếu writer còn chạy thì hủy bước backup;
   không archive DB đang hoạt động.
4. Tạo thư mục backup chỉ root đọc/ghi (`0700`), gồm `hub-data.tar` (toàn bộ
   persistent data tree), `checkpoint-state.tar` (binary/plist/installer state
   chính xác để khôi phục service 0.19), `SHA256SUMS` và inventory operator ghi
   version, timestamp, path, owner/mode, size và dung lượng trống. Inventory
   không chứa secret value. Dùng archive bảo toàn cấu trúc,
   mode, ownership, symlink và mọi file. Có thể dùng BSD `tar` trên iPad sau khi
   xác nhận các option trên thiết bị trong lần bảo trì production được duyệt:

   ```sh
   umask 077
   backup_id=$(date -u +%Y%m%dT%H%M%SZ)
   backup_dir="/var/backups/beszel-hub/$backup_id"
   mkdir -p "$backup_dir"
   chmod 700 "$backup_dir"
   tar -cpf "$backup_dir/hub-data.tar" -C /var/lib beszel-hub
   tar -cpf "$backup_dir/checkpoint-state.tar" -C / \
     usr/local/bin/beszel-agent \
     usr/local/bin/beszel-hub \
     Library/LaunchDaemons/dev.beszel.agent.plist \
     Library/LaunchDaemons/dev.beszel.hub.plist \
     var/lib/beszel-ios/install-state
   (cd "$backup_dir" && shasum -a 256 hub-data.tar checkpoint-state.tar > SHA256SUMS)
   (cd "$backup_dir" && shasum -a 256 -c SHA256SUMS)
   tar -tf "$backup_dir/hub-data.tar" >/dev/null
   tar -tf "$backup_dir/checkpoint-state.tar" >/dev/null
   ```

   Nếu file tùy chọn không tồn tại, ghi nhận và archive các file bắt buộc còn
   lại. Không bỏ qua file đang tồn tại. Trên iPad owner restore thường là
   `root:wheel`; Linux test harness dùng owner phù hợp fixture, thường là
   `root:root`. Giữ mode đã ghi; không chmod đệ quy cây live.
5. Xác minh manifest, danh sách archive, dung lượng, đường dẫn DB/key kỳ vọng,
   owner/mode và dung lượng còn. Lưu thêm bản thứ hai ở storage được bảo vệ,
   giới hạn quyền truy cập. Manifest chỉ chứa hash, không có credential hay DB
   content.
6. Thử giải nén và khởi chạy Hub 0.19 trên **bản restore disposable riêng**
   trước khi chấp nhận checkpoint cho nâng cấp production.

Backup nhạy cảm vì chứa account, session/config, lịch sử, private SSH key Hub,
metadata service và sidecar SQLite. Giữ quyền root-only và mã hóa khi lưu; không
đẩy lên GitHub, đính kèm PR hay đặt trong staging disposable.

## Kiểm thử migration offline trên dữ liệu disposable

Chỉ chạy trên scratch VM/workspace hoặc data directory thiết bị cô lập. Không
bao giờ trỏ vào `/var/lib/beszel-hub` trên iPad production.

1. Tạo DB tương thích 0.19 synthetic bằng đường khởi tạo/test-app 0.19 được hỗ
   trợ hoặc fixture repository. Không sửa snapshot hiện tại rồi gọi đó là DB
   0.19.
2. Thêm users/roles, settings, systems, config, fingerprints, metric mẫu và
   time range, alerts/history, collection rows liên quan. Ghi ID, relation,
   count, timestamp.
3. Tạo backup nhất quán; xác minh archive/hash; chứng minh bản restore test mở
   được bằng binary 0.19.
4. Clone backup sang data directory disposable mới. Chạy Hub 0.21 tương lai với
   port riêng, không dùng credential/endpoint production.
5. Thu startup/migration log và PocketBase ledger. So collection, field, index,
   rule, ID, relation, count và giá trị đại diện. Khởi động lần hai để kiểm tra
   idempotence.
6. Thử `/api/health`, login/authorization fixture, settings, systems, history,
   alert, tạo/sửa/xóa monitor, monitor history và frontend asset. Kiểm tra
   retention/rollup không xóa history fixture ngoài dự kiến.
7. Restore backup chưa đổi sang data directory disposable khác; chạy binary
   0.19 và xác nhận health cùng account/settings/system/history/key invariant.
8. Ngắt migration/startup ở checkpoint kiểm soát và thử recovery. Không mở DB
   partial bằng 0.19. Recovery phải restore toàn bộ backup và binary cũ cùng lúc.
9. Lưu recipe/hash fixture, migration log, invariant trước/sau, output restore và
   kết quả failure injection làm bằng chứng P013/P017. Binary nâng cấp chưa có
   trong P003 nên các bước này mới được định nghĩa, chưa được báo là đạt.

## Restore và khôi phục khẩn cấp

Migration lỗi, Hub không khởi động, schema/ledger sai, thiếu account/history,
hash mismatch hoặc key đổi bất ngờ là tình huống khẩn cấp. Dừng Agent trước rồi
Hub. Giữ cả hai service dừng cho đến khi kiểm tra xong recovery set. Không chạy
binary cũ trên data directory đã migrate hoặc migrate một phần.

1. Xác minh `SHA256SUMS` và danh sách archive trước khi giải nén. Nếu hash/archive
   sai, không ghi đè data hiện tại; dùng bản thứ hai đã xác minh.
2. Giữ data lỗi và log trong thư mục quarantine có timestamp ngoài staging.
   Không xóa trong lúc điều tra.
3. Restore `hub-data.tar` vào `/` để phục hồi nguyên cây
   `/var/lib/beszel-hub`; restore checkpoint gồm binary cũ, plist và install state.
   Giữ owner/mode đã ghi và kiểm tra trước khi chạy.
4. Kiểm tra hash/inventory. Nếu có `sqlite3`, chạy integrity check trên bản copy
   disposable của DB phục hồi, không chạy trên bản recovery duy nhất. Xác nhận
   `data.db`/sidecar, `auxiliary.db`, key, config và migration file trước khi start.
5. Start Hub cũ trước, kiểm tra, rồi chỉ start Agent sau khi Hub health đạt:

   ```sh
   launchctl load /Library/LaunchDaemons/dev.beszel.hub.plist
   # Kiểm tra /api/health và invariant account/system/history, không đọc secret.
   launchctl load /Library/LaunchDaemons/dev.beszel.agent.plist
   ```

   Kiểm tra reconnect và telemetry mới. Nếu health/invariant không đạt, dừng Hub
   lại và giữ cả hai service ở trạng thái dừng.
6. Nếu restore không đạt kiểm tra, giữ service dừng và dùng backup thứ hai đã
   xác minh. Không chạy lặp nhiều phiên bản Hub trên state chưa rõ.

## Cổng rollback production

Không duyệt nâng cấp production cho đến khi các điểm sau được chứng minh trên
dữ liệu disposable và operator có recovery set đã xác minh:

* Backup đầy đủ, nhất quán đã xác minh và nằm ngoài staging disposable.
* Đã diễn tập restore bằng Hub 0.19 cũ.
* Account, settings, systems, key, config và history gốc được bảo toàn.
* Schema mới, monitor/alert, retention và migration ledger được kiểm tra.
* Đã chứng minh phục hồi khi migration bị ngắt hoặc startup lỗi.
* Installer rollback khớp với yêu cầu restore DB và service state; rollback chỉ
  binary được xem là không đủ.

P003 chỉ mô tả quy trình. Tài liệu này không chạy migration, thay đổi installer,
đụng vào DB production hay phát hành binary.
