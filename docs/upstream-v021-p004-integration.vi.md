# P004 — tích hợp upstream Beszel v0.20.0

Đã tích hợp mã nguồn P004; cổng tích hợp hiện **PASS**.
Gate B vẫn chờ P005 và P006. Kết quả này không xác nhận bản iOS đã sẵn sàng phát hành.

## Nền tảng và lịch sử Git

| Mốc | SHA |
| --- | --- |
| `iOS` ban đầu và nhánh tích hợp mới | `0c3e4cd22ac24264834c401ad412e44aeda4e4f3` |
| `main` đóng băng / đích roadmap | `bb23f99b3eeaced9608e11e280e7c986dba650be` |
| Điểm phân nhánh lịch sử | `f204dc17e64bba444478c60210d018e43a777193` |
| Tag `v0.20.0^{commit}` đã kiểm chứng | `a0dc19eacf6902d2f6934a3efd7ea7277198c40b` |
| Commit merge upstream | `65731f9a86f19b1668d260e49f6d5ee7318453be` |

Đã clone rõ ràng bằng `--branch iOS --single-branch`; SHA sản phẩm khớp P003.
Nhánh `ios/upstream-v021-integration` chưa tồn tại nên được tạo và push từ
`origin/iOS` đã xác minh. Nhánh làm việc là `ios/p004-upstream-v020-merge`;
PR chỉ nhắm vào `ios/upstream-v021-integration`, không nhắm vào `iOS` hoặc `main`.

Lệnh thực thi:

```sh
git merge --no-ff --no-commit a0dc19eacf6902d2f6934a3efd7ea7277198c40b
git commit -m 'merge(ios): integrate upstream Beszel v0.20.0 (P004)'
```

Parent thứ nhất: `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`.
Parent thứ hai thực: `a0dc19eacf6902d2f6934a3efd7ea7277198c40b`.
Đủ **25/25 SHA** thuộc các dòng P004 **2–26** của CSV P002 bất biến
(dòng 1 là tiêu đề). [Bản ghi JSON](upstream-v021-p004-preservation.json)
chứa từng SHA đầy đủ, dòng chuẩn, mức ưu tiên và đường dẫn mã nguồn; chủ sở hữu
chức năng lấy từ crosswalk P003. P005 còn **116**, P006 còn **37** commit;
không SHA nào của hai nhóm đó là tổ tiên của kết quả P004.
Không merge `upstream/main` di động hoặc commit chỉ thuộc v0.21.

## File thay đổi, xung đột và quyết định

Merge thay đổi **161 file**, thêm 17.672 dòng và xóa 1.156 dòng. Danh sách đầy đủ
nằm ở `merge_changed_files` trong JSON: Agent monitor/probe/lịch chạy/lịch sử,
CBOR, Hub đồng bộ/lưu trữ, alerts, snapshot, rollup, giao diện monitor/bản dịch,
phụ thuộc Go đúng v0.20, Docker và phiên bản nguồn/package.

**Không có xung đột thực tế.** Danh sách unmerged, `git diff --cc` và các hunk
của `git show --remerge-diff` đều trống. Không áp dụng chiến lược bỏ một phía;
không chép toàn bộ Agent hoặc Hub để thay thế bản iOS.
Mọi đường dẫn thay đổi khớp tree đích, trừ `docker-images.yml` kết hợp sửa
`latest=false` của upstream với loại trừ `!v*-ios.*` hiện có. Backend, bảo mật,
model và migration khớp upstream nguyên vẹn. `agent/system.go` không đổi;
vẫn giữ bỏ qua CPU probe trên iOS và hook `adjustPlatformSystemDetails()`.

Sửa riêng fixture kiểm thử systemd: lượng CPU giả lập tỷ lệ với
`runtime.NumCPU()` thay vì giả định tám core. Runner cloud chỉ có ba CPU.
Không đổi phép tính production hoặc giảm yêu cầu assertion; đây không phải
xử lý xung đột hay tích hợp thêm lịch sử upstream.

## Bảo toàn 13 bản vá iOS

Đã ghi inventory trước merge. JSON có file hiện hành, hash SHA-256 trước/sau,
chủ sở hữu và đường dẫn lịch sử không còn hiện hành. Các workflow probe cũ đã
được hợp nhất vào `ios-build.yml` từ trước P004; `readme.md` chữ thường cũng đã
được thay bằng README EN/VI trước đó. Không tính các file này là mất bản vá.

