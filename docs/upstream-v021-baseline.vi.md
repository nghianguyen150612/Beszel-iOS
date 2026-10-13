# Baseline Beszel upstream v0.21.0 (P001)

Tài liệu này đóng băng baseline đối chiếu của Beszel-iOS cho milestone P001.
Nội dung ghi nhận trạng thái mã nguồn và các contract tương thích cho lần xem
xét sau; đây không phải là tích hợp upstream hay phân loại toàn bộ commit đang
chờ.

## Snapshot audit

- Ngày audit: **2026-10-13**, múi giờ **Asia/Ho_Chi_Minh**.
- Repository sản phẩm: <https://github.com/nghianguyen150612/Beszel-iOS>.
- Upstream chính thức: <https://github.com/henrygd/beszel>.
- Nhánh sản phẩm: `iOS` (phân biệt chữ hoa/thường).
- Các ref đóng băng:

  | Ref | SHA đầy đủ | Version mã nguồn |
  | --- | --- | --- |
  | `origin/iOS` | `a6b73365147fc8c3a820552a145d370418e5b680` | `0.19.0` |
  | `origin/main` | `bb23f99b3eeaced9608e11e280e7c986dba650be` | `0.21.0` |
  | `upstream/main` | `bb23f99b3eeaced9608e11e280e7c986dba650be` | `0.21.0` |
  | Merge-base của `origin/iOS` và `origin/main` | `f204dc17e64bba444478c60210d018e43a777193` | — |

  Version mã nguồn bên trên được đọc từ `beszel.go` tại từng ref. Khi fetch,
  ref `upstream/main` chính thức trùng với snapshot `origin/main` đã đóng băng;
  upstream chưa tiến xa hơn mục tiêu này.

## Lệnh tái lập và số commit chênh lệch

Checkout trên cloud được clone tường minh từ nhánh sản phẩm phân biệt hoa
thường, sau đó fetch riêng các ref sản phẩm và upstream chính thức:

```sh
git clone --branch iOS --single-branch \
  https://github.com/nghianguyen150612/Beszel-iOS.git Beszel-iOS
cd Beszel-iOS
git fetch origin refs/heads/main:refs/remotes/origin/main
git remote add upstream https://github.com/henrygd/beszel.git
git fetch upstream refs/heads/main:refs/remotes/upstream/main
git branch --show-current
git status --short
git rev-parse HEAD
git rev-parse origin/iOS
git rev-parse origin/main
```

Lịch sử và khai báo version được kiểm tra bằng các lệnh:

```sh
git merge-base origin/iOS origin/main
git rev-list --left-right --count origin/iOS...origin/main
git log -1 --format=fuller origin/iOS
git log -1 --format=fuller origin/main
git diff --stat origin/iOS...origin/main
git status --porcelain
git show origin/iOS:beszel.go
git show origin/main:beszel.go
git show upstream/main:beszel.go
git rev-parse upstream/main
git merge-base --is-ancestor origin/main upstream/main
```

Merge-base đã xác minh là `f204dc17e64bba444478c60210d018e43a777193`.
`git rev-list --left-right --count origin/iOS...origin/main` trả về `35 178`:
nhánh iOS có 35 commit riêng và **178 commit upstream chưa có trong nhánh
iOS**. Phép so sánh diff gồm 333 file (42.863 dòng thêm, 4.614 dòng xóa).
Tài liệu này không tuyên bố đã audit riêng từng commit trong 178 commit đó;
việc xem xét và ma trận phân loại thay đổi thuộc P002.

Con số **118 commit** là delta đồng bộ riêng của nhánh `main`, không phải delta
tích hợp iOS. Điểm tháng 9 là ancestor `origin/main`
`d80a2f49f9d00abe4c7853f0b041dcb65846814b` (2026-09-24); snapshot tháng 10 là
`bb23f99b3eeaced9608e11e280e7c986dba650be`:

```sh
git rev-list --left-right --count \
  d80a2f49f9d00abe4c7853f0b041dcb65846814b...origin/main
```

Lệnh trả về `0 118`. Audit tháng 9 trước đó được lưu trong
[`upstream-sync.vi.md`](upstream-sync.vi.md) vẫn là bằng chứng lịch sử và không
được viết lại thành snapshot tháng 10. Con số 118 commit đồng bộ `main` và 178
commit upstream chưa có trong iOS mô tả hai dải nhánh khác nhau.

