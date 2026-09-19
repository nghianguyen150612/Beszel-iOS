# Đồng bộ với bản gốc

Đây là hợp đồng bảo trì để giữ bản port iOS gần với Beszel chuẩn,
không làm rơi lặng lẽ bản vá tương thích iOS nào.

## Vai trò các nhánh

- `upstream/main` — mã nguồn và hành vi Beszel chuẩn. Nó là nguồn audit,
  không phải nhánh sản phẩm.
- `main` — nhánh bảo tồn bám sát bản gốc. Công việc bảo trì iOS không
  đẩy hay merge vào đó.
- `ios` — nhánh phát triển/mặc định của bản port cộng đồng: lịch sử bản gốc cộng
  các delta iOS-specific liệt kê bên dưới.
- `origin` — `https://github.com/nghianguyen150612/beszel-ios.git`;
  `upstream` — `https://github.com/henrygd/beszel.git`.

Không bao giờ biến `upstream/main` thành nhánh sản phẩm, và không bao giờ cập nhật `ios`
bằng merge hay rebase không có người xem.

## Baseline audit

Đợt audit bảo trì bắt đầu từ commit đã kiểm thử `8fed846c`, với
`HEAD == origin/ios` và `beszel.Version == "0.19.0"`.

Tại snapshot đã fetch lúc audit:

- Nhánh mặc định bản gốc: `main` (`upstream/HEAD -> upstream/main`).
- `main` bản gốc: `4bf70700` (`feat(hub): add TRUSTED_PROXY_IPS allowlist for TRUSTED_AUTH_HEADER`).
- Tag/release ổn định mới nhất của bản gốc: `v0.19.0`, commit tag
  `ffcdb041`, ngày 2026-09-03.
- Điểm fork của `ios` từ `main` bản gốc: `f204dc17`
  (`feat(agent): Add docker image update available flag (#2211)`).
- `ios` gồm 20 commit port iOS sau điểm fork đó.
- `origin/main` ở `6a7b2772`, sau `upstream/main` untagged đã fetch sáu commit;
  nó vẫn được bảo tồn, không đụng tới.
- Base Beszel iOS vì vậy vẫn là `0.19.0`, và trạng thái lệch tag ổn định
  là **IN SYNC**.
- Tám commit untagged của bản gốc sau `f204dc17` đang chờ xem xét; chúng
  không lặng lẽ nằm trong binary iOS đã kiểm thử.
- Bản binary đã kiểm thử: `v0.19.0-ios.1`.
- Phiên bản installer: `1.0.0`.
- Phần cứng đã kiểm thử: iPad mini 2 / `iPad4,4` / `A1489`, Apple A7, arm64,
  iOS 12.5.7.

Điểm fork quan trọng: chỉ so với tag `v0.19.0` sẽ trộn lẫn các commit sau release
mà bản gốc đã tích hợp với delta iOS thực sự.

## Danh mục bản vá iOS-specific

Danh mục bên dưới rút từ nội dung thật của 20 commit trong
`git log upstream/main..ios` và `git diff f204dc17..ios`, không chỉ từ tiêu đề commit.
Nó trả lời: “Nếu bản gốc đổi X, hành vi iOS nào phải xem lại?”

