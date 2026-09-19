# Ghi chú build iOS

[English](ios-build-notes.md)

Tài liệu này ghi lại các yêu cầu đặc biệt để build được binary iOS chạy được. Tham chiếu chính là file hợp nhất `.github/workflows/ios-build.yml`, nơi sở hữu cả bước build lẫn bước phát hành theo tag. Muốn biết từng bước chính xác, hãy đọc trực tiếp file YAML.

## Mục tiêu

- `GOOS=ios`, `GOARCH=arm64`
- `CGO_ENABLED=1` (bắt buộc cho target `ios/arm64`)
- Deployment target tối thiểu: iOS 12.0 (`-mios-version-min=12.0`)
- Runner: macOS (`macos-14`) với Xcode iPhoneOS SDK

## Chuỗi công cụ

1. Vá runtime Go trước: `sudo python3 .github/scripts/patch-go-ios-arm64-runtime.py` (workflow sẽ fail nếu bước này fail).
2. Build frontend của Hub trước với Bun 1.4.0 đã ghim (`internal/site`: `bun install --frozen-lockfile`, `bun run build`). File `bun.lock` đã check-in phản ánh `overrides` trong `package.json`, nên bắt buộc cài frozen và tái lập được từ checkout sạch.
3. Resolve SDK và compiler: `xcrun --sdk iphoneos --show-sdk-path`, `xcrun --sdk iphoneos --find clang`. Không bao giờ hardcode đường dẫn SDK.
4. Tạo một clang wrapper truyền `-arch arm64 -isysroot <SDK> -mios-version-min=12.0`, rồi dùng chung làm `CC` cho cả hai lượt build.
5. Build bằng wrapper vào `build/ios/`:
   - Agent: `go build -trimpath -ldflags="-s -w" -o build/ios/beszel-agent-ios-arm64 ./internal/cmd/agent`
   - Hub: `go build -trimpath -ldflags="-s -w" -o build/ios/beszel-hub-ios-arm64 ./internal/cmd/hub`
6. Kiểm tra cả hai binary: `file`, `otool -hv`, `otool -l | grep LC_BUILD_VERSION|LC_VERSION_MIN_IPHONEOS`. Workflow sẽ fail khi thiếu/rỗng file, output không phải Mach-O, không phải arm64, thiếu load command iOS, hoặc thiếu metadata deployment `12.0`.
7. Sinh checksum bằng `shasum -a 256` tương thích macOS (không đường dẫn tuyệt đối):
   - `shasum -a 256 beszel-agent-ios-arm64 beszel-hub-ios-arm64 > SHA256SUMS`
8. Kiểm tra bằng `shasum -a 256 -c SHA256SUMS`.
9. Tải lên một artifact `beszel-ios-arm64` với đúng ba file trên. Với push nhánh và dispatch thủ công, workflow dừng ở đây; với push tag `v*-ios.*`, job phụ `release-ios` chạy tiếp (xem Releases bên dưới).

## Vì sao lúc cài cần `ldid`

iOS đã jailbreak vẫn yêu cầu pseudo-sign cho binary chạy trực tiếp tự biên. Sau khi chép lên máy:

```sh
chown root:wheel /usr/local/bin/beszel-agent /usr/local/bin/beszel-hub
chmod 755 /usr/local/bin/beszel-agent /usr/local/bin/beszel-hub
ldid -S /usr/local/bin/beszel-agent /usr/local/bin/beszel-hub
```

Cài dưới `/usr/local/bin`; chạy từ `$HOME` và `/tmp` trước đây từng lỗi thực thi/sandbox.

## Vì sao có bản vá runtime Apple A7