## Baseline version

- Version mã Beszel hiện tại trên `origin/iOS`: **`0.19.0`**.
- Version mã nguồn mục tiêu upstream đóng băng trên `origin/main` và
  `upstream/main`: **`0.21.0`**, gồm các commit đến
  `bb23f99b3eeaced9608e11e280e7c986dba650be`.
- Khai báo version được đọc trực tiếp bằng `git show origin/iOS:beszel.go`,
  `git show origin/main:beszel.go` và `git show upstream/main:beszel.go`.
- P001 không tự đổi mục tiêu nếu upstream tiến xa hơn sau này. Milestone này
  không tạo release tag, revision binary hay deployment.

## Contract tương thích đóng băng cho công việc tích hợp

Đây là các bất biến bắt buộc cho công việc port sau này. Việc có mặt trong mã
hoặc test không có nghĩa mọi mục đều được chạy lại trong P001.

| Khu vực | Bất biến bắt buộc | Tài liệu tham chiếu chính |
| --- | --- | --- |
| Runtime và platform | `GOOS=ios`, `GOARCH=arm64`, `CGO_ENABLED=1`; deployment tối thiểu iOS 12.0; hỗ trợ Apple A7; vá Go `runtime.procyieldAsm` cho A7; từ chối vá khi dạng source runtime không được nhận diện | `.github/workflows/ios-build.yml`; `.github/scripts/patch-go-ios-arm64-runtime.py`; `tests/ios-patch-sentinels.sh` |
| Agent | Đọc pin native qua `AppleARMPMUCharger`; metadata iOS từ `sysctl` và `SystemVersion.plist`; giữ protocol và model Beszel dùng chung; bảo toàn chức năng Agent hiện có | `agent/battery/battery_ios.go`; `agent/battery/battery_darwin.go`; `agent/system.go`; `agent/system_platform_ios.go`; `agent/system_platform_other.go`; `internal/common/`; `docs/architecture.vi.md` |
| Hub | Hub chuẩn tương thích upstream, gồm hành vi PocketBase và migration; đường dẫn dữ liệu bền vững `/var/lib/beszel-hub`; không thao tác phá hủy database trong công việc port hay installer | `internal/cmd/hub/`; `internal/hub/`; `internal/migrations/`; `scripts/ios/install-beszel.sh`; `docs/architecture.vi.md` |
| Installer và service | LaunchDaemon Agent và Hub độc lập (`dev.beszel.agent`, `dev.beszel.hub`); giữ contract CLI `beszel-ios` hiện có; xác minh SHA-256; ký bằng `ldid`; backup và rollback; asset release `beszel-agent-ios-arm64`, `beszel-hub-ios-arm64`, `SHA256SUMS` | `install.sh`; `scripts/ios/install-beszel.sh`; `.github/workflows/ios-build.yml`; `tests/install-sh-test.sh`; `tests/bootstrap-test.sh`; `docs/architecture.vi.md` |
| Phần cứng tham chiếu | iPad mini 2, model A1489 / `iPad4,4`, Apple A7, iOS 12.5.7, môi trường jailbreak Amethyst / Procursus | `docs/device-validation.vi.md`; `docs/ios-port-status.vi.md` |

Thư mục dữ liệu Hub lưu tài khoản, cấu hình và lịch sử. Update và gỡ installer
thông thường phải giữ thư mục này. Xóa dữ liệu là luồng purge riêng, cần xác
nhận tường minh và nằm ngoài bảo trì thông thường.

## Trạng thái triển khai và xác thực

### Đã triển khai và có trong ref iOS đóng băng

- Workflow build iOS biên dịch Agent và Hub cho iOS arm64 có CGO, nhắm iOS
  12.0, áp dụng bản vá runtime A7 có kiểm tra, xác minh metadata Mach-O, và
  chuẩn bị hai binary cùng `SHA256SUMS`.
- File platform cho pin và metadata hệ thống iOS hiện diện. Hub, protocol,
  model và mã migration vẫn là mã Beszel dùng chung; baseline này không chứa
  implementation Agent hay Hub thay thế.