| # | Vùng | File/hành vi iOS-specific | Thay đổi bản gốc có thể làm mất hiệu lực |
| --- | --- | --- | --- |
| 1 | Workflow build iOS | `.github/workflows/ios-build.yml`: runner macOS, setup Go, vá runtime, build Bun/frontend, clang wrapper, build Agent + Hub, kiểm tra Mach-O/deployment, checksum, artifact, và job phát hành theo tag | Bố cục module/toolchain Go, bố cục build frontend, entrypoint lệnh, hay quy ước trigger workflow |
| 2 | `GOOS=ios` / `GOARCH=arm64` | Cả hai lệnh build đặt `CGO_ENABLED=1`, `GOOS=ios` và `GOARCH=arm64` | Go bỏ hoặc đổi target iOS, hay package build không còn hỗ trợ iOS |
| 3 | Clang wrapper CGO iPhoneOS | Workflow resolve SDK iPhoneOS bằng `xcrun` rồi truyền `-arch arm64`, `-isysroot` và `-mios-version-min=12.0` | Đổi Xcode/SDK/compiler; đổi dependency CGO hay cờ build của bản gốc |
| 4 | Deployment target iOS | Wrapper nhắm iOS 12.0; `otool` kiểm tra trường `minos` của `LC_BUILD_VERSION` hay trường `version` của `LC_VERSION_MIN_IPHONEOS` cũ | Đổi build/linker làm mất hay nâng metadata deployment tối thiểu |
| 5 | Workaround runtime Go Apple A7 | `.github/scripts/patch-go-ios-arm64-runtime.py` thay thân `runtime.procyieldAsm` dùng `CNTVCT_EL0` bằng vòng `YIELD` và từ chối source lạ | `src/runtime/asm_arm64.s` của runtime Go; mọi đổi phiên bản Go đều phải xem lại dạng source |
| 6 | Đo pin iOS | `agent/battery/battery_ios.go` (`//go:build ios`) gọi `/usr/sbin/ioreg -r -c AppleARMPMUCharger -l`, xử lý output cũ, map pin hệ thống thành `Primary`, và đọc capacity/sạc | Đổi API package pin: `Battery`, hằng trạng thái, normalize hay `Primary`; đổi tách file/build-tag pin của bản gốc |
| 7 | Loại trừ pin Darwin | `agent/battery/battery_darwin.go` gắn `darwin && !ios`, nên iOS không chọn bản `AppleSmartBattery -a` của macOS | Bản gốc đổi build tag pin Darwin hay chọn file theo platform |
| 8 | Xử lý đĩa Darwin / fallback disk-I/O | Không fork đĩa iOS-specific: `agent/disk.go`, mã storage-pool và counter đĩa gopsutil vẫn là mã gốc dùng chung. Bản port không tuyên bố fallback Darwin-specific nào | API đĩa gopsutil, `agent/storage_pool.go` / `agent/zfs/*` dùng chung, hay build tag platform; phải test lại thay vì cho rằng hành vi mount/I-O iOS giống vậy |
| 9 | Xử lý mạng iOS | Không fork mạng iOS-specific; dùng nguyên `agent/network.go` và collector mạng bản gốc | API mạng gopsutil hay giả định interface/địa chỉ; hành vi VPN và từng interface chưa kiểm thử rộng |
| 10 | Agent support / metadata hệ thống | `agent/system.go` bỏ probe CPU Darwin trên iOS và gọi `adjustPlatformSystemDetails()`; `agent/system_platform_ios.go` cấp metadata sysctl/plist; `agent/system_platform_other.go` là no-op cho non-iOS | `refreshSystemDetails`, `system.Details`, `system.Darwin`, hay API CPU/platform của gopsutil |
| 11 | Hub support | Không fork mã Hub; `.github/workflows/ios-build.yml` build `./internal/cmd/hub` cho iOS bằng mã Hub chuẩn và migration | `internal/cmd/hub`, API PocketBase, khởi tạo Hub, migration, hay dependency ngừng biên dịch cho iOS |
| 12 | Yêu cầu build/embed frontend | `internal/site/embed.go` nhúng `all:dist`; `internal/site/dist` bị ignore; workflow chạy `bun install --frozen-lockfile` (Bun 1.4.0 đã ghim) và `bun run build` trước cả hai lượt build Go | Đường dẫn embed, package/build output frontend, Bun/Vite, hay API/type frontend sinh ra |
| 13 | Workflow phát hành GitHub | Workflow iOS tải lên payload ba file và chỉ job tag của nó mới phát hành; nó kiểm tra phiên bản nguồn, lịch sử iOS, checksum, số asset, và trạng thái Latest/không draft/không prerelease | Khai báo phiên bản `beszel.go`, bố cục artifact, hành vi action GitHub, hay quyền release |
| 14 | Định dạng tag phát hành iOS | `v<upstream>-ios.<positive-integer>`, hiện là `v0.19.0-ios.1`; `install.sh` và job phát hành iOS cùng ép | Khai báo phiên bản bản gốc hay đổi contract tag/asset phát hành |
| 15 | `install.sh` | Installer 1.0.0: cài mới Agent/Hub/Both, tải ghim checksum, `ldid`, LaunchDaemon, kiểm tra health, và các action vòng đời ngoại tuyến | Tên asset phát hành, dạng URL Latest-release, hay chính sách version/tag bản gốc |
| 16 | Test installer | `tests/install-sh-test.sh`: 551 test baseline bao phủ parsing, xử lý checksum/state, giao dịch, quay lại, chẩn đoán, sửa, cấu hình lại, gỡ, giữ dữ liệu và đường lỗi | Mọi contract hàm installer hay tên asset/state |
| 17 | Xử lý LaunchDaemon | Plist do `install.sh` sinh trên máy thành `dev.beszel.agent` và `dev.beszel.hub`; không có plist nào trong repo là chuẩn | Đổi đường dẫn/nhãn installer hay hành vi launchd iOS |
| 18 | Ký `ldid` | Installer ký binary staging bằng `ldid -S`, cài dưới `/usr/local/bin`, và ép `root:wheel` / mode 755 | Đổi công cụ ký hay môi trường jailbreak trên máy; không phụ thuộc mã nguồn bản gốc |
| 19 | Theo dõi trạng thái | `/var/lib/beszel-ios/install-state` của installer ghi tag từng thành phần và SHA-256 asset gốc, không lưu key | Định dạng state installer hay logic cập nhật giao dịch |
| 20 | Cập nhật / quay lại | Staging đã ký, sao lưu binary `/usr/local/bin/*.bak`, tự quay lại, giữ plist, và không sửa thư mục dữ liệu Hub | Asset phát hành, hành vi khởi động/health, hay giả định giao dịch installer |
| 21 | Sửa / cấu hình lại / gỡ | Chẩn đoán chỉ đọc; khôi phục/khởi động lại thận trọng; giao dịch key/cổng Agent và cổng Hub đã kiểm tra; gỡ ứng dụng giữ dữ liệu theo mặc định; xóa cần xác nhận gõ chính xác | Đường dẫn installer, ngữ nghĩa launchd, tình trạng dịch vụ, hay kỳ vọng giữ dữ liệu |
| 22 | Tài liệu máy thật | `docs/device-validation.vi.md`, `docs/ios-port-status.vi.md`, `docs/ios-build-notes.vi.md` và `docs/architecture.vi.md` ghi máy tham chiếu, hạn chế và contract vận hành | Đổi tuyên bố kiểm thử, chỉ số hỗ trợ, bố cục máy, hay điều kiện build |
| 23 | Cô lập phát hành iOS | `release.yml` và `docker-images.yml` loại `v*-ios.*`, ngăn automation GoReleaser/Docker đụng tag iOS thuần | Bản gốc sửa trigger tag hay sở hữu workflow phát hành |

