# P011 — Telemetry cập nhật gói Procursus/APT chỉ đọc

## Trạng thái và nguồn gốc

P011 bổ sung telemetry cập nhật gói cục bộ cho cài đặt Procursus APT được hỗ trợ trên iOS. Adapter chỉ mô phỏng nâng cấp bằng metadata đã có trong bộ nhớ đệm. Adapter không làm mới index, tải gói hoặc thay đổi gói đã cài.

Kho: https://github.com/nghianguyen150612/Beszel-iOS
Nhánh: ios/procursus-package-updates
Nhánh đích: ios/upstream-v021-integration
SHA tích hợp ban đầu: c2bd226f54f52a5fb2efaa33a63aa9fb632f4216
SHA iOS sản phẩm ban đầu: 0c3e4cd22ac24264834c401ad412e44aeda4e4f3
Frozen main: bb23f99b3eeaced9608e11e280e7c986dba650be
PR: https://github.com/nghianguyen150612/Beszel-iOS/pull/13

Các refs baseline được tải và xác minh trước khi sửa. Commit frozen main vẫn là tổ tiên của integration. Validator ma trận upstream xác nhận đủ 178 commit upstream và 13 bản vá iOS logic. Validator readiness đầy đủ cũng thành công. Các checkpoint triển khai đã được đẩy lên nhánh tính năng; PR vẫn Draft đến khi hoàn tất kiểm tra hosted và review cuối.

Nguồn Procursus chính thức đã xem là ProcursusTeam/Procursus tại b6185221bf1ce3dda8b4baeee61e75e3f7c7d399. Recipe APT ghim APT 2.9.4; recipe và metadata gói xác định ngữ cảnh khóa Procursus. Tài liệu apt-get APT 2.9.4 mô tả chế độ simulation và no-download. Đây là bằng chứng từ mã nguồn và fixture, không chứng minh binary hoặc cấu hình trên iPad tham chiếu.

## Phát hiện và layout được hỗ trợ

Factory riêng cho iOS chỉ được chọn khi GOOS=ios. Phát hiện package manager Linux vẫn ở nhánh Linux hiện tại; macOS và Windows không nhận factory Procursus.

Adapter kiểm tra các đường dẫn binary cụ thể:

- Rootful: /usr/bin/apt-get, /usr/bin/apt-config, /usr/bin/dpkg-query
- Rootless: /var/jb/usr/bin/apt-get, /var/jb/usr/bin/apt-config, /var/jb/usr/bin/dpkg-query

Cả ba phải là file thường có quyền thực thi. Các truy vấn có giới hạn và allowlist kiểm tra apt-get --version, apt-config dump, và dpkg-query xác nhận apt cùng procursus-keyring đã cài. Adapter đọc các cấu hình Dir, Dir::State, Dir::State::status, Dir::State::lists, Dir::Etc, source-list, source-parts, main-config và config-parts do APT báo cáo. Adapter xác nhận file status đã phân giải, thư mục danh sách APT, index Packages đã cache và các đường dẫn cấu hình liên quan. Adapter không giả định /var/lib/dpkg hoặc /var/jb/var/lib/dpkg nếu chưa hỏi cấu hình APT.

Triển khai hỗ trợ dạng Procursus rootful và rootless khi cấu hình APT đã cài phân giải nhất quán package DB, lists và cây /etc/apt. Nếu hai candidate trỏ đến các cài đặt xung đột, thiếu metadata, lệnh chưa đầy đủ, biến thể/keyring không hỗ trợ hoặc metadata bắt buộc không đọc được, adapter sẽ không khả dụng. Adapter không sửa chữa hoặc tự điền cài đặt. Vị trí binary khác sẽ bị từ chối an toàn.

Context phát hiện ba giây giới hạn các probe lúc khởi động. Factory iOS không tìm hoặc chọn package manager Linux như DNF, Zypper, Pacman, APK hay XBPS.

## Truy vấn chỉ đọc và bằng chứng toàn vẹn

argv duy nhất để đếm cập nhật là:

    apt-get -s --no-download -o Debug::NoLocking=1 -o Dir::Cache::pkgcache= -o Dir::Cache::srcpkgcache= dist-upgrade

Các đối số được truyền trực tiếp cho đường dẫn apt-get đã xác thực, không qua shell. Tên executable và argv chính xác nằm trong allowlist ở ranh giới subprocess. Tài liệu APT 2.9.4 đã xem xác nhận simulation không thực hiện hành động hoặc thay đổi hệ thống. Cờ no-download chặn tải gói. Simulation tắt locking; Debug::NoLocking nêu rõ chính sách đó. Đích package-cache rỗng ngăn ghi cache APT. stdout giới hạn 1 MiB, stderr bị bỏ, môi trường chỉ còn LC_ALL=C. Context truy vấn iOS giới hạn thời gian chạy 20 giây.

