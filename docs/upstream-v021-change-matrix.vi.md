# Audit ma trận thay đổi upstream v0.21 (P002)

## Phạm vi và kết quả

Audit này bao phủ dải upstream đã ghim từ điểm fork iOS đến target v0.21 đã
đóng băng. Danh mục chuẩn, mỗi commit một dòng, nằm trong
[`upstream-v021-change-matrix.csv`](upstream-v021-change-matrix.csv); danh mục
tương thích iOS lịch sử nằm trong
[`upstream-v021-ios-patch-map.csv`](upstream-v021-ios-patch-map.csv). Hai CSV
này là nguồn dữ liệu chuẩn; báo cáo tóm tắt kết quả và không lặp lại 178 dòng.

**Độ bao phủ audit: 178/178 commit upstream đã xem xét; 35/35 commit iOS lịch
sử đã được ánh xạ; 0 dòng ma trận chưa giải quyết.** Không merge mã nguồn
upstream và không sửa mã Agent hoặc Hub trong production.

## Tham chiếu Git đã đóng băng và phương pháp

| Tham chiếu | SHA / kết quả |
| --- | --- |
| `origin/iOS` ban đầu / checkout | `14abe465c6eb23b484b8beaa2f0d62c67ac6fe20` |
| Target `origin/main` và `upstream/main` đã đóng băng | `bb23f99b3eeaced9608e11e280e7c986dba650be` |
| Merge-base / điểm fork upstream | `f204dc17e64bba444478c60210d018e43a777193` |
| Snapshot iOS lịch sử P001 | `a6b73365147fc8c3a820552a145d370418e5b680` |
| Target đã peel của `v0.20.0` | `a0dc19eacf6902d2f6934a3efd7ea7277198c40b` |
| Target đã peel của `v0.21.0` | `ec96c134660c998f7eb044e349e3715b65733fe1` |

Cả hai tag phát hành đều là ancestor của target upstream đã đóng băng. Dải so
sánh chính xác là
`f204dc17e64bba444478c60210d018e43a777193..bb23f99b3eeaced9608e11e280e7c986dba650be`.
`git rev-list --reverse --topo-order` trả về 178 commit. Metadata, số parent,
patch first-parent và các đường dẫn đã đổi của từng commit đều được xem xét;
phát hiện theo commit nằm trong CSV. Dải này có 178 commit thường và không có
merge commit thực sự. Commit có tiêu đề “Merge commit from fork” là
`50f6fc075d7af0482287acb5c61fbc8a88578335`, nhưng chỉ có một parent nên được
phân loại đúng là commit thường. Không bỏ sót diff giải quyết merge nào.

So sánh P001 `f204dc17..a6b73365` có 35 commit iOS lịch sử. `origin/iOS` hiện
cách điểm fork 37 commit vì còn có hai commit tài liệu/PR P001; hai commit này
không được tính là bản vá tương thích lịch sử. Phân batch dựa trên ancestry Git,
không dựa ngày hay tiêu đề. Ba batch rời nhau và hợp của chúng đúng bằng dải
178 commit.

| Batch tích hợp | Dải ancestry | Số commit |
| --- | --- | ---: |
| P004 | Từ điểm fork đến hết `v0.20.0` | 25 |
| P005 | Sau `v0.20.0` đến hết `v0.21.0` | 116 |
| P006 | Sau `v0.21.0` đến `bb23f99b3` | 37 |
| **Tổng** | **Dải đã đóng băng** | **178** |

CSV dùng dấu chấm phẩy cho các giá trị phân loại nhiều mục, chứa mọi đường dẫn
đã đổi ở first-parent làm bằng chứng và đánh dấu đủ 178 dòng `Reviewed`.
Validator thư viện chuẩn dựng lại dải từ SHA đóng băng, kiểm tra ancestry của
batch và tập SHA chính xác, đồng thời kiểm tra danh mục 35 bản vá iOS và các
tham chiếu ID bản vá.

## Số liệu phân loại

Số theo độ ưu tiên và loại thay đổi là phân loại đơn, mỗi nhóm cộng lại bằng
178.

| Độ ưu tiên | Commit |
| --- | ---: |
| P0 | 21 |
| P1 | 74 |
| P2 | 83 |
| none | 0 |

| Loại thay đổi | Commit |
| --- | ---: |
| fix | 102 |
| Feature | 28 |
| refactor | 13 |
| security | 6 |
| dependency | 6 |
| test | 8 |
| documentation | 10 |
| CI | 2 |
| other | 3 |