| Bản vá logic | Chủ sở hữu | Trạng thái P004 |
| --- | --- | --- |
| `IOS-A7-RUNTIME` | P007 | GIỮ NGUYÊN; triển khai hiện có không đổi |
| `IOS-AGENT-BATTERY` | P008 | GIỮ NGUYÊN; triển khai hiện có không đổi |
| `IOS-AGENT-BUILD-PROBE` | P007 | GIỮ NGUYÊN; triển khai hiện có không đổi |
| `IOS-AGENT-SYSTEM-HOOK` | P008 | GIỮ NGUYÊN; triển khai hiện có không đổi |
| `IOS-FRONTEND-BUILD` | P015 | GIỮ NGUYÊN; triển khai hiện có không đổi |
| `IOS-HUB-BUILD-PIPELINE` | P015 | GIỮ NGUYÊN; triển khai hiện có không đổi |
| `IOS-INSTALLER-ARCH-DETECTION` | P016 | GIỮ NGUYÊN; triển khai hiện có không đổi |
| `IOS-INSTALLER-BOOTSTRAP-INTEGRITY` | P016 | GIỮ NGUYÊN; triển khai hiện có không đổi |
| `IOS-INSTALLER-LIFECYCLE` | P016 | GIỮ NGUYÊN; triển khai hiện có không đổi |
| `IOS-INSTALLER-VALIDATION` | P017 | GIỮ NGUYÊN; triển khai hiện có không đổi |
| `IOS-PORT-DOCUMENTATION` | P017 | GIỮ NGUYÊN; triển khai hiện có không đổi |
| `IOS-RELEASE-ISOLATION` | P016 | GIỮ NGUYÊN; giữ sửa Docker variant và loại trừ tag iOS |
| `IOS-UPSTREAM-SYNC-CI` | P017 | GIỮ NGUYÊN; triển khai hiện có không đổi |

Giữ nguyên patch A7 `runtime.procyieldAsm`, kiểm tra hình dạng nguồn và từ chối
layout lạ. Năm fixture xác nhận nguồn Go 1.27.1 hợp lệ chuyển sang YIELD;
marker đầu thiếu/trùng, instruction lạ và marker cuối thiếu đều thất bại mà
không ghi file. Vẫn giữ `GOOS=ios GOARCH=arm64 CGO_ENABLED=1`, clang iPhoneOS SDK,
iOS tối thiểu 12.0 và xác minh Mach-O. `go list` chọn `battery.go` cùng
`battery_ios.go`; giữ AppleARMPMUCharger / `ioreg -l`, Darwin `!ios` và hook hệ thống.

Engine và bootstrap installer không đổi byte. Phiên bản **1.0.0**, state **1**,
engine pin `d4b4e381a76c75ac79cb7c99558853b62c7c2163879a72c0333acc5c40092d0c`.
Giữ LaunchDaemon riêng, CLI, checksum, ldid, tên asset, backup binary và giữ dữ
liệu. Rollback binary không chứng minh rollback DB sau migration; kế hoạch
khôi phục đầy đủ P003 vẫn là điều kiện chặn nâng cấp production.
Giữ loại trừ tag iOS trong release/Docker chung. Không tạo release tag,
phát hành release, cập nhật installer hoặc triển khai production.

## Rà soát bảo mật và migration

Các dòng P0 của P004 (6, 9, 14, 19, 21) được giữ đầy đủ:

- `50f6fc…`: bootstrap người dùng đầu tiên kiểm tra trạng thái, tạo user/admin
  và xóa admin tạm trong cùng transaction; giữ kiểm thử cạnh tranh/rollback.
- `4bf707…`: trusted header xét peer `RemoteAddr` và allowlist IP/CIDR.
  Allowlist không đặt/rỗng giữ hợp đồng opt-in upstream; cấu hình chỉ có giá
  trị sai từ chối mọi peer.
- `f7fd3e…`: user mới từ OAuth bắt đầu với role `user`; giữ role user hiện hữu.
- `b0bc72…`: snapshot v0.20 cuối thay snapshot 0.19/dev. Upstream chủ động
  gộp/xóa `1788955200_pool_updates.go` và
  `1789079746_container_update_available.go`. Không sửa identifier, nội dung
  hoặc thứ tự đăng ký migration tại iOS.
- `187dc8…`: monitor đồng bộ lại sau SSH reconnect và retry khi lỗi, kể cả
  cấu hình rỗng; không làm hỏng thu thập stats thành công.

Rule monitor/history vẫn yêu cầu đăng nhập và membership hệ thống; readonly
không được ghi. Giữ hợp đồng `SHARE_ALL_SYSTEMS=true` khi cấu hình chủ động.
Alert CRUD lấy user từ auth và kiểm tra quyền hệ thống. Không tắt auth hoặc
xác minh TLS; giữ triển khai chung HTTP(S), TCP, DNS, ICMP, deadline, cancel,
retry, đồng bộ và rollup.

Response systemd detail đã xác thực vẫn có `Cache-Control: public, max-age=60`
từ baseline. Hardening được audit cho lịch sử sau v0.20 vẫn thuộc P005/P006 và
P013/P017 Gate D. Không xóa bảo vệ hiện có hoặc cherry-pick sớm commit sau đích.

Mọi kiểm thử DB chỉ dùng test app PocketBase tổng hợp dùng một lần.
Không mở `/var/lib/beszel-hub` hoặc DB thật trên iPad. Kiểm thử collection,
quyền, bootstrap và rollup không thay thế diễn tập nâng cấp/backup/restore.
Harness seed 0.19 và gián đoạn migration chưa hoàn thành: DB-BACKUP-01,
DB-MIG-01, DB-RESTORE-01, DB-INTERRUPT-01 **DEFERRED**, P013/P017, Gate D/E.
Không xóa dữ liệu để migration qua; không chạy Hub cũ trên DB đã migrate.

