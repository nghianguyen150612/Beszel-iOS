# P008 — Tương thích số liệu hệ thống iOS và pin

**Gate nguồn/fixture/native-build P008: PASS** tại
`a096d1eabe11e41667a726470f502e21a6b93650`. Xác minh HEAD cuối gồm tài liệu
và merge integration được ghi trong [PR #10](https://github.com/nghianguyen150612/Beszel-iOS/pull/10).

P008 xác minh mã nguồn, fixture tổng hợp, hồi quy dùng chung và bản dựng native.
Không khẳng định Beszel v0.21 đã chạy đúng trên iPad vật lý. Gate C còn chờ
P009–P012 và các tiêu chí phụ thuộc thiết bị; P018 / Gate F chịu trách nhiệm
chạy phần cứng. Không triển khai P009 trong mốc này.

## Trạng thái ban đầu và nguồn bằng chứng

Clone mới theo yêu cầu chọn nhánh phân biệt chữ hoa `iOS`. Integration ban đầu:
`2b5d8fa31e6b6da25a3073c5148b3375301f40dd`; nhánh iOS sản phẩm:
`0c3e4cd22ac24264834c401ad412e44aeda4e4f3`; main/upstream đóng băng:
`bb23f99b3eeaced9608e11e280e7c986dba650be`. Các ref đều khớp trước khi sửa.
Target đóng băng và mọi SHA upstream chuẩn vẫn là tổ tiên: **178/178**.
Cả **13 bản vá iOS logic** đều được đối chiếu trong bản ghi bảo toàn JSON.
Nhánh làm việc là `ios/ios-system-metrics`, chỉ nhắm tới
`ios/upstream-v021-integration`, dùng commit thông thường và merge commit.

Đã đọc mã nguồn ban đầu, bản ghi bảo toàn P006, kế hoạch nghiệm thu/tương thích,
sổ rủi ro/crosswalk, báo cáo runtime/native P007 và ghi chú build/thiết bị lịch
sử. Bản sao nguồn ban đầu và log kiểm thử baseline được lưu ngoài repository.
Bộ Go chuẩn và build/kiểm thử Bun thực đều đạt ở baseline. SHA đã kiểm thử,
lệnh, hash và kết luận workflow nằm trong
[JSON bằng chứng](ios-system-metrics-validation.json); PR ghi thêm xác minh
HEAD cuối vì một commit không thể chứa SHA của chính nó. Báo cáo EN/VI mô tả
cùng triển khai và giới hạn nghiệm thu.

Chỉ dùng làm tham chiếu lịch sử: iPad mini 2, A1489 / iPad4,4, Apple A7, arm64,
iOS 12.5.7, Amethyst / Procursus. Bản ghi 0.19 không thay đổi. Pin 42/100,
dung lượng thô 3989 và quan sát bộ nhớ dùng làm tham chiếu fixture đã làm sạch;
không hardcode định danh, tổng RAM hoặc phiên bản OS cho mọi thiết bị.

## Lỗi đã xác nhận và thay đổi có giới hạn

| Đường thu thập | Trạng thái ban đầu / lỗi xác nhận | Hành vi mới |
| --- | --- | --- |
| CPU | Thiếu baseline đầu có thể tính usage từ lúc khởi động; nguồn rỗng có thể trả lỗi nil; chưa loại reset riêng lẻ/NaN; map baseline toàn cục thiếu khóa dùng chung. | Trả trạng thái không có mẫu cho baseline/lỗi; kiểm tra counter hữu hạn, đơn điệu và định danh; mutex quanh thu thập runtime; kiểm tra reset/phục hồi/độ dài per-core. Giữ cách seed từ khoảng mặc định. |
| RAM | gopsutil Darwin dereference lookup `vm_kernel_page_size` không kiểm tra và trừ counter trang 32-bit. Fallback cache thiếu suy ra cache từ nhóm bộ nhớ khác. | Adapter Mach native chỉ cho iOS, kích thước trang OS và `hw.memsize`; tính toán mở rộng/bão hòa; giữ available = free + inactive; không suy diễn cache/swap. Giữ nguồn Linux/macOS và cgroup Linux. |
| Đĩa | Có thể đoán I/O root từ đĩa macOS hoạt động nhiều nhất; phép trừ byte/op unsigned có thể wrap; thiết bị thiếu/lỗi giữ throughput ExtraFs cũ. | Danh tính usage iOS dựa trên partition/mountpoint thực, mặc định `/`; I/O vật lý ghi rõ không có. Rate dùng chung tránh underflow/overflow; biến mất xóa baseline mọi khoảng; snapshot ExtraFs mới không alias trạng thái filesystem có thể sửa. |
| Mạng | Fixture chỉ có receive hoặc transmit bị loại sai. Danh sách startup không rỗng che interface mới. Reset coi counter tích lũy mới là traffic khoảng; phép nhân overflow; tổng bất thường không khớp rate interface. | Chấp nhận traffic một chiều; cập nhật danh tính mỗi poll; interface UP có địa chỉ với counter 0 hoặc NICS chỉ định được tạo baseline; rollback không báo rate; lỗi ngưỡng xóa rate tổng và từng interface cùng nhau. |
| Pin | Subprocess không deadline; nhân dung lượng overflow; boolean thiếu ngụ ý xả pin; có thể nhận khóa lồng nhiều dòng/block thiếu hoặc trộn. | Giữ nguồn AppleARMPMUCharger; giới hạn deadline/output, cache chung, parser portable chính xác, block hoàn chỉnh với thuộc tính trực tiếp và đúng class, phần trăm an toàn, trạng thái unknown và bỏ dữ liệu khi lỗi. |
| Metadata | Mọi `iPad4,*` đều thành A7, kể cả ID chưa biết; metadata kế thừa có thể tồn tại khi nguồn iOS thiếu. | Danh sách A7 chính xác; SoC chưa biết giữ tên chung; nguồn sysctl/plist có seam kiểm thử, xóa giá trị không có và xác minh count. |

Symbol VM chưa kiểm tra và giả định driver đĩa macOS là rủi ro xác nhận từ mã
nguồn; không phải crash quan sát trên iPad v0.21, cũng không chứng minh mọi
driver iOS đều không có counter. P008 tránh các giả định này; adapter I/O vật
lý tương lai cần bằng chứng nguồn/thiết bị riêng. File CPU non-Linux **không**
vô hiệu CPU chuẩn: nó chỉ tắt cgroup.

## Ngữ nghĩa CPU, RAM và filesystem

CPU giữ đường Darwin của gopsutil v4.26.9:
`host_statistics(HOST_CPU_LOAD_INFO)` và
`host_processor_info(PROCESSOR_CPU_LOAD_INFO)` qua libSystem. Tick user/system/
nice/idle chuyển thành giây; tỷ lệ dùng delta counter, không phụ thuộc tần số
tick hoặc khoảng cache. iowait/steal bằng 0 vì nguồn này không có các nhóm đó;
không thay thế bằng số đo khác. Tổng/breakdown hữu hạn, trong 0–100. Mẫu thiếu,
không đổi hoặc không hợp lệ bỏ breakdown/per-core tùy chọn; scalar CPU bắt
buộc giữ mặc định 0 hiện có của protocol. Vector per-core bị giữ lại khi độ
dài/danh tính đổi hoặc, trên iOS, mâu thuẫn với logical-thread count đã biết.
Không tổng hợp per-core. Lỗi nguồn ngắt tính liên tục; mẫu hợp lệ sau tạo
baseline rồi phục hồi. Giữ guard `runtime.GOOS != "ios"` cho probe model;
fallback `/proc/cpuinfo` chỉ chạy Linux.

RAM đọc `hw.memsize`, `host_statistics(HOST_VM_INFO)` và kích thước trang OS
qua CGO chỉ cho iOS. Host port được deallocate mỗi lần. Available = free +
inactive, giới hạn bởi tổng vật lý; used = total − available. Counter trang
được mở rộng trước phép tính; số liệu không nhất quán/tạm thời bão hòa trong
giới hạn total. Active/inactive/wired không biến thành cache Linux.
`MEM_CALC=htop` không diễn giải lại nguồn iOS. Đo cache/buffer riêng và swap
**không có**: bỏ field swap tùy chọn; scalar `MemBuffCache` bắt buộc giữ mặc
định 0. Số 0 không chứng minh thiết bị không có cache. Thiếu RAM vật lý dùng
mặc định scalar bắt buộc hiện có, không sao chép mẫu cũ. Hồi quy Linux cho
`calculateHostMemoryUsage`, `saturatingSub`, cgroup và ZFS ARC đều đạt.

Discovery dùng getfsstat Darwin lúc khởi tạo và statfs usage khi poll, không
duyệt cây filesystem. Mặc định `/` là filesystem root, kể cả root read-only;
không khẳng định đây là toàn bộ lưu trữ vật lý của thiết bị. `/private/var`
chỉ được chọn theo partition/cấu hình thực. `FILESYSTEM` nhận mountpoint/device
và tên tùy chỉnh; `EXTRA_FILESYSTEMS` chọn thêm rõ ràng. Cùng mount/device được
loại trùng; mount liên quan không tự thêm hoặc cộng. Báo cáo volume APFS riêng
có thể dùng chung dung lượng container và không được hiểu là dung lượng vật
lý cộng được. Thiếu device dùng mountpoint làm khóa usage nội bộ, không bịa tên
`/dev`. Mẫu không truy cập được/dung lượng 0 an toàn; poll sau thử lại. Root
được refresh mỗi poll; extra giữ duration cache usage cấu hình hiện có.
**I/O đĩa vật lý không có trên iOS**: bỏ throughput, counter tích lũy và thời
gian bằng key tùy chọn hiện có. Không thêm adapter Linux diskstats, SMART hoặc
ZFS. Giữ discovery Linux ZFS/Btrfs/pool và wrap counter thời gian 32-bit.

## Ngữ nghĩa mạng, pin và metadata

Counter byte interface Darwin đến từ nguồn routing/sysctl của gopsutil. Rate
là byte/giây từng interface, cộng thành tuple bandwidth upstream; tổng thô giữ
ngữ nghĩa gốc. Mỗi khoảng cache có baseline riêng. Mẫu đầu, interface biến mất/
tái xuất hiện, reset, lỗi nguồn, khoảng thời gian 0 và timestamp tương lai không
bịa rate. Phép nhân/cộng không wrap; giữ ngưỡng bất thường hiện có. Giữ lọc mặc
định loopback/docker/bridge/veth/bond/cali. VPN, cellular hoặc AWDL đang hoạt
động theo chính sách hiện có, không bị loại vì tên lạ. Whitelist NICS exact/
wildcard vượt lọc tự động, kể cả counter 0; blacklist kết hợp lọc tự động. Với
interface chưa theo dõi và cả hai counter 0, UP cùng địa chỉ xác nhận tính hợp
lệ; interface down/không địa chỉ chờ, traffic một chiều sau đó có thể làm nó
hợp lệ. Discovery lỗi không bịa interface. Throughput không biến thành RSSI;
native RSSI vẫn thuộc P012.

Nguồn pin giữ `/usr/sbin/ioreg -r -c AppleARMPMUCharger -l`, không thay bằng
AppleSmartBattery macOS hoặc chế độ plist. `CommandContext` giới hạn mỗi lần
**2 giây**; Wait reap tiến trình trực tiếp bị kill, WaitDelay giới hạn chờ pipe
kế thừa **250 ms**. Stdout tối đa **1 MiB**, bỏ stderr, không log output hệ thống
thô. Cache **60 giây** áp dụng cho thành công và lỗi, tránh subprocess realtime
đắt và trả bản sao. Lỗi xóa dữ liệu pin đọc được; poll đủ khoảng tiếp theo có
thể phục hồi. Không timer thường trực, retry nền hoặc shell.

Entry production gọi đúng acquisition/parser portable được kiểm thử trên host.
Block registry hoàn chỉnh dạng recursive/legacy có pipe được hỗ trợ; loại khóa
BatteryData/child lồng, sai registry class, khóa trùng, nhiều block pin, output
bị cắt, dung lượng âm/sai/overflow và maximum 0/thiếu. Phần trăm dùng số nguyên
an toàn overflow, trong 0–100. Thiếu BatteryInstalled giữ giả định trước đây;
No rõ ràng hoặc giá trị installed sai loại pin. Raw maximum tùy chọn chỉ cung
cấp full-charge capacity khi hợp lệ. Enum upstream không đổi: unknown 0, empty
1, full 2, charging 3, discharging 4, idle 5. Disconnected hoặc charging đã biết
được ưu tiên; full/empty/idle cần đủ thông tin nguồn. Field charging thiếu/sai
không ngụ ý charging/discharging. Giữ tên `Primary` và payload `Battery`/
`Batteries`.

Metadata giữ `hw.machine`, `hw.physicalcpu`, `hw.logicalcpu`, `kern.hostname`,
`kern.osrelease`, SystemVersion.plist/ProductVersion và `runtime.GOARCH`. Os vẫn
là `system.Darwin` upstream (số 1); OsName là `iOS` hoặc `iOS <ProductVersion>`,
không nhãn macOS. Mapping A7 chính xác: `iPad4,1`–`iPad4,9`, `iPhone6,1`/
`iPhone6,2`; ID chưa biết là `Apple SoC (<id>)`. Giá trị tùy chọn thiếu để trống;
count sai/quá bất hợp lý bằng 0; cores không vượt threads đã biết; plist sai/
thiếu giữ nhãn iOS thuần. Fixture không-arm64 kiểm tra architecture lấy từ
nguồn runtime đã chọn, không hardcode.

## Kiểm thử, protocol và giới hạn hiệu năng

Fixture là tổng hợp. Hình dạng/nguồn pin ở
`agent/battery/testdata/ios/README.md`; số lịch sử lấy STEP 39 của bản ghi 0.19
không đổi. CPU, trang RAM, partition, counter byte, metadata và lỗi được inject
vào boundary production, không sao chép logic sang test.

| Nhóm mới | Test cấp cao nhất | Subtest có tên |
| --- | ---: | ---: |
| CPU | 2 | 10 |
| RAM | 2 | 6 |
| Đĩa | 3 | 9 |
| Mạng | 2 | 0 (chuỗi kịch bản lifecycle/rate) |
| Pin | 4 | 37 |
| Metadata | 2 | 22 |
| Phép tính rate dùng chung | 1 | 0 |
| Field hiện diện trong payload | 1 | 3 |

Helper subprocess là hạ tầng, không tính vào bốn nhóm test pin. Phủ CPU đầu/
ổn định/idle/bão hòa/reset/sai; lỗi/phục hồi nguồn và thay core; RAM thường/
thấp/0/sai/rất lớn, cache/swap thiếu và poll lặp; filesystem root-only/liên
quan/chọn rõ/thiếu/read-only, I/O không có, device biến mất, thời gian/reset/
lỗi/phục hồi; interface động/idle/một chiều/VPN/chỉ định/loại trừ, nhiều khoảng
và overflow; pin sai/thiếu/lồng/trạng thái/deadline/output limit/lỗi/cache/
concurrency/phục hồi; metadata biết/chưa biết/thiếu và plist.

Không đổi layout entity dùng chung, key CBOR, tag JSON, dependency hoặc mã
production transport. Test JSON v1/v2 và CBOR phủ scalar 0 bắt buộc, pin empty
hợp lệ, số đo có mặt, swap/đĩa/RSSI vắng, interface một chiều và mảng per-core.
Giữ khoảng cache và khóa Agent. Baseline CPU thêm khóa chung; ExtraFs mới sao
chép counter có thể sửa. Bộ Agent đầy đủ trên macOS hosted phát hiện Unix
socket path quá dài trong fixture SSH cũ; thư mục tạm ngắn sửa fixture mà không
giảm assertion listener/payload hoặc đổi mã transport.

Test subprocess có giới hạn và race dùng chung xác nhận logic/cleanup host.
Không đo overhead CPU/RAM Apple A7, tính đúng tick native, quyền Mach hoặc độ
trễ collector iOS thực. Không đưa ước lượng overhead.

## Xác minh đã chạy và nghiệm thu

Kết quả local chính xác, SHA nguồn candidate, URL hosted, checksum native và
kết luận artifact được ghi trong JSON và PR. Audit/readiness, 49 test audit (35 P007), sync 10, sentinels 45, bootstrap 54,
installer 814, bộ Go chuẩn, race dùng chung, build Bun thật/44 test và build
Linux Agent/Hub đều đạt. Cả 95 asset frontend thật đã embed. Test Agent/race/GPU
và build Agent/Hub macOS hosted đạt. Native Agent/Hub cùng gate target/runtime/
frontend/checksum/repeat-build đạt. Build native giữ Go 1.27.1, Bun 1.4.0, GOROOT riêng vá
A7, CGO/iPhoneOS SDK, ARM64 v8.0, platform iOS 2/minimum 12.0, so sánh cache
độc lập, kiểm tra instruction runtime, embed frontend thật và SHA256SUMS.
Chạy lại HEAD cuối gồm tài liệu; job release được skip.

| Tiêu chí | Bằng chứng P008 | Nghĩa vụ còn lại |
| --- | --- | --- |
| IOS-AGENT-01 | Mapping, fixture lỗi sysctl/plist và native linkage | P018 đối chiếu sysctl/plist thiết bị với payload |
| IOS-AGENT-02 | Test tổng hợp/lỗi/phục hồi CPU/RAM/đĩa/mạng; build | P018 accounting kernel thực, mount và counter Wi-Fi/VPN; I/O vật lý ghi rõ không có |
| IOS-AGENT-03 | Test đúng parser/acquisition/cache/state/deadline pin | P018 output thật và quan sát charging/discharging/idle/full |
| Gate C | Phần source/test/build P008 | P009–P012 và nghiệm thu vật lý áp dụng |
| Gate F | Không kết quả thiết bị mới | P018 chạy và nghiệm thu Apple A7 vật lý |

Field không có và câu hỏi chỉ thiết bị được giữ rõ ràng; chạy macOS hoặc link
iOS không chứng minh hành vi kernel iOS. Bản ghi nghiệm thu bổ sung phân biệt
các nhóm này. Không truy cập DB production, chạy lệnh thiết bị, signing, install,
reboot, đổi service, tạo release/tag hoặc deployment. Nhánh sản phẩm `iOS` và
`main` đóng băng không thay đổi.


CI native candidate nguồn: [38526577598](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38526577598).
Installer: [38526580705](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38526580705).
Cả hai thành công tại SHA nguồn trên; release được skip. Xcode 15.4 (15F31d),
iPhoneOS SDK 17.5. Payload tải xuống được kiểm tra độc lập:
SHA-256 Agent `2a63b88ffc6b28bd3ca72ef2650e7d06df03e81efd8bee274841ad87ed8f8082`;
Hub `0cc91d9b516158cbc81ee50c71d99c8f860ccab74d785fbf165521b33b428b7f`.
Cả hai là ARM64 MH_EXECUTE, platform 2/minimum 12.0, Go 1.27.1, CGO=1,
VCS revision sạch khớp candidate. Hash HEAD cuối khác theo metadata VCS và
được ghi trong review cuối của PR. Hai lần lỗi fixture macOS và một lần hủy
vì nguồn đã được thay thế vẫn nằm trong JSON; không lần nào là bằng chứng đạt.
CLI tải artifact trực tiếp trả HTTP 403; công cụ artifact GitHub kết nối tải
được hai archive, ZIP digest upload và checksum payload đều khớp. Không bỏ qua
gate xác minh.