| Khả năng áp dụng với iOS | Commit | Nghĩa dùng trong audit này |
| --- | ---: | --- |
| Direct | 13 | Hành vi build/platform iOS hoặc file/contract do iOS sở hữu bị thay đổi |
| shared | 40 | Hành vi Agent, giao thức hoặc dữ liệu dùng chung bởi iOS và Agent từ xa |
| Hub-only | 96 | Đường Hub/API/database/frontend, gồm telemetry từ Agent từ xa |
| platform-other | 24 | Runtime/tooling đặc thù platform ngoài target Agent iOS |
| none | 5 | Không có hành vi runtime hay Hub trực tiếp, ví dụ test/tài liệu hỗ trợ |

Số subsystem bên dưới có thể chồng lấp: một commit có thể ảnh hưởng nhiều
vùng. Số tổng quát được theo sau bởi một số tag subsystem cụ thể trong trường
`areas` chuẩn.

| Vùng tổng quát | Commit |
| --- | ---: |
| Agent | 63 |
| Protocol | 32 |
| Hub | 43 |
| Database | 12 |
| Frontend | 79 |
| Platform | 8 |
| CI | 4 |
| Tests | 97 |
| Documentation | 20 |
| Dependencies | 12 |
| Installer | 0 |

| Subsystem | Commit | Subsystem | Commit |
| --- | ---: | --- | ---: |
| Agent: CPU/RAM | 5 | Agent: Cgroups | 2 |
| Agent: Disk I/O | 14 | Agent: Network | 7 |
| Agent: Wi-Fi RSSI | 7 | Agent: Package updates | 5 |
| Agent: NVMe SMART | 7 | Agent: ZFS | 6 |
| Agent: GPU | 7 | Agent: Systemd | 6 |
| Agent: Containers | 6 | Agent: Polling/errors | 13 |
| Protocol: WebSocket | 18 | Protocol: SSH | 8 |
| Protocol: TLS | 2 | Protocol: CBOR/JSON | 22 |
| Protocol: Headers/auth | 2 | Hub: Authorization | 8 |
| Hub: API | 12 | Hub: Sessions/users | 3 |
| Hub: Polling | 27 | Hub: Alerts | 10 |
| Hub: Monitor records | 15 | Database: Migrations | 7 |
| Database: History | 4 | Database: Records | 20 |
| Frontend: Dashboard | 18 | Frontend: Network monitors | 21 |
| Frontend: Charts | 18 | Frontend: Alerts UI | 5 |
| Frontend: Package updates UI | 6 | Frontend: Wi-Fi charts | 6 |
| Frontend: Logs | 3 | Frontend: Settings | 4 |
| Frontend: Localization | 13 | Frontend: Dependencies | 6 |
| Platform: Linux | 16 | Platform: Windows | 4 |
| Platform: macOS | 2 | Platform: Docker/release | 6 |
| Dependencies: Go | 6 | Dependencies: Bun/npm | 6 |

Trường `areas` cũng ghi tag platform/iOS, embedded assets, history và các tag
cụ thể khác khi có. Số tổng quát và số chi tiết không cộng dồn với nhau.

## Rà soát P0

Các thay đổi này liên quan xác thực/ủy quyền, cô lập dữ liệu giữa hệ thống,
trạng thái migration, serialization hoặc vòng đời kết nối. Bảng ghi invariant
cần giữ khi tích hợp sau này; đường dẫn lấy từ diff đã xem xét.