Các dòng ghi rõ “không fork iOS-specific” là delta âm có chủ ý:
hành vi gốc dùng chung vẫn cần xem lại chức năng sau đổi bản gốc,
nhưng không có bản vá iOS nào phải đắp lại.

## Các mặt thay đổi bản gốc

Bảng sau bao phủ các mặt bản gốc được yêu cầu, đối chiếu dải pending
`f204dc17..4bf70700`. Mỗi mặt có một phân loại và một lý do cụ thể.
Tích hợp có sinh binary vẫn cần đủ checklist thiết bị thật bên dưới.

| Mặt | Bằng chứng trong dải pending | Phân loại | Lý do kỹ thuật |
| --- | --- | --- | --- |
| `internal/cmd/agent` | Không đụng | UNAFFECTED | Entrypoint Agent, cờ CLI, subcommand health và đường output không đổi trong dải này. |
| `internal/cmd/hub` / entrypoint Hub | Không đụng | UNAFFECTED | Khởi động Hub, cờ CLI, `/api/health` và đăng ký migration không đổi. |
| `internal/agent` và collector hệ thống | `agent/smart.go` / test và mã storage-pool dùng chung có đụng | AUTO-MERGE LIKELY | Không có fork iOS trong logic SMART hay collector dùng chung; vẫn cần xem lại build và tương đương chức năng. |
| Collector đĩa / disk-I/O | `agent/storage_pool.go`, `agent/zfs/*` có đụng qua `98210174` | IOS PATCH REVIEW REQUIRED | File `zfs_nonlinux.go` mới gắn tag `!linux`, `GOOS=ios` sẽ chọn; phải kiểm tra nó sống chung với build iOS và không đổi hành vi ZFS-vắng mặt. |
| Collector pin | Không đụng | UNAFFECTED | `battery_ios.go` và loại trừ Darwin nằm ngoài dải pending. |
| Collector mạng | Không đụng | UNAFFECTED | Không đổi collector mạng hay xử lý interface. |
| Frontend nhúng | Bảy file frontend trong `f0f1f798`; hai trong `6a7b2772`; một trong `086091a0` | AUTO-MERGE LIKELY | Hub iOS nhúng cùng frontend sinh ra; phải kiểm tra lại output build và hành vi dashboard. |
| Output/toolchain build frontend | Không đổi đường embed hay package-manager | AUTO-MERGE LIKELY | Điều kiện `bun install --frozen-lockfile` (Bun 1.4.0 đã ghim) + `bun run build` vẫn là chuẩn, nhưng output sinh ra phải build lại. |
| Phiên bản Go / `go.mod` / `go.sum` | Không đụng | UNAFFECTED | Dải này giữ Go 1.27.1 và đồ thị dependency không đổi. |
| Build tag / giả định platform | Thêm `zfs_nonlinux.go`; không đụng runtime hay file platform iOS | IOS PATCH REVIEW REQUIRED | `!linux` bao gồm iOS, nên `GOOS=ios go list/build` là bước kiểm tra bắt buộc dù không xung đột chữ. |
| Runtime Go / giả định A7 | Không đụng | UNAFFECTED | Bản vá runtime đắp vào toolchain Go đã cài, độc lập với các commit bản gốc này; sentinel dạng source vẫn bắt buộc. |
| Tên asset phát hành | Không đụng | UNAFFECTED | `beszel-agent-ios-arm64`, `beszel-hub-ios-arm64` và `SHA256SUMS` vẫn là contract của port. |
| Migration DB | Không đụng | UNAFFECTED | Không đổi file migration hay snapshot collection trong dải. |
| Định dạng cấu hình | Không đổi schema cấu hình | UNAFFECTED | Cấu hình Hub/Agent và contract plist installer không đổi. |
| Giao thức Agent↔Hub | Không đổi `internal/common`, action WebSocket hay version tối thiểu | UNAFFECTED | Ngưỡng tương thích CBOR/WebSocket và ngữ nghĩa payload không đổi. |
| Cổng / tham số CLI | Không đụng | UNAFFECTED | Mặc định Agent 45876, Hub 8090, cờ và lệnh health vẫn vậy. |
| Endpoint health | Không đụng | UNAFFECTED | Hành vi `/api/health` không sửa. |
| Xử lý version/phát hành | Không đổi `beszel.go` hay workflow phát hành | UNAFFECTED | Code base vẫn `0.19.0`; đợt audit này không tạo tag hay revision phát hành. |
| Xác thực/users Hub | `internal/hub/api.go`, `internal/users/users.go` và test có đụng | AUTO-MERGE LIKELY | Đổi hành vi Hub bản gốc, không xung đột mã iOS; cần test hồi quy login/tài khoản và trusted-header. |