Truy vấn không gọi apt-get update và không tạo metadata repository. Thiếu index làm truy vấn trả unknown. Không cài prerequisite. Kết quả phản ánh góc nhìn APT cache cục bộ; độ mới tùy vào lần gần nhất chủ thiết bị cập nhật metadata sẵn có. Adapter không cập nhật metadata và không gọi kết quả là trạng thái repository trực tiếp.

Fixture APT dùng thư mục tạm, source loopback cục bộ, package database biệt lập, status/available, lists, extended state, đường dẫn archive/cache, file quản trị dpkg và cấu hình APT. apt-get của host chạy trong bubblewrap, tắt network namespace và mount filesystem chỉ đọc. Listener loopback ghi nhận không có kết nối. Snapshot bất biến gồm 30 file/thư mục, hash file, mode, kích thước và mtime. Test chạy liên tiếp ba lần. Mỗi lần xác nhận cả 30 entry fixture cùng metadata không đổi; SHA256 snapshot trước và sau khớp ở từng lần: `2bc11d3c9d0bc989db04c1b1f161c9fafc15973d2e5734fcb38654e1aff669d9`, `09c9d24415ee87d2aee29f14bac67f368ca54252a50175dfac85d8a29ac77c62` và `4aa97a989d6dbf6a3201406a94f710726e79997e59a4e45be6809cce2edfd4ed`.

Host sandbox có APT 3.0.3. Procursus ghim APT 2.9.4 và các tùy chọn tương thích đã được đối chiếu tài liệu của phiên bản đó. Fixture không chứng minh hành vi APT trên iPad tham chiếu. Output APT, thông tin xác thực, URI source và lỗi subprocess không được ghi log thông thường hoặc gửi Hub.

## Phân tích output và ngữ nghĩa kết quả

Fixture apt_procursus_simulate.txt đã được làm sạch và tổng hợp theo grammar simulation APT được tài liệu hóa; fixture không được thu từ thiết bị Procursus vật lý. Test bao phủ zero/one/nhiều cập nhật; tên dài; chữ số, gạch nối, dấu cộng và architecture qualifier; epoch, tilde, dấu cộng và dấu câu trong version; mô tả repository và warning; dependency mới chưa có version đã cài; package held, removed, obsolete, version bằng nhau và downgrade; entry trùng/xung đột; output malformed, bản địa hóa, rỗng hoặc bị cắt; thiếu current/candidate version; output quá lớn và giới hạn số package.

Chỉ Inst có version đã cài và candidate mới hơn nghiêm ngặt mới được đếm. Dependency mới chưa có version đã cài không phải cập nhật. Package bị gỡ, held, version bằng nhau và downgrade không được tính. Entry trùng hoàn toàn được khử trùng; duplicate xung đột hoặc summary không khớp entry khiến lần kiểm tra không khả dụng. Kết quả được sắp xếp ổn định. Output tối đa 1 MiB và 8.192 package, thấp hơn nhiều giới hạn uint16.

Nhãn repository security đặc thù distro của APT không phải nguồn advisory Procursus đáng tin cậy. Kết quả Procursus luôn SecurityKnown=false và mọi PackageUpdate.Security=false. Không tạo security count hoặc advisory giả.

Model và wire contract dùng chung không đổi: system stats Agent giữ trường []uint16 hiện có; nil nghĩa là chưa có count thành công/khả dụng, còn []uint16{0} là lần kiểm tra thành công không thấy cập nhật. Test JSON và CBOR xác nhận hai trạng thái vẫn khác nhau. Action GetPackageUpdates, entity system.PackageUpdates, mapping CBOR/JSON, giao diện Hub và telemetry Agent Linux/macOS/Windows từ xa được giữ nguyên. Không thêm PocketBase migration hoặc trường protocol.

## Timeout, lịch, cache và phục hồi

Adapter dùng cấu hình PACKAGE_UPDATES_INTERVAL và cơ chế tìm biến môi trường có prefix hiện tại. Mặc định dùng một giờ hiện có; duration dương được dùng; zero tắt manager; giá trị âm hoặc sai theo chính sách default an toàn hiện có.

Phát hiện bị giới hạn ba giây. Truy vấn cập nhật có context 20 giây, giới hạn output 1 MiB, bỏ stderr, os/exec WaitDelay hai giây và hủy process group trên Unix để dừng child giữ stdout cùng truy vấn. Lệnh chạy trực tiếp qua exec.CommandContext. Test bao phủ thiếu lệnh, không có quyền execute, exit khác zero, timeout, cancellation, output quá cỡ, child giữ stdout và phục hồi sau query lỗi.

Các caller chia sẻ một lần refresh chạy nền. Refresh theo interval đã cấu hình, không theo từng yêu cầu stats. Lần thử lỗi bị throttle đến interval sau và báo unknown, không phải zero. Lỗi xóa chi tiết cache cũ và CheckedAt bằng zero. Trên iOS, count stats không có timestamp trở thành nil khi kết quả hết hạn bắt đầu refresh. Action chi tiết có thể giữ danh sách thành công trước đó chỉ kèm CheckedAt ban đầu trong lúc refresh; nếu refresh lỗi thì xóa danh sách. Shutdown hủy context manager và chờ refresh thoát trong thời gian ngắn. Lỗi không chặn telemetry CPU, memory, battery, network hay disk.