| Commit upstream | Invariant đã rà soát và bằng chứng |
| --- | --- |
| `50f6fc075d7af0482287acb5c61fbc8a88578335` | Tạo user đầu tiên kiểm tra trạng thái bootstrap một lần và tạo user/superuser trong cùng transaction; bootstrap đồng thời hoặc lỗi giữa chừng không để lại tài khoản nửa hoàn tất. `internal/users/users.go` |
| `4bf70700f20cc2994774995dca5e725c7173996b` | `TRUSTED_PROXY_IPS` giới hạn `TRUSTED_AUTH_HEADER` theo IP/CIDR. Giá trị unset/rỗng vẫn không giới hạn; allowlist đã cấu hình nhưng toàn giá trị sai thì không tin peer nào. `internal/hub/api.go` |
| `f7fd3ef403085b64dc8a21a02011105b2251333a` | User mới tạo qua OAuth bắt đầu ở role `user`; user hiện có giữ role. `internal/users/users.go`, `internal/hub/hub.go` |
| `b0bc72741126b2eadaf284e803827261eac82d54` | Snapshot collection v0.20 hấp thụ field `updatable` của container và xóa hai migration đã gộp vào snapshot. Kiểm tra thứ tự nâng cấp và giữ field hiện có. `internal/migrations/0_collections_snapshot_0_20_0.go` |
| `187dc886a96f7124090f7e5a258c9c720bd2cb2c` | Sau SSH reconnect, Hub đồng bộ lại cấu hình monitor và thử lại sync lỗi mà không làm hỏng lần lấy stats thường. `internal/hub/systems/network_monitors.go`, `system.go` |
| `8bf6917fe07ce0a33c9ae29a70bcea45410b0a7e` | Collection pool chậm được cache/refresh có mutex; xử lý deadline dữ liệu của Hub không biến phản hồi chậm thành vòng reconnect. `agent/storage_pool.go`, `internal/hub/systems/system.go` |
| `24ec18a937b1b18d606eef2f56728230326557e6` | Mỗi phản hồi Agent được decode vào struct mới, tránh giữ field CBOR cũ bị lược và tránh realtime/regular fetch sửa cùng dữ liệu. `internal/hub/systems/system.go`, `internal/hub/transport/transport.go` |
| `3f20ecae503117ff052061136b080b8f8cce2df9` | Trạng thái kết nối được đồng bộ; lỗi WebSocket handshake bất đồng bộ không dừng thử kết nối lại. `agent/connection_manager.go` |
| `b684ac9910652bdc39b61982b079fb5213dcc07a` | SSH handshake có deadline và được xóa sau khi handshake thành công, giới hạn ảnh hưởng của peer im lặng lên Hub updater. `internal/hub/systems/system.go` |
| `09a277f07487c1e78a82eb331b0e8dcc4906e753` | Đọc/ghi status hệ thống dùng accessor đồng bộ sau khi publish, kể cả khi chuyển alert. `internal/hub/systems/system.go`, `system_manager.go` |
| `921ded1af0f955ddaba1dabafe0d361f1d12fb7d` | Truy cập vòng đời WebSocket được đồng bộ/atomic giữa goroutine. `internal/hub/ws/ws.go` |
| `a77ed345d76daf7676dbf2d7b68873ffdd240537` | Hệ thống báo cáo chỉ cập nhật monitor do chính nó sở hữu; chỉ ghi history sau khi xác nhận quyền sở hữu. `internal/hub/systems/system.go` |
| `0484c54919ff071e21e1db3197060fc8277213e0` | Subscription, gửi, history và evaluation của alert kiểm tra quyền hiện tại của user với system. Không thể đổi user/system qua cập nhật thường; Hub startup áp dụng lại rule owner sau collection snapshot. `internal/alerts/*`, `internal/hub/collections.go`, migration snapshot |
| `24881fbfdf3feba4567e41dc011ea283716873ee` | Phản hồi chi tiết systemd dùng `private` và `Vary: Authorization`, tránh proxy cache dùng chung làm lộ dữ liệu. `internal/hub/api.go` |
| `97db8bd199b10f90ce854ddc913491d43fda9216` | Hủy request SSH đóng connection/session đang chạy và đồng bộ trạng thái transport. `internal/hub/transport/ssh.go` |
| `1997984325de5057bee0bd5b309fa534dac5d639` | `SSHTransport` là nơi duy nhất sở hữu SSH connection của mỗi system; polling, monitor sync và ZFS dùng chung, reconnect chỉ mở một connection thay thế. `internal/hub/transport/ssh.go`, `internal/hub/systems/system.go` |
| `4ffd83677d18fe285ce1647da9d64ed4b2798f40` | WebSocket/SSH fallback, listener và connection của Agent xử lý retry/shutdown mà không làm service bị treo. `agent/client.go`, `connection_manager.go`, `server.go` |
| `36c14e903d73b226a2df5bbf7ab4fbf1943c2906` | Agent không gửi payload stats trùng lặp cho Hub phiên bản cũ. `agent/server.go` |
| `7305eda972cb423bd51ac18f4966312b346360d0` | Hub gửi keepalive giữa các lần poll để Agent chậm nhưng còn hoạt động giữ được connection. `internal/hub/systems/system.go`, `internal/hub/ws/ws.go` |
| `d91f06f3c533b231346dc1d0ca5bf0bca265e462` | Dừng keepalive sau ba phút Agent im lặng để không duy trì connection cũ vô hạn. `internal/hub/systems/system.go`, `internal/hub/ws/ws.go` |
| `f8d3fe1817b2ac977c051243d6ffc9077a000ba3` | Field JSON scalar dùng `omitzero` với json v2; xác minh hình dạng response ở cả hai encoder. CBOR tag không đổi. `internal/entities/*`, `internal/hub/server.go` |