### Phân loại commit pending

Các commit untagged đã fetch của bản gốc, từ cũ tới mới:

| Commit | Thay đổi | Phân loại | Lý do |
| --- | --- | --- | --- |
| `086091a0` | Bỏ lịch sử pending khi chuyển sang live chart | AUTO-MERGE LIKELY | Chỉ hành vi frontend; UI nhúng dùng nó, không overlap vá iOS. |
| `6a7b2772` | Theo theme hệ thống trực tiếp | AUTO-MERGE LIKELY | Chỉ hành vi frontend; build lại UI nhúng và kiểm tra render login/dashboard. |
| `98210174` | Bỏ gọi tiện ích ZFS khi thiếu `/dev/zfs` | IOS PATCH REVIEW REQUIRED | Thêm file `!linux` mà iOS sẽ chọn, đổi hành vi storage-pool dùng chung; chạy kiểm tra package/build iOS. |
| `a0bf3387` | Đặt giới hạn batch request/body cho Hub | AUTO-MERGE LIKELY | Chỉ default Hub; không xung đột mã iOS hay đổi giao thức, nhưng nên thử khởi động Hub và hành vi alert. |
| `50f6fc07` | Bootstrap first-user atomic và test | AUTO-MERGE LIKELY | Chỉ hành vi Hub/users; không xung đột migration hay mã iOS, nhưng kiểm tra lại login/bootstrap. |
| `f0f1f798` | Lưu view preference và ngôn ngữ | AUTO-MERGE LIKELY | Frontend nhúng cộng hành vi user settings; build lại và thử lưu dashboard/settings. |
| `18f7a4bb` | Revert cảnh báo SMART cho attribute 5/197/198 | AUTO-MERGE LIKELY | Xóa parser SMART dùng chung, không fork iOS; hành vi về lại chuẩn bản gốc. |
| `4bf70700` | Hạn chế trusted auth header theo proxy IP/CIDR | AUTO-MERGE LIKELY | Thêm setting xác thực Hub tùy chọn, mặc định không đổi; test lại auth và truy cập trực tiếp/proxy. |