## Bằng chứng kiểm thử

Toolchain scratch: **Go 1.27.1**, **Bun 1.4.0**. Giữ phụ thuộc theo đích v0.20
và lock iOS hiện có. Cả baseline và kết quả đều cài bằng
`bun install --frozen-lockfile`, build asset thật với `bun run build` rồi mới
build Hub có embed. Không tạo dist giả. Cảnh báo kích thước chunk Vite không
làm build thất bại.

| Kiểm tra | Baseline P003 | P004 |
| --- | --- | --- |
| Validator matrix | PASS | PASS |
| Validator readiness | PASS | PASS |
| Unittest audit `discover -s tests/audit -p 'test_*.py' -v` | PASS; 14 | PASS; 14 |
| `sh tests/upstream-sync-test.sh` | PASS; 10 | PASS; 10 |
| `sh tests/ios-patch-sentinels.sh` | PASS; 44 | PASS; 44 |
| `sh tests/bootstrap-test.sh` | PASS; 54 | PASS; 54 |
| `sh tests/install-sh-test.sh` | PASS; 814 | PASS; 814 |
| `git diff --check` | PASS | PASS |
| Frozen Bun install/build | PASS | PASS |
| `bun test` | Không có test; exit 1 không phải PASS | PASS; 4 |
| `env -u NETWORK go test -tags 'testing no_ui' ./...` | Fixture CPU lỗi trên 3 core | PASS; all packages after fixture repair |
| Linux Agent/Hub sau build asset thật | Không dùng lại binary lịch sử | PASS; Agent and Hub |
| Fixture runtime A7 | Review nguồn hiện có | PASS; 5 |
| Native iOS / frontend / Mach-O / checksum | Không dùng kết quả lịch sử | PASS; Agent, Hub, frontend, Mach-O arm64 / iOS 12.0, SHA256SUMS |

Các lần thử `go test ./...` và chỉ `-tags testing` thất bại trên cả hai snapshot:
thiếu helper sau build tag và route UI PocketBase trùng. Makefile quy định
`testing no_ui`. Biến `NETWORK` của cloud còn ảnh hưởng fixture CLI; chỉ loại
khỏi tiến trình test, giữ nguyên proxy/CA/credential. Lỗi giả định số CPU đã
được phân tích và sửa fixture. Không ghi các lần thất bại này thành PASS.
Suite chuẩn bao gồm bootstrap/OAuth/proxy, permission, alert, monitor và rollup.

[iOS Build 38390065511](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38390065511)
và [installer 38390070057](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38390070057)
được dispatch trên nhánh P004, commit
`65731f9a86f19b1668d260e49f6d5ee7318453be`. Job release chỉ cho tag được skip
đúng điều kiện và không được tính PASS. Commit fixture/tài liệu sau đó không
đổi đầu vào binary; được review riêng. JSON chứa kết quả và hash bằng chứng.

## Nghĩa vụ hoãn và cổng chặn

| Nghĩa vụ | Lý do / kết quả | Chủ sở hữu | Gate |
| --- | --- | --- | --- |
| Lịch sử P005/P006 | Còn 116 + 37; ngoài phạm vi P004 | P005/P006 | B |
| ICMP iOS | Quyền socket chưa kiểm chứng; exec fallback chung từ chối `ios`; lỗi là probe loss, không giả thành công | P010/P017/P018 | C/F |
| A7 không SIGILL | Build Mach-O không chứng minh chạy thật | P007/P018 | C/F |
| Metrics, transport, cặp phiên bản và collector unsupported | Hook còn nguyên; matrix P003 chưa thực thi đủ | P008/P009/P010/P017 | C |
| Nâng cấp/backup/restore/gián đoạn DB | Chưa có harness 0.19 đầy đủ; unit test không đủ | P013/P017 | D/E |
| Cache và matrix bảo mật cuối | Hardening về sau còn pending; cần negative matrix | P013/P017 | D |
| Browser và Agent Linux/Windows/macOS remote | Build asset không thay fixture E2E | P014/P015/P017 | D |
| Package update / Wi-Fi RSSI | Cần adapter/fallback ở prompt sau | P011/P012 | C |
| Installer rollback dữ liệu và candidate | Regression hiện hữu PASS; cần DB restore và candidate | P016/P017 | E |
| iPad mini 2 / A7 / iOS 12.5.7 thật | Chưa thực thi; kết quả 0.19 vẫn là lịch sử | P018 | F |

Cổng tích hợp P004 **PASS**: suite chuẩn và các CI áp dụng đã thành công.
PR chỉ được merge sau khi review diff và xác nhận các yêu cầu GitHub. Gate B–F chưa hoàn tất. Không triển khai production.
Tiếp theo là P005; chưa bắt đầu P005 trong tác vụ này.