Thay đổi rủi ro cao khác nhưng không xếp P0 được nêu ở P1: kiểm tra hạn
certificate, tùy chọn bỏ xác thực TLS, cấu hình resolver DNS monitor và
`HUB_HEADERS`. `d89f844a2a0abc7474555f0635344245456f82d6` thêm tùy chọn
`skipTlsVerify` riêng từng monitor, mặc định false và chỉ áp dụng HTTP; không
mở rộng phạm vi. `a99fe5e99725edf2a8ff582f5b21793fc88731f2` dùng một TLS dial
riêng không xác minh chỉ để đọc metadata certificate; HTTPS probe thực tế vẫn
xác minh certificate bình thường.
`d324ccc0977d0347e4f94529490fefbc72e54900` parse `HUB_HEADERS` tùy chỉnh,
từ chối tên/giá trị sai mà không in giá trị bí mật, và bỏ qua override dành
riêng `X-Token`/`X-Beszel`; cần giữ handshake xác thực hiện tại.

## Tính năng và tương thích P1

* `bb1b39928e317592159e65dbcac1de12510d49b8` thêm network monitoring đầu-cuối:
  probe và CBOR sync phía Agent, lưu monitor/history/alert phía Hub, migration,
  retention và giao diện nhúng. Rà soát như một tính năng schema/protocol/UI
   xuyên suốt; các sửa quyền sở hữu/ủy quyền P0 ở trên thuộc contract an toàn
  của tính năng này.
* `86ab0fae8b7b2d87251a90a4fa56138d17cca553` và các commit sau thêm RSSI Wi-Fi
  từng interface, history gọn và biểu đồ. Collector Darwin upstream gọi
  `/usr/bin/osascript`; iOS dùng chung Darwin build selection nhưng không có
  macOS AppleScript. Dải này không có collector Wi-Fi iOS native. P003 cần
  tách collector Darwin khỏi iOS hoặc tạo no-op được hỗ trợ rõ ràng; không
  tuyên bố code này có RSSI iOS.
* `c25408651f076c59c13f35f27016461ada8a9093` thêm số lượng package update;
  `8613cfe548c882b57c5748efd7007ecbe2ba2553` thêm danh sách đầy đủ qua request/API
  và bảng trên trang system; `bc2278e7e9820825ce990fb2fe7fca4b9fe9a1c3` thêm
  parser xbps. Giữ tương thích message/version và hành vi parser cho Agent
  Linux từ xa.
* `7626e5e3c84806a67d9e10c8791a6aca9991d2ce` thêm log service systemd qua action
  Agent và Hub endpoint mới. Hub chạy trên iOS vẫn phải nhận log từ Agent từ xa
  được hỗ trợ dù iOS không có collector systemd.
* Các bổ sung network monitor gồm resolver DNS cấu hình được
  (`f7528a02082a6fd2a54857470e40ff5a54782801`), hạn certificate HTTPS
  (`a99fe5e99725edf2a8ff582f5b21793fc88731f2`), tùy chọn TLS rõ ràng
  (`d89f844a2a0abc7474555f0635344245456f82d6`) và thay đổi compare/view/chart.
  Cần giữ tương thích migration, ID, polling interval và alert history.
* `017ebb091aa64596f9545c89e67f6c880f8bb121`,
  `32d469285382521bae4ad3f368a7045f32ff2f5e` và
  `bb23f99b3eeaced9608e11e280e7c986dba650be` chuyển trạng thái ZFS và sửa tính
  dung lượng root dataset qua Agent, Hub persistence và UI. Sửa SMART NVMe,
  kiểu số và rescan cùng sửa GPU JSON/power fallback làm đổi telemetry dùng
  chung; cần parity fixture kể cả khi collector phụ thuộc platform.
* View/language setting, package-update UI, log timestamp và vòng đời chart
  ảnh hưởng frontend nhúng. Hub iOS build cùng frontend này, vì vậy cần build
  lại `dist` bằng workflow Bun đã ghim.

## Thay đổi platform và hỗ trợ P2