Không có commit pending nào ở mức `IOS PATCH CONFLICT`. Hai phân loại `IOS PATCH REVIEW
REQUIRED` là review build-tag/platform, không phải tuyên bố sẽ xung đột chữ khi merge.
Mọi bản phát hành chứa đổi binary hay frontend này đều thuộc diện `REAL-DEVICE RETEST REQUIRED`.

## Quy trình đồng bộ bản gốc tái lập được

Chạy quy trình này trên `ios` với cây sạch. Nó cố ý review trước,
không cập nhật `main`.

1. Fetch riêng và xem ref:

   ```sh
   git fetch origin
   git fetch upstream --tags
   git branch --show-current
   git status --short
   git rev-parse HEAD
   git rev-parse origin/ios
   git symbolic-ref --short refs/remotes/upstream/HEAD
   ```

2. Xác định tag ổn định bản gốc mục tiêu. Ưu tiên tag `vX.Y.Z` mới nhất
   hơn tip `main` untagged. Ghi lại `beszel.Version` cũ, commit iOS hiện tại,
   điểm fork, tag/commit mục tiêu và ngày tag.

3. Đọc release notes/changelog bản gốc và xem toàn dải:

   ```sh
   git log --stat <old-upstream-ref>..<target-ref>
   git diff --name-status <old-upstream-ref>..<target-ref>
   ```

   Phân loại mọi mặt bị đụng thành `UNAFFECTED`, `AUTO-MERGE LIKELY`,
   `IOS PATCH REVIEW REQUIRED`, `IOS PATCH CONFLICT` hoặc
   `REAL-DEVICE RETEST REQUIRED`, kèm lý do kỹ thuật.

4. Chỉ tích hợp sau khi review xong. Trên `ios`, dùng merge thường có người xem,
   giữ lịch sử, ví dụ:

   ```sh
   git merge --no-ff --no-commit <target-ref>
   ```

   Resolve conflict có chủ ý, giữ hành vi iOS, xem kết quả staged rồi commit.
   Không rebase, không viết lại lịch sử, không force-push, không
   merge bản gốc vào `main`. Nếu mục tiêu chỉ là tip phát triển untagged,
   đừng thể hiện nó như bản ổn định.

5. Chạy lại danh mục vá và xem mọi file nhạy iOS đã đổi.
   Bản vá runtime phải khớp dạng source runtime Go đã cài; không bao giờ
   nới lỏng hay bypass sentinel lỗi.

6. Chạy kiểm tra theo thứ tự:

   ```sh
   sh tests/ios-patch-sentinels.sh
   sh tests/upstream-sync-test.sh
   git diff --check
   sh -n install.sh
   sh -n tests/install-sh-test.sh
   shellcheck -s sh install.sh
   shellcheck -s sh tests/install-sh-test.sh
   sh tests/install-sh-test.sh
   ```