## Regression và native validation

Test tập trung phát hiện Procursus, chọn path, parser, argv allowlist, invariant DB chỉ đọc, sandbox bắt buộc tắt mạng, giới hạn package count, timeout, cancellation, child cleanup, cache/phục hồi, serialization và handler pass với Go 1.27.1 đã pin. Focused race suite pass ba lần; test lỗi process pass lặp lại. Canonical Go suite và toàn bộ Agent race suite pass. Build Linux Agent và Hub pass.

Validator chọn source iOS/macOS/Linux ARM64 CGO=1 xác nhận chỉ build iOS chọn adapter và process helper Unix. Regression Linux tập trung bao phủ APT, DNF, Zypper, Pacman, APK, XBPS, handler/cache/security và chặn package manager trong container. Audit suite (49 test), upstream matrix/readiness, iOS sentinels (45), upstream-sync (10), bootstrap (54) và installer lifecycle (814) đều pass. Hai test kết nối đồng thời có một lần lỗi trong các lần chạy suite trước; canonical chạy tuần tự cuối cùng pass, và P011 không sửa file implementation/test P009. Hosted native build, installer workflow và regression P007–P010 cuối cùng pass trên source SHA `d777ad42fe8cfc2e2dec5ab5f942659abc49eae9`; URL workflow và evidence artifact được liệt kê bên dưới.

Validation hosted cuối cùng chạy trên source SHA `d777ad42fe8cfc2e2dec5ab5f942659abc49eae9`:

- iOS Build: PASS — https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38675307900
- Installer lifecycle: PASS — https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38675307584
- Hosted job package-update pass các test detection/parser/argv/failure. Query APT disposable tắt mạng bị skip rõ ràng vì Ubuntu runner không có sandbox filesystem read-only và network-disabled; fixture bubblewrap cục bộ riêng pass ba lần như mô tả ở trên.
- Runner macOS 14.8.9 arm64; Xcode 15.4; iPhoneOS SDK 17.5 và clang của Xcode. Dùng Go 1.27.1 và Bun 1.4.0 đã pin. Frontend production đã cài dependency, sinh catalog, test và build; Hub embed thật có 95 asset.
- Chọn source iOS pass. Agent và Hub build hai lần với cache riêng và byte-identical. Patch runtime A7 cô lập cho Go 1.27.1 pass; kiểm tra instruction không thấy truy cập CNTVCT/CNTFRQ trong cả hai binary.
- Cả hai output là Mach-O `MH_EXECUTE` arm64 cho iOS platform 2, minimum iOS 12.0. VCS provenance ghi revision `d777ad42fe8cfc2e2dec5ab5f942659abc49eae9` và `vcs.modified=false`.
- SHA256 Agent: `9afffa924917c46bd48f18175bb8532d85692664edd642522bf7e5798f46754e`; Hub: `7eec915001fb1ec3d7a36072b50afefef258147a792b9d1da610dd4af2676a8e`. Workflow đã xác minh checksum hai artifact.
- `release-ios` bị skip. Không tạo release, tag hay deployment.

Ở source candidate này, các gate implementation, test, compatibility và hosted build của P011 đều pass. PR vẫn Draft trong lúc review diff cuối và xác minh yêu cầu approval. SHA nguồn trên là candidate đã test native; các commit sau đó chỉ cập nhật evidence và metadata PR.

## Giới hạn đã xác nhận và acceptance

Lỗi baseline đã xác nhận: factory hiện hữu trả nil trên mọi nền tảng không phải Linux, nên Agent iOS không khởi tạo telemetry cập nhật gói. P011 chỉ bổ sung đường iOS và giữ detector Linux. Shared schema không có freshness riêng cho package count system stats; P011 dùng nil/absent cho unknown hoặc count stale khi iOS đang refresh. Chi tiết package giữ timestamp CheckedAt hiện hữu.

Fixture cloud chỉ xác nhận hành vi triển khai. Không truy cập iPad mini 2 / A1489 / iPad4,4 trên iOS 12.5.7 với Amethyst và Procursus. Binary package manager, APT version/path cấu hình, quyền đọc, độ mới metadata và hành vi truy vấn thực tế trên máy vẫn chưa xác minh. Kiểm tra Procursus vật lý chỉ đọc được để P018 / Gate F. Không sửa package DB thật, database Hub production hoặc thiết bị.

Acceptance P011 còn chờ hosted native CI trên final HEAD, PR checks áp dụng được, review cuối và merge thường. Canonical Go/race suite đã pass. Gate C có thể tiến sau khi P011 đạt; native Wi-Fi RSSI vẫn thuộc P012. Gate database/restore và release/deployment vẫn pending. Chưa bắt đầu P012.