- Installer có service Agent và Hub riêng, tải release có xác minh checksum,
  ký bằng `ldid`, update qua staging, backup binary, rollback và giữ dữ liệu
  Hub.

### Xác thực lịch sử, không chạy lại trong P001

[`device-validation.vi.md`](device-validation.vi.md) ghi ma trận máy thật trước
đây, ngày 2026-09-17/18: iPad mini 2 / A1489 / Apple A7 / iOS 12.5.7 trong môi
trường jailbreak semi-untethered Amethyst / Procursus. Tài liệu ghi nhận pin và
metadata hệ thống, hoạt động Agent và Hub, vòng đời installer, bảo toàn DB và
lịch sử, cùng độ bền sau reboot khi kích hoạt lại jailbreak thủ công. Các kết
quả đó vẫn là bằng chứng lịch sử cho binary 0.19.0 và version installer đã
thử lúc đó. P001 không kiểm thử máy thật và không tuyên bố 0.21.0 đã được xác
thực trên phần cứng.

### Test baseline cục bộ của P001

Bốn lệnh đầu tiên được chạy trước khi sửa tài liệu trên checkout
`origin/iOS` sạch và chạy lại sau khi hoàn tất thay đổi P001. Cả hai lượt đều
chạy cục bộ trong cloud workspace, không phải GitHub Actions hay phần cứng
thật.

| Lệnh chính xác | Kết quả | Bằng chứng |
| --- | --- | --- |
| `sh tests/upstream-sync-test.sh` | Đạt | 10 đạt, 0 lỗi |
| `sh tests/ios-patch-sentinels.sh` | Đạt | 44 đạt, 0 lỗi |
| `sh tests/bootstrap-test.sh` | Đạt | 54 đạt, 0 lỗi |
| `sh tests/install-sh-test.sh` | Đạt | 814 đạt, 0 lỗi |
| `git diff --check` | Đạt sau khi sửa | Không có lỗi whitespace |

Bộ test Go đầy đủ bị chặn trong workspace này. Checkout không có thư mục
frontend sinh ra `internal/site/dist`, trong khi `internal/site/embed.go` yêu
cầu thư mục này qua `//go:embed all:dist`; không có Bun và lệnh
`/usr/bin/go` khả dụng không phải Go toolchain (`go test ./...` trả về
`Go: Unknown option: test`). Không tạo thư mục frontend rỗng. Build iOS CGO/Mach-O
cũng cần SDK iPhoneOS trên macOS nên không chạy được tại đây. Các thay đổi chỉ
tài liệu này không kích hoạt build Go của workflow iOS; workflow installer
riêng khớp các đường dẫn tài liệu bảo trì đã sửa và sẽ được kiểm tra trên PR.
Workflow build iOS bị bỏ qua hay không chạy không được tính là build thành
công.

### Mục chưa kiểm thử hoặc bị chặn

- P001 chưa build source 0.21.0 cho iOS hoặc chạy trên thiết bị tham chiếu.
- Hành vi iOS của toàn bộ delta upstream vẫn chưa audit cho tới P002.
- Baseline này không tuyên bố có kết quả CI hoặc máy thật cho P001.
- Các thiết bị iOS, SoC, phiên bản iOS và môi trường jailbreak khác vẫn chưa
  kiểm thử như đã ghi trong tài liệu port-status và device-validation.

## Phạm vi đóng băng cho P002 và các bước sau

P001 chỉ đóng băng các ref, khai báo version, contract tương thích và bằng
chứng test. Việc tích hợp upstream **chưa bắt đầu**: không có mã Beszel 0.20.0
hay 0.21.0 nào được merge, và không có production code, dependency, asset sinh
ra, hành vi installer, runtime patch, schema database hay release logic nào bị
thay đổi.

P002 chịu trách nhiệm audit toàn bộ 178 commit chỉ có ở upstream và lập ma trận
phân loại thay đổi, kèm bằng chứng source và yêu cầu review iOS-specific. P001
không phân loại trước dải này. Công việc tích hợp về sau thuộc các milestone
P004–P006 và phải giữ các contract bên trên. Mục tiêu vẫn là snapshot
`bb23f99b3eeaced9608e11e280e7c986dba650be` đã đóng băng, trừ khi milestone sau
ghi nhận mục tiêu mới một cách tường minh.