7. Trên runner macOS, build cả hai output `GOOS=ios GOARCH=arm64` bằng build frontend
   tương đương workflow, bản vá runtime A7, clang wrapper iPhoneOS và đúng tên output.
   Kiểm tra Mach-O, arm64, load command iOS 12.0, checksum và artifact ba file.

8. Đối chiếu mặt chức năng với checklist tương đương. Chỉ khi mọi kiểm tra tự động
   và build đều đạt mới xét release candidate.
   Bắt buộc kiểm thử máy thật trên phần cứng tham chiếu trước khi
   phát hành hay gắn Latest cho bản iOS mới.

## Kiểm tra lệch

`.github/scripts/check-upstream-drift.sh` là bước kiểm tra thông tin chỉ đọc.
Nó đọc gán `Version` trong `beszel.go`, hỏi tag bản gốc bằng
`git ls-remote --tags`, chỉ xét tag ổn định đúng dạng `vX.Y.Z` (annotated hay
lightweight), rồi in cả base iOS lẫn bản ổn định mới nhất của bản gốc.
Nó không fetch vào checkout, không sửa mã, không merge, không bump version, không tạo
tag, không tạo issue, không phát hành.

Exit code:

- `0` — version ổn định khớp nhau (`IN SYNC`).
- `1` — lệch version ổn định, hoặc base iOS lạ vượt trước bản gốc.
- `2` — mã lỗi, thiếu tag ổn định, hay lỗi remote/truy vấn.

Phép so cố ý chỉ báo lệch bản ổn định, không báo từng commit untagged
trên `upstream/main`. Đổi main untagged vẫn cần audit tay bên trên.
`tests/upstream-sync-test.sh` dùng remote fixture local để test đọc tag,
báo version và mọi trạng thái mà không cần mạng.

`.github/workflows/upstream-drift.yml` chạy kiểm tra hàng tuần và khi dispatch tay
với `contents: read`, không secret, không quyền ghi, không spam issue,
không action phát hành.

## Sentinel hồi quy vá iOS

`tests/ios-patch-sentinels.sh` kiểm tra tĩnh, fail to, cho:

- Có mặt bản vá runtime A7, chặn đúng dạng source, nhận `CNTVCT_EL0` /
  `CNTFRQ_EL0`, và thứ tự vá-runtime-trước-build.
- `GOOS=ios`, `GOARCH=arm64`, bao phủ đủ đường dẫn mã Go trong workflow,
  `-mios-version-min=12.0`, kiểm tra Mach-O/arm64 và load-command iOS.
- Tên output cả Agent/Hub, `SHA256SUMS` và hai entrypoint build.
- Source/build-tag pin iOS, `AppleARMPMUCharger`, `ioreg` và loại trừ
  `darwin && !ios`.
- Cú pháp tag installer/phát hành và loại tag iOS khỏi workflow
  GoReleaser/Docker bản gốc.
- Contract `//go:embed all:dist`, `dist` bị ignore, và thứ tự frontend-trước-Go.
  Không chấp nhận thư mục frontend giả rỗng.
- Hook metadata hệ thống iOS và companion build-tag no-op cho non-iOS.

Các kiểm tra sentinel là chốt mã/workflow, không thay cho
cross-build macOS hay test máy thật. Bản thân script vá runtime cũng
được test trên bản copy cô lập của runtime Go hiện tại và một dạng cố ý
không quen.

## Phát hiện về tái lập build

Workflow checkout sạch dựng rõ các điều kiện cần:

1. Workflow iOS chạy `tests/ios-patch-sentinels.sh` và
   `tests/upstream-sync-test.sh` quyết định trước khi cài toolchain hay biên dịch.
2. `actions/setup-go` đọc đúng phiên bản Go từ `go.mod` (`1.27.1` tại
   snapshot audit này).
3. `oven-sh/setup-bun` cài Bun 1.4.0 đã ghim.
4. `bun install --frozen-lockfile` và `bun run build` tạo `internal/site/dist`
   production bị ignore trước khi biên dịch Hub.
5. Bản vá runtime chạy trước cả hai lượt build Go và từ chối
   runtime lạ.
