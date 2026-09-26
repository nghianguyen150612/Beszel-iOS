# Kiến trúc

## Quan hệ với bản gốc

```text
henrygd/beszel
      |
      v
    main            nhánh bám sát bản gốc
      |
      + công việc tương thích iOS (pin, metadata hệ thống,
      |                         bản vá runtime, workflow
      |                         build/phát hành hợp nhất)
      v
     iOS            nhánh port Beszel-iOS đang hoạt động (nhánh mặc định)
```

Nguyên tắc thiết kế: nhánh `iOS` vẫn phải nhận ra là Beszel. Không viết lại Agent, không viết lại Hub, không đổi ngữ nghĩa cơ sở dữ liệu, không lược bỏ tính năng của bản gốc.

## Quan hệ runtime (giữ nguyên như bản gốc)

```text
Beszel Hub
    |
    | Giao thức Beszel Agent (giống bản gốc)
    v
Beszel-iOS Agent
```

Agent thu thập chỉ số của máy và phục vụ cho Hub trên cổng đã cấu hình; Hub lưu lịch sử và phục vụ bảng điều khiển. iOS chỉ thay đổi *cách* Agent lấy metadata pin/hệ thống và *cách* build cả hai binary — không thay đổi giao thức. Chi tiết giao thức nằm trong mã nguồn bản gốc, tài liệu này không nhắc lại.

## Bố cục runtime trên máy (iOS)

```text
/usr/local/bin/beszel-agent          Binary Agent (ký bằng ldid)
/usr/local/bin/beszel-hub            Binary Hub (ký bằng ldid)

/var/lib/beszel-agent                Trạng thái Agent
/var/lib/beszel-hub                  DB / tài khoản / cấu hình Hub (KHÔNG BAO GIỜ được coi là rác)

/Library/LaunchDaemons/dev.beszel.agent.plist
/Library/LaunchDaemons/dev.beszel.hub.plist
```

Cổng mặc định: Agent `45876`, Hub `8090`. Kiểm tra tình trạng Hub: `http://127.0.0.1:8090/api/health`.

Binary phải nằm dưới `/usr/local/bin` (`chown root:wheel`, `chmod 755`, `ldid -S`). Chạy từ thư mục home hay `/tmp` trước đây từng gặp lỗi sandbox của iOS.

## Bản đồ mã iOS-specific

- `agent/battery/battery_ios.go` — đọc pin qua lớp charger của ioreg.
- `agent/battery/battery_darwin.go` — đường đi macOS, nay là `darwin && !ios`.
- `agent/system.go` — bỏ qua `cpu.Info()` của Darwin trên iOS, gọi `adjustPlatformSystemDetails()`.
- `agent/system_platform_ios.go` / `agent/system_platform_other.go` — metadata sysctl/plist cho iOS so với no-op.
- `.github/scripts/patch-go-ios-arm64-runtime.py` — workaround `procyield` cho A7 (lúc build, trên runner macOS).
- `.github/workflows/ios-build.yml` — pipeline hợp nhất (Agent + Hub + SHA256SUMS → một artifact; với tag `v*-ios.*`, một job phụ sẽ kiểm tra tag rồi phát hành đúng ba file đó thành GitHub Release Latest).
- `.github/workflows/ios-installer.yml` — CI installer: kiểm tra cú pháp POSIX của bootstrap/engine, tham chiếu nhánh `iOS` chuẩn, kiểm tra pin `engine_sha` của bootstrap, và các bộ test installer.
- `install.sh` — bootstrap an toàn khi pipe: tải engine qua HTTPS, đối chiếu SHA-256 ghim, chạy bản đã xác minh.
- `scripts/ios/install-beszel.sh` — engine vòng đời do bootstrap thực thi và được lưu thành `/var/lib/beszel-ios/manager.sh`.
- `tests/bootstrap-test.sh` — test contract bootstrap, gồm cả trường hợp lệch hash bị từ chối.

## Tình trạng phân phối

