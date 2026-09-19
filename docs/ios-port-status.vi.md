# Tình trạng bản port iOS

Nhãn trạng thái dùng trong file này: **Working** (đã làm và tin là đúng), **Tested** (đã quan sát trên máy kiểm thử), **Experimental** (có nhưng mới kiểm thử sơ), **Untested** (chưa có bằng chứng), **Planned** (chưa làm).

## Mục đích

Cung cấp binary Beszel Agent + Hub chạy trực tiếp cho thiết bị iOS đã jailbreak, trong khi giữ nguyên chức năng Beszel gốc. Đây là bản port cộng đồng không chính thức, không phải fork mô hình giám sát.

## Quan hệ với bản gốc

- Bản gốc: [henrygd/beszel](https://github.com/henrygd/beszel).
- `main` là nhánh bảo tồn bám sát bản gốc; công việc bảo trì iOS không đẩy
  hay merge vào đó.
- `ios` = base bản gốc + các file/thay đổi iOS-specific (xem bên dưới). Không viết lại Agent/Hub, không đổi DB Hub, không lược tính năng gốc.
- Base iOS ghi theo `beszel.Version`, còn các commit untagged trên upstream-main
  được audit riêng trước khi tích hợp. Xem
  [docs/upstream-sync.vi.md](upstream-sync.vi.md) để biết snapshot hiện tại,
  phân loại và quy trình.

## Mô hình nhánh

- `main` — bám sát bản gốc. Không có thay đổi chỉ cho iOS.
- `ios` — nhánh port iOS đang hoạt động và là nhánh mặc định của repo. Mọi việc iOS nằm ở đây.

## Phần cứng đã kiểm thử

**Tested:**

- iPad mini 2 (iPad4,4 / A1489), Apple A7, arm64
- iOS 12.5.7, đã jailbreak, môi trường Amethyst / Procursus semi-untethered

Sau khi khởi động lại hoàn toàn, phải kích hoạt lại thủ công jailbreak thì
LaunchDaemon tùy chỉnh mới chạy được; sau khi kích hoạt, Agent và Hub đã được
kiểm chứng là tự quay lại. iOS gốc/chưa jailbreak không được hỗ trợ.

Mọi thứ khác đều **Untested** cho tới khi có báo cáo máy thật trong repo.

## Mã iOS-specific (đã xác minh có mặt)

| Đường dẫn | Mục đích | Trạng thái |
| --- | --- | --- |
| `agent/battery/battery_ios.go` (`//go:build ios`) | Pin iOS qua `AppleARMPMUCharger` + `ioreg -l`; đọc Current/Max/RawMax capacity, ExternalConnected, IsCharging, FullyCharged, BatteryInstalled; expose pin `Primary` | **Tested** — dạng đã kiểm thử `Battery:[23 3]`, `Batteries:map[Primary:23]` |
| `agent/battery/battery_darwin.go` (`darwin && !ios`) | Giới hạn đường đi macOS `AppleSmartBattery -a` cho non-iOS để iOS dùng lớp charger thay thế | **Working** |
| `agent/system.go` (guard `runtime.GOOS != "ios"` + hook `adjustPlatformSystemDetails()`) | Tránh probe CPU Darwin của gopsutil trên iOS; áp override metadata iOS | **Working** |
| `agent/system_platform_ios.go` (`//go:build ios`) | Đặt OS/Arch, hostname/kernel/cores/threads qua sysctl, model CPU từ `hw.machine` (map họ A7), `OsName` từ `SystemVersion.plist` | **Tested** |
| `agent/system_platform_other.go` (`//go:build !ios`) | Hook no-op để bản non-iOS không đổi | **Working** |
| `.github/scripts/patch-go-ios-arm64-runtime.py` | Thay `procyieldAsm` dùng `CNTVCT_EL0` bằng vòng `YIELD` kiểu cũ; từ chối vá runtime lạ | **Tested** (bắt buộc trên A7/iOS 12) |
| `.github/workflows/ios-build.yml` | Pipeline hợp nhất: một job macOS build Agent + Hub vào `build/ios/`, kiểm tra metadata Mach-O/iOS-12, sinh + kiểm tra `SHA256SUMS`, tải lên một artifact `beszel-ios-arm64`; job phụ `release-ios` (chỉ push tag) kiểm tra lại payload rồi phát hành đúng ba file đó thành GitHub Release Latest không draft, không prerelease | **Working** |

Đã kiểm tra chọn build-tag: `GOOS=ios go list ./agent/battery` chỉ ra `battery.go + battery_ios.go`; `GOOS=darwin` ra `battery_darwin.go`. `gofmt` sạch, `go test ./agent/battery` đạt trên Linux, `go vet` đạt cho package pin iOS.

## Trạng thái thành phần

### Agent — Tested

- Binary: `/usr/local/bin/beszel-agent`, thư mục dữ liệu `/var/lib/beszel-agent`, cổng `45876`.
- LaunchDaemon: `/Library/LaunchDaemons/dev.beszel.agent.plist`.
- Báo cáo chỉ số hệ thống qua giao thức Beszel Agent chuẩn về Hub.
- Chỉ số CPU/memory/disk/network/load đi qua mã gốc dùng chung (dựa trên gopsutil); chỉ probe model CPU và metadata là iOS-specific.

### Hub — Tested

- Binary: `/usr/local/bin/beszel-hub`, thư mục dữ liệu `/var/lib/beszel-hub`, cổng `8090`.
- LaunchDaemon: `/Library/LaunchDaemons/dev.beszel.hub.plist`.
- Health: `http://127.0.0.1:8090/api/health`.
- **`/var/lib/beszel-hub` không bao giờ được coi là rác** — nó giữ DB / cấu hình người dùng. Nâng cấp và đóng gói phải giữ nó.

### Pin — Tested

- Dùng `/usr/sbin/ioreg -r -c AppleARMPMUCharger -l` (không `-a`; `ioreg` iOS 12 báo lỗi với `-a`).
- Xử lý dòng legacy tiền tố `| "` và bỏ dict `BatteryData` lồng nhau bằng cách khớp tiền tố `"Key" = ` ở top-level.
- Map phần trăm + charging/discharging/full/idle/empty đã kiểm thử trên phần cứng.

### Metadata hệ thống — Tested

- Hostname (`kern.hostname`), kernel (`kern.osrelease`), cores/threads (`hw.physicalcpu`/`hw.logicalcpu`), model CPU từ `hw.machine`, tên OS `iOS <ProductVersion>`.
- `iPad4,*` → `Apple A7`; `iPhone6,1/6,2` → `Apple A7`; còn lại `Apple SoC (<machine>)`.

### Chỉ số mạng / filesystem

- **Working** qua mã Agent gốc dùng chung (không fork iOS cho các đường này). Không thấy hồi quy iOS-specific trên máy kiểm thử, nhưng bao phủ từng interface và từng mount trên iOS ở mức **Experimental** — các ca biên (interface VPN, bố cục mount iOS) coi như chưa kiểm thử.

### Workaround runtime Go Apple A7 — Tested

- `procyieldAsm` hiện đại dùng `CNTVCT_EL0`, không dùng được trong userspace iOS cũ đã thử → `SIGILL` trong `runtime.procyieldAsm`.
- Bản vá Python chạy bằng `sudo` trên runner macOS trước cả hai lượt build iOS. Nó gánh runtime; đừng xóa hay "đơn giản hóa".

### Trạng thái build — Working

- `.github/workflows/ios-build.yml` hợp nhất build cả hai binary vào `build/ios/`, kiểm tra Mach-O arm64 + load command iOS + metadata deployment 12.0, sinh `SHA256SUMS` bằng `shasum -a 256`, kiểm tra lại, rồi tải lên một artifact `beszel-ios-arm64` (`beszel-agent-ios-arm64`, `beszel-hub-ios-arm64`, `SHA256SUMS`).
- Push tag khớp `v*-ios.*` chạy thêm job `release-ios`: tải lại artifact, kiểm tra lại (tồn tại, khác rỗng, `SHA256SUMS` đúng hai mục, khớp checksum), kiểm tra tag với `beszel.Version` và lịch sử `ios`, rồi phát hành ba file bằng `gh release create --latest` (không draft, không prerelease).
- Các workflow probe cũ (`ios-agent-probe.yml`, `ios-hub-probe.yml`) đã gỡ khi pipeline hợp nhất chứng minh được mình; mọi hành vi của chúng đã bao phủ ở trên.

## Hạn chế đã biết

- Chỉ target iPad mini 2 / A7 / iOS 12.5.7 là đã kiểm thử.
- Binary phải cài dưới `/usr/local/bin` với `chown root:wheel`, `chmod 755`, `ldid -S`. Chạy từ `$HOME`/`/tmp` trước đây từng lỗi.
- GitHub Releases phát hành từ tag iOS và `install.sh` lo cài mới, cập nhật giao dịch có sao lưu/quay lại, chẩn đoán, sửa, cấu hình lại và gỡ an toàn với tùy chọn xóa dữ liệu rõ ràng. Vòng đời installer đã cài đặt xong và đã **kiểm thử end-to-end trên máy tham chiếu** (xem [device-validation.vi.md](device-validation.vi.md)), gồm cả bài kiểm thử khởi động lại đầy đủ: sau khởi động lại và kích hoạt lại thủ công jailbreak semi-untethered, cả hai LaunchDaemon tự quay lại, Agent kết nối lại, Hub khỏe mạnh, dữ liệu/lịch sử giữ nguyên đều kiểm tra lại đạt. Không tuyên bố tương thích iOS gốc/chưa jailbreak.
- Plist LaunchDaemon và script đóng gói chưa nằm trong repo (mới chỉ ghi đường dẫn; installer tự sinh plist trên máy).

## Thiết bị / iOS chưa thử

Mọi iPhone/iPad khác, mọi SoC khác (A8+), và mọi phiên bản iOS khác (kể cả iOS hiện đại) đều **Untested**. Đặc biệt, đừng cho rằng bản vá runtime A7 là cần thiết — hay vô hại — trên máy mới nếu chưa thử.

## Việc installer / phát hành đã có và dự kiến

**Đã có:** pipeline build CI hợp nhất cho ra `beszel-agent-ios-arm64`, `beszel-hub-ios-arm64`, `SHA256SUMS` thành một artifact `beszel-ios-arm64`; phát hành GitHub Release theo tag với đúng ba tên asset đó (xem [ios-build-notes.vi.md](ios-build-notes.vi.md) cho scheme `v<upstream>-ios.<rev>`); bộ cài đặt một lệnh tương tác `install.sh` (1.0.0) cho cài mới Agent / Hub / Agent+Hub kèm kiểm tra checksum, ký `ldid`, dựng LaunchDaemon và kiểm tra health Hub, cộng cập nhật giao dịch Agent / Hub / Agent+Hub có theo dõi trạng thái bản (`/var/lib/beszel-ios/install-state`), staging đã ký, sao lưu binary (`/usr/local/bin/*.bak`), tự quay lại, giữ plist, và giữ DB Hub, cộng chẩn đoán chỉ đọc, sửa thận trọng (khởi động lại tại chỗ, khôi phục binary thiếu, tạo lại cấu hình khi được xác nhận) và cấu hình lại an toàn (key/cổng Agent, cổng Hub) với giao dịch plist đã kiểm tra, sao lưu plist (`/Library/LaunchDaemons/*.plist.bak`) và tự quay về cấu hình, cộng gỡ ứng dụng theo giao dịch (Agent, Hub hoặc Agent+Hub; giữ dữ liệu theo mặc định, chạy ngoại tuyến) với tùy chọn xóa dữ liệu cần xác nhận gõ (`DELETE AGENT DATA` / `DELETE HUB DATA`).

**Dự kiến** (chưa làm): plist LaunchDaemon dưới `packaging/launchd/` (hiện do installer sinh thay). Installer resolve tag Latest một lần mỗi lượt chạy rồi ghim mọi lượt tải về tag đó (`.../releases/download/<tag>/...`); URL thô của nó trỏ nhánh `ios` (`.../ios/install.sh`). Xem [architecture.vi.md](architecture.vi.md).