6. `xcrun` resolve SDK iPhoneOS và clang hiện tại; wrapper mang cờ
   arm64 và iOS 12.0.
7. Agent và Hub build độc lập ra hai tên contract.
8. `file` và `otool` kiểm tra Mach-O, arm64 và load command deployment tối thiểu.
9. `shasum -a 256` tạo và kiểm tra `SHA256SUMS`.
10. Chỉ tải lên ba file payload phát hành; job phát hành kiểm tra lại
    đúng ba asset và hai mục checksum.

Đường dẫn push của workflow nay gồm file Go gốc và lồng nhau, nên đổi dưới
`internal/hub`, `internal/migrations`, `internal/entities` và các package Go khác
không thể lặng lẽ bỏ qua build iOS. Đổi chỉ tài liệu vẫn không kích hoạt build nhánh;
push tag iOS luôn chạy đủ pipeline.

`internal/site/dist` bị ignore và vắng mặt sau checkout sạch. Build Hub trực tiếp
mà không build frontend thật vì vậy không được hỗ trợ, không được “chữa” bằng thư mục
rỗng: làm vậy sẽ ra Hub thiếu dashboard production. Môi trường Linux local không tái lập
được build CGO/Mach-O iPhoneOS; workflow macOS vẫn là chuẩn cho
bước đó.

Bước dependency frontend tái lập được từ checkout sạch: `internal/site/bun.lock`
đã check-in (lockfileVersion 3, sinh bằng Bun 1.4.0) phản ánh mục `overrides`
trong `package.json`
(`@nanostores/router` → `nanostores ^0.11.3`, thừa hưởng từ bản gốc từ 0.19.0),
nên `bun install --frozen-lockfile` xong mà không sửa lockfile, rồi build production thật
(`bun run build`: Lingui extract/compile + Vite). `oven-sh/setup-bun@v2` ghim
Bun 1.4.0 trong `.github/workflows/ios-build.yml`. Đừng giấu lỗi frontend
bằng thư mục `dist` rỗng; workflow macOS vẫn là chuẩn cho
binary iPhoneOS thật.

## Chính sách đánh phiên bản phát hành

Version mã và version installer độc lập:

- Base bản gốc `0.19.0` cộng revision binary iOS đầu:
  `v0.19.0-ios.1`.
- Base bản gốc mới `0.20.0` sẽ reset revision binary:
  `v0.20.0-ios.1`.
- Đổi chỉ installer, chỉ tài liệu hay chỉ CI không sinh
  `ios.2`.
- Đổi mã Go ảnh hưởng binary Agent hay Hub trên cùng base bản gốc
  thì tăng revision: `v0.19.0-ios.1` → `v0.19.0-ios.2`.
- Release notes phải nêu đúng base bản gốc. Không bao giờ tuyên bố version mới hơn
  mã đã tích hợp vào bản phát hành.
- `INSTALLER_VERSION` giữ `1.0.0`; đợt audit bảo trì này không bump nó.

Đợt audit bảo trì này không tạo tag phát hành, revision binary hay version installer nào.

## Bảo trì ngôn ngữ tài liệu

Tài liệu tiếng Anh (`README.md`, `docs/*.md`) là bản chuẩn. Các bản tiếng Việt
(`README.vi.md`, `docs/*.vi.md`) là bản dịch phụ. Hãy cập nhật bản tiếng Anh
chuẩn trước, rồi đồng bộ bản dịch tiếng Việt để hai ngôn ngữ giữ cùng sự kiện.

## Checklist tương đương tương lai

Áp checklist này cho mọi lần bump bản gốc sau này. Ghi metric không hỗ trợ
là không hỗ trợ; đừng biến việc thiếu metric platform thành fail.

### Agent

- [ ] Chạy dưới môi trường launchd iOS tham chiếu.
- [ ] Kết nối Hub và kết nối lại sau khi Hub khởi động lại.
- [ ] Dữ liệu CPU và số core CPU ở nơi hỗ trợ.
- [ ] Dữ liệu bộ nhớ và swap ở nơi hỗ trợ.
- [ ] Dùng filesystem và xử lý filesystem root/tùy chỉnh.
- [ ] Disk I/O và map thiết bị đĩa; test rõ ca biên mount iOS.
- [ ] Throughput mạng và các interface liên quan.
- [ ] Dữ liệu uptime và load ở nơi hỗ trợ.
- [ ] Phần trăm/trạng thái pin qua `Primary` trên máy tham chiếu.
- [ ] Nhiệt độ/quạt/GPU/SMART/ZFS chỉ nơi máy và môi trường
  thực sự expose; còn lại ghi không hỗ trợ.