Hành vi LXC/cgroup và nl80211 chỉ có trên Linux, tên adapter Wi-Fi Windows,
kiểm tra systemd và lấy journal, đóng gói Docker/Helm, hành vi installer OpenWrt,
chỉnh chart/layout ít quan trọng, catalog dịch, fixture test portable và
changelog được ghi nhận P2 hoặc applicability `none` tùy trường hợp. Chúng
không tạo collector iOS native. Một số vẫn ảnh hưởng dữ liệu/giao diện mà Hub
trên iOS nhận từ Agent Linux, Windows hoặc macOS; các dòng đó được đánh dấu
`Hub-only` hoặc `shared`, không bỏ qua như no-op iOS.

Hai commit dependency `195afc10b8924ef4d6afc5d409eb8a8e1c43b96c` và
`e30e980854408d46b2e8837a14a0c0d771a18545` được đối chiếu với call site và
lockfile đã ghim. Lock P001 phân giải valibot 1.3.1 và Vite 7.1.11; advisory
database hiện tại báo valibot `<=1.4.1` và một số advisory Vite đến 7.3.4.
Ứng dụng gọi Valibot `parse`/`safeParse`, không gọi helper `flatten()` bị ảnh
hưởng; CI production chạy `vite build`, không chạy Vite dev server. Target đóng
băng phân giải valibot 1.5.0 và Vite 7.3.7; audit endpoint không báo advisory
trực tiếp cho hai package này. Năm phát hiện khác còn lại tại target
(`@lingui/cli`, `@lingui/vite-plugin`, `braces`, `chokidar`, `micromatch`) cần
được phân loại ở P014. Bằng chứng này hỗ trợ xếp hardening dependency P2;
không chứng minh có đường khai thác production hay toàn bộ dependency graph đã
sạch.

## Quan hệ phụ thuộc giữa subsystem

| Bề mặt thay đổi | Phụ thuộc / cung cấp dữ liệu cho | Hệ quả tích hợp |
| --- | --- | --- |
| Agent collector và monitor task | CBOR/WebSocket action trong `internal/common`, model `internal/entities` | Giữ field tag, giá trị tùy chọn, retry/timeout và hành vi với phiên bản cũ. |
| Hub polling và SSH transport | Agent handshake, monitor sync, SMART/ZFS/systemd response | Giữ một transport owner, cancellation, retry và resync sau reconnect. |
| Hub record và migration | Monitor, alert, package update, ZFS và system field | Kiểm tra migration/snapshot, default, index, ownership và history cùng nhau. |
| Frontend và asset nhúng | Hub API, PocketBase record, `internal/site/dist` | Build bằng Bun lock cố định và kiểm tra API trước khi link hai binary iOS. |
| Collector platform từ xa | Entity/history/UI dùng chung của Hub | Collector Linux/Windows/macOS vẫn ảnh hưởng telemetry hiển thị trong Hub iOS. |

## Rủi ro với bản vá iOS hiện có

Đủ 35 commit iOS lịch sử được ánh xạ vào 13 logical patch ID trong CSV patch.
11 commit upstream có đường dẫn thay đổi trùng với commit iOS; đây là điểm cần
review, không phải xung đột dòng đã được chứng minh:

| Đường dẫn chung | Commit upstream | Bản vá iOS cần giữ |
| --- | --- | --- |
| `agent/system.go` | `86ab0fae8b7b2d87251a90a4fa56138d17cca553`, `16e3fbadce2026d5591aa08c6518e1c5754fa1a2`, `4ffd83677d18fe285ce1647da9d64ed4b2798f40`, `01d91728f00d234f68cc6ebd2ab7df09226a2789`, `ef840e0e6d8f066419a351c25312acb2d5323af7` | `IOS-AGENT-SYSTEM-HOOK`: bỏ qua Darwin CPU probe và gọi `adjustPlatformSystemDetails()` |
| `.github/workflows/docker-images.yml` | `97de1471d7ddb92d3138d9baaecb1f58db0bc06b`, `7748651f5cb3ee0ccd3d3e6141f315ee5d20454e` | `IOS-RELEASE-ISOLATION`: loại `v*-ios.*` khỏi Docker publish chung |
| `readme.md` | `e68372dce417fe922c0a27aded5ce4e4bfb66cf3`, `4a5915b141bfa96627f5493e5174ec8bc36734d5` | `IOS-PORT-DOCUMENTATION`: giữ nội dung user/contributor riêng cho port |
| `internal/site/bun.lock` | `1ab38e57c59c470ae579caa268b7d9dcdbda95ba`, `e30e980854408d46b2e8837a14a0c0d771a18545` | `IOS-FRONTEND-BUILD`: giữ Bun 1.4.0 và frozen install tương thích |