Trên Apple A7 / iOS 12, hàm `runtime.procyieldAsm` ARM64 của Go hiện đại đọc `CNTVCT_EL0`, vốn fault (`SIGILL`) trong userspace cũ này. Script vá thay thân hàm đó bằng vòng lặp `YIELD` kiểu cũ và từ chối chạy nếu không thấy đúng chuỗi `CNTVCT_EL0` mong đợi (để fail to trên Go lạ thay vì làm hỏng toolchain). Nó phải chạy trước mọi lượt build iOS cho target đã kiểm thử. SoC/iOS mới hơn có thể không cần — điều đó chưa thử.

## Yêu cầu build frontend Hub

Hub nhúng web UI. Trước khi biên dịch binary Hub:

```sh
cd internal/site
bun install --frozen-lockfile
bun run build
```

Bun ghim ở 1.4.0 trong `.github/workflows/ios-build.yml`
(`oven-sh/setup-bun@v2` với `bun-version: 1.4.0`). Bản build CI macOS
vẫn là chuẩn cho binary iPhoneOS thật.

Bỏ qua bước này sẽ ra Hub thiếu frontend hiện tại. Build agent không cần bước này.

## Bản phát hành

Đẩy tag dạng `v<upstream-version>-ios.<revision>` (ví dụ
`v0.19.0-ios.1`) sẽ kích hoạt cùng workflow và chạy thêm job
`release-ios` (`ubuntu-latest`, `contents: write`), phụ thuộc vào
job build macOS:

1. Tag phải khớp `v<beszel.Version>-ios.<positive integer>`, trong đó
   `beszel.Version` đọc từ `beszel.go` tại commit gắn tag — không
   hardcode trong workflow. Bản thân hằng `beszel.Version` vẫn báo
   phiên bản base của bản gốc (không có hậu tố `-ios.N` trong mã).
   Khi Beszel gốc lên phiên bản mới, revision iOS reset
   (ví dụ `v0.20.0-ios.1`).
2. Commit gắn tag phải là ancestor của `origin/ios`, nên tag lỡ tạo
   trên lịch sử `main` không liên quan sẽ bị từ chối mà không phát hành
   gì. Rebuild lịch sử (tag nằm sau HEAD `ios` hiện tại) vẫn cho phép.
3. Job tải artifact `beszel-ios-arm64` từ chính lượt chạy của nó rồi
   kiểm tra lại: cả ba file tồn tại và khác rỗng, `SHA256SUMS`
   chứa đúng hai mục binary, và `sha256sum -c` đạt.
4. Nó phát hành bằng `gh release create <tag> --title "Beszel iOS <tag>"
   --latest` kèm ba file — không draft, không prerelease, gắn Latest —
   rồi assert trạng thái release và `/releases/latest` trỏ đúng tag mới.

Path filter trong trigger workflow chỉ áp dụng cho push nhánh `ios`;
GitHub không xét path filter cho push tag, nên tag phát hành luôn build. Automation
theo tag của bản gốc được loại trừ tag iOS: `release.yml`
(GoReleaser) và `docker-images.yml` đều loại `v*-ios.*`, nên bản phát hành iOS
không bao giờ dính asset bản gốc hay build Docker.

Mỗi bản phát hành vì vậy cung cấp đúng contract cho installer tương lai:

- `.../releases/latest/download/beszel-agent-ios-arm64`
- `.../releases/latest/download/beszel-hub-ios-arm64`
- `.../releases/latest/download/SHA256SUMS`

Installer (`install.sh`, v1.0.0) đi theo redirect `/releases/latest`
một lần mỗi lượt chạy để resolve tag hiện tại (kiểm tra theo
`v<upstream>-ios.<rev>`), rồi ghim mọi lượt tải của lượt đó về
`.../releases/download/<tag>/...` để `SHA256SUMS` và cả hai binary luôn
từ cùng một bản bất biến. Cài đặt và cập nhật thành công ghi lại
tag bản phát hành cùng SHA-256 gốc của asset vào
`/var/lib/beszel-ios/install-state`; binary đã ký `ldid` trên máy không
bao giờ đem so hash với `SHA256SUMS`.