- [ ] Hành vi process/container/system theo đúng máy.

### Hub

- [ ] Khởi động và giữ khỏe.
- [ ] `/api/health` trả thành công.
- [ ] Đăng nhập, bootstrap/tài khoản đầu và cài đặt tài khoản chạy.
- [ ] Danh sách systems và xác thực Agent chạy.
- [ ] DB lịch sử mở không mất dữ liệu.
- [ ] Thống kê mới của bản gốc hiện khi được hỗ trợ.
- [ ] Migration áp và giữ dữ liệu cũ.
- [ ] Frontend/dashboard nhúng tải được sau build frontend thật.
- [ ] Agent kết nối lại và tiếp tục lịch sử sau khi Hub khởi động lại.

### Installer

- [ ] Cài mới Agent.
- [ ] Cài mới Hub.
- [ ] Cài mới Agent + Hub.
- [ ] Cập nhật thành phần và hành vi update no-op.
- [ ] Quay lại khi update lỗi.
- [ ] Chẩn đoán chỉ đọc.
- [ ] Sửa và khôi phục thành phần thiếu.
- [ ] Cấu hình lại Agent/Hub có quay về cấu hình.
- [ ] Gỡ giữ dữ liệu theo mặc định.
- [ ] Cài lại giữ dữ liệu.
- [ ] Theo dõi trạng thái và hành vi checksum.

### Bền sau khởi động

- [ ] Nhãn, đường dẫn, sở hữu và quyền LaunchDaemon vẫn đúng.
- [ ] Sau khi khởi động lại hoàn toàn, kích hoạt lại thủ công jailbreak semi-untethered;
  rồi kiểm tra cả hai LaunchDaemon tự quay lại.
- [ ] Kiểm tra lại Agent kết nối lại, Hub khỏe và dữ liệu/lịch sử giữ nguyên sau
  khi kích hoạt lại.

Tham chiếu kiểm thử cho các mục này vẫn là iPad mini 2 / `iPad4,4` /
`A1489` / Apple A7 / iOS 12.5.7. Máy và iOS khác vẫn
chưa thử.

## Quyết định giai đoạn tiếp theo hiện tại

Không có bản ổn định bản gốc nào mới hơn `v0.19.0`, nên không cần
tích hợp base ổn định và không tạo bản phát hành mới.

Còn một pha tích hợp tương lai tùy chọn cho dải main untagged của bản gốc:

- Mục tiêu đề xuất: `upstream/main` tại `4bf70700`, dải
  `f204dc17..4bf70700`, gồm tám commit đã phân loại trên.
- Mục tiêu phát hành nên dùng: chờ tag ổn định tiếp theo của bản gốc rồi audit
  tag đó; đừng gắn nhãn tip main untagged thành bản Beszel ổn định.
- Xung đột chữ dự kiến: không có trong file iOS hiện tại. Xem lại
  chọn build `agent/storage_pool.go` / `agent/zfs/*` và xem ngữ cảnh
  `agent/system.go` / `agent/battery/*` sau tích hợp.
- Hệ quả Go/runtime: dải pending không đổi `go.mod`; vẫn chạy lại
  kiểm tra dạng source A7 với runtime Go đã cài.
- Migration DB: dải pending không có.
- Giao thức Agent↔Hub, cổng/CLI và endpoint health: không thấy đổi.
- Nếu tích hợp và phát hành dải này trong khi base vẫn `0.19.0`,
  revision binary sẽ là `v0.19.0-ios.2` sau khi đủ kiểm tra. Hiện không
  tạo revision nào.
- Trước khi phát hành, lặp lại đủ checklist tự động/build/tương đương và
  kiểm thử máy thật, gồm khởi động lại và kích hoạt lại jailbreak.