Patch map 35 commit cũng tách riêng bản vá runtime A7, workflow build iOS/
CGO/iPhoneOS SDK, battery `AppleARMPMUCharger`, metadata sysctl/plist, hook
Agent dùng chung, Hub startup/migration, frontend embedding, LaunchDaemon,
installer state/rollback, checksum pin, release assets, test và tài liệu. Dải
upstream không sửa installer iOS, file battery, script vá A7 hay tên release
asset. Chỗ `agent/system.go` bị đụng cần giữ cả điều kiện
`runtime.GOOS != "ios"` và `adjustPlatformSystemDetails()`.

## Bàn giao P003–P018

Repository có bối cảnh P001/P002 nhưng không định nghĩa tiêu đề/phạm vi chính
thức P003–P018. Nhãn P007–P017 bên dưới là bucket định tuyến tạm thời suy ra từ
đường dẫn đã đổi và roadmap được yêu cầu; cần đối chiếu roadmap trước khi bắt
đầu triển khai.

| Prompt | Bàn giao |
| --- | --- |
| P003 | Lập ma trận tương thích, risk register, rollback strategy và acceptance criteria iOS 12/A7. Bắt đầu từ 21 invariant P0 và rủi ro Wi-Fi/Darwin build-tag. |
| P004 | Review/tích hợp 25 commit đến hết v0.20.0, giữ mọi commit upstream sau điểm fork ban đầu. |
| P005 | Review/tích hợp 116 commit kế tiếp đến hết v0.21.0. |
| P006 | Review/tích hợp 37 commit cuối đến `bb23f99b3`. |
| P007 | Agent collector, platform/build tag, Wi-Fi, package update, SMART, ZFS và GPU metrics. |
| P008 | WebSocket, SSH fallback, header, TLS, serialization và vòng đời kết nối. |
| P009 | Hub API, auth, session, alert và system ownership. |
| P010 | PocketBase snapshot/migration, record, history và tương thích dữ liệu. |
| P011 | Network-monitor model, polling, chart và UI monitor. |
| P012 | Package update, systemd và log telemetry. |
| P013 | Dashboard, setting, chart, alert UI và localization. |
| P014 | Go/Bun/npm dependency, TypeScript và build frontend nhúng. |
| P015 | Regression fixture, audit validator và CI gate. |
| P016 | Cô lập release iOS, workflow build, installer contract và checksum. |
| P017 | Tài liệu Anh/Việt, changelog và đối chiếu patch map. |
| P018 | Acceptance tích hợp và kiểm tra release candidate sau triển khai; không thuộc P002. |

P002 dừng ở bước bàn giao này. Không bắt đầu triển khai P003 hay tích hợp
upstream.

## Xác thực, điểm chưa rõ và giới hạn bằng chứng

Validator xác định là
[`scripts/audit/validate-upstream-v021-matrix.py`](../scripts/audit/validate-upstream-v021-matrix.py);
test Git fixture tổng hợp cùng negative case nằm tại
[`tests/audit/test_validate_upstream_v021_matrix.py`](../tests/audit/test_validate_upstream_v021_matrix.py).
Output lệnh và exit status chi tiết được ghi trong
[`upstream-v021-test-results.txt`](upstream-v021-test-results.txt).

`Unresolved audit rows: 0`. Câu hỏi còn lại để bàn giao là thiếu định nghĩa
prompt chính thức P003–P018, Agent iOS có nên bỏ hẳn collector Wi-Fi macOS
upstream hay không, và năm phát hiện advisory dependency còn lại ở target.
Audit chạy trên Linux cloud workspace: không tuyên bố đã build binary iOS bằng
iPhoneOS SDK/Go hoặc thử trên iPad thật. Không phát hành, deploy, tạo tag hay
merge mã upstream trong audit. Mọi publish P002 lên `iOS` sẽ qua merge PR được
yêu cầu; không push trực tiếp vào `main` hoặc `iOS`.

Lệnh `go` hiện có là wrapper không phải Go (`go version` exit 1 với
`Go: Unknown option: version`); Bun và `xcodebuild` không được cài. Không tuyên
bố đã chạy test bằng Go toolchain, frozen Bun install hay build binary iOS.