**Đã có:** pipeline CI hợp nhất. Mỗi lượt chạy tạo `build/ios/` với đúng `beszel-agent-ios-arm64`, `beszel-hub-ios-arm64`, `SHA256SUMS`, tải lên thành artifact `beszel-ios-arm64`. Đẩy một tag hợp lệ dạng `v<upstream-version>-ios.<revision>` (khớp `beszel.Version` trong `beszel.go`, nằm trên lịch sử `iOS`) sẽ phát hành đúng ba file đó thành GitHub Release không phải draft, không phải prerelease, gắn cờ Latest. Các automation theo tag của bản gốc (`release.yml`, `docker-images.yml`) bỏ qua tag `v*-ios.*` nên bản phát hành iOS vẫn sạch. Bộ cài đặt một lệnh (bootstrap `install.sh` ghim `engine_sha` rồi chạy engine `scripts/ios/install-beszel.sh`, 1.0.0) dùng bản Latest cho các lượt cài mới Agent / Hub / Agent+Hub (kiểm tra checksum, ký `ldid`, tự sinh LaunchDaemon trên máy, kiểm tra tình trạng Hub) và cho cập nhật giao dịch: nó resolve tag Latest một lần mỗi lượt chạy, ghim mọi lượt tải về đúng tag bất biến đó, theo dõi bản đã cài trong `/var/lib/beszel-ios/install-state` (binary đã cài được ký `ldid` nên hash của chúng không bao giờ đem so với `SHA256SUMS`), staging binary đã ký trước khi dừng dịch vụ, giữ sao lưu `/usr/local/bin/*.bak`, tự quay lại khi kiểm tra health/khởi động thất bại, giữ plist nguyên từng byte, và không bao giờ đụng tới `/var/lib/beszel-hub`. Nó còn có chẩn đoán chỉ đọc (giá trị key của Agent không bao giờ hiển thị), sửa lỗi thận trọng, và cấu hình lại qua các giao dịch plist đã kiểm tra, sao lưu sang `/Library/LaunchDaemons/*.plist.bak` với khả năng tự quay về cấu hình cũ; cấu hình lại và sửa kiểu chỉ khởi động lại không làm đổi trạng thái cài đặt, chỉ những lần sửa có cài binary Latest vừa tải mới ghi nhận bản mới. Gỡ ứng dụng (Agent, Hub hoặc Agent+Hub) chạy theo giao dịch có quay lại, không cần mạng hay `ldid`, luôn giữ `/var/lib/beszel-agent` và `/var/lib/beszel-hub` theo mặc định, và chỉ xóa dữ liệu khi có xác nhận gõ đúng (`DELETE AGENT DATA` / `DELETE HUB DATA`); trạng thái từng thành phần được xóa riêng và metadata rỗng của bộ cài đặt chỉ xóa bằng đường dẫn chính xác.

Không còn công việc vòng đời bộ cài đặt nào nữa: cài đặt, cập nhật, sửa/cấu hình lại và gỡ bỏ đều đã làm xong. Việc còn lại là kiểm thử (chạy end-to-end trên máy thật, thêm thiết bị/SoC iOS).

## Bootstrap installer và nguồn gốc manager

```text
curl -fsSL https://raw.githubusercontent.com/nghianguyen150612/Beszel-iOS/iOS/install.sh | sudo sh
        |
        v
install.sh                     bootstrap nhỏ, POSIX, an toàn khi pipe
        |  tải scripts/ios/install-beszel.sh chỉ qua HTTPS
        |  (curl --proto '=https', --proto-redir '=https', không chạy qua pipe)
        |  đối chiếu SHA256(engine) == engine_sha ghim trong install.sh
        |  chạy bản đã xác minh dưới dạng /tmp/beszel-installer.XXXXXX/manager.sh
        v
scripts/ios/install-beszel.sh  engine vòng đời (menu, cài/cập nhật,
        |                      sửa/cấu hình lại, gỡ/purge, CLI bền vững,
        |                      status/diagnostics/doctor, điều khiển service)
        v
/var/lib/beszel-ios/manager.sh nguyên vẹn từng byte chính tệp engine đã chạy
                                giao dịch (sao chép cục bộ, không tải lại)
```

Hai lớp ghim hoàn toàn tách biệt: binary ứng dụng Beszel được ghim qua tag phát hành bất biến `v<version>-ios.<revision>` kèm `SHA256SUMS` (giải pháp này không đổi), còn engine vòng đời installer được ghim bằng `engine_sha` trong `install.sh`. CI (`.github/workflows/ios-installer.yml`) fail khi engine đổi mà `engine_sha` không được cập nhật cùng commit.

Phạm vi bảo mật, nói cho chính xác: với một bootstrap vừa tải, thiết kế mang lại chọn engine tất định, bảo vệ TOCTOU giữa lúc tải bootstrap và lúc tải engine (đổi nhánh giữa chừng làm dừng trước khi thực thi), và lưu manager đã xác minh. Chính bootstrap vẫn tải qua HTTPS từ nhánh `iOS` có thể thay đổi, nên đây không phải ký số phát hành đầy đủ.

Việc nhận manager mới là tường minh: chạy lại lệnh bootstrap sẽ tải bootstrap mới hơn với digest ghim mới hơn. `beszel-ios update` thông thường chỉ làm mới binary ứng dụng Beszel và không bao giờ thay manager. Trạng thái kiểm thử: các thao tác vòng đời ứng dụng đã kiểm chứng trên thiết bị (xem [device-validation.vi.md](device-validation.vi.md)); các lệnh CLI service/doctor chỉ test bằng fixture; kiến trúc bootstrap được kiểm chứng trên host/fixture/CI.

## Kiến trúc phân phối tương lai (dự kiến, chưa làm)

```text
GitHub Release
     |
     +-- beszel-agent-ios-arm64
     |
     +-- beszel-hub-ios-arm64
     |
     +-- SHA256SUMS
```

```text
install.sh (bootstrap ghim checksum)
     |
     +-- đối chiếu và chạy scripts/ios/install-beszel.sh
           |
           +-- Install Agent
           +-- Install Hub
           +-- Install Both
           +-- Update (giữ /var/lib/beszel-hub)      [đã làm]
           +-- Repair / reconfigure                 [đã làm]
           +-- Uninstall (giữ dữ liệu; xóa cần xác nhận gõ) [đã làm]
```

Thư mục `packaging/launchd/` trong tương lai sẽ chứa hai plist LaunchDaemon; `scripts/ios/` đã chứa engine vòng đời (`install-beszel.sh`). Xem [ios-port-status.vi.md](ios-port-status.vi.md) và [ios-build-notes.vi.md](ios-build-notes.vi.md).
