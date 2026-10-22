# Ghi chú build Beszel-iOS

Tài liệu này ghi lại các yêu cầu đặc biệt để build được binary iOS chạy được. Tham chiếu chính là file hợp nhất `.github/workflows/ios-build.yml`, nơi sở hữu cả bước build lẫn bước phát hành theo tag. Muốn biết từng bước chính xác, hãy đọc trực tiếp file YAML.

## Mục tiêu

- `GOOS=ios`, `GOARCH=arm64`
- `CGO_ENABLED=1` (bắt buộc cho target `ios/arm64`)
- Deployment target tối thiểu: iOS 12.0 (`-mios-version-min=12.0`)
- Runner: macOS (`macos-14`) với Xcode iPhoneOS SDK

## Chuỗi công cụ

1. Dùng đúng bản Go trong `go.mod` và Bun 1.4.0. Chạy test runtime/build validator với fixture tạm.
2. Trong `internal/site`, chạy `bun install --frozen-lockfile`, `bun run sync`, `bun test`, rồi `bun run build`. Hash lock phải giữ nguyên; khôi phục PO text đã track được sinh lại, giữ asset production thật và catalog đã compile.
3. Chạy `.github/scripts/check-ios-source-selection.py` để kiểm tra lựa chọn file Go và dependency thật cho iOS, macOS, Linux. Chạy test parser GPU macOS và build macOS bằng toolchain gốc.
4. Sao chép **toàn bộ** GOROOT setup-go vào thư mục riêng theo job. Kiểm tra cấu trúc/phiên bản, chọn bằng GOROOT/PATH và GOTOOLCHAIN=local, rồi chạy `.github/scripts/patch-go-ios-arm64-runtime.py` trên bản sao. Dùng GOCACHE iOS riêng. Hash source runtime gốc phải giữ nguyên; không cần sudo.
5. Dùng `.github/scripts/ios-clang.sh` làm CC cho cả hai build native. Wrapper resolve SDK thiết bị và Apple clang bằng `xcrun --sdk iphoneos`, từ chối component thiếu hoặc cờ đích không tương thích, truyền `-arch arm64 -isysroot <SDK> -mios-version-min=12.0`. Xóa CGO/GOFLAGS host kế thừa.
6. Build Agent và Hub vào `build/ios/` với `GOOS=ios GOARCH=arm64 GOARM64=v8.0 CGO_ENABLED=1`, `-trimpath`, `-ldflags="-s -w"`. Lặp lại bằng cache sạch độc lập và so sánh cả hai output.
7. Build binary kiểm tra riêng không strip bằng cùng toolchain đã patch; chạy `.github/scripts/verify-ios-runtime-binary.py` để kiểm tra toàn bộ symbol procyield. Giữ cờ strip production.
8. Kiểm tra từng output bằng `file`, `otool -hv`, `otool -l` native. Sinh SHA256SUMS chỉ có tên file bằng `shasum -a 256`, xác minh cả hai entry, rồi chạy `.github/scripts/verify-ios-artifacts.py build/ios --frontend internal/site/dist`. Validator kiểm tra ARM64 MH_EXECUTE, platform iOS thiết bị/minimum 12.0, entry point, Go entry package/CGO, payload chính xác và byte frontend được embed.
9. Tải artifact review gồm đúng `beszel-agent-ios-arm64`, `beszel-hub-ios-arm64`, `SHA256SUMS`. JSON kiểm tra nằm trong artifact review riêng. Chạy theo nhánh/dispatch bỏ qua `release-ios`; quy trình này không gọi lệnh ký, phát hành hay deploy.

Xem [review runtime A7](ios-a7-runtime-compatibility.vi.md),
[kiểm tra native](ios-native-build-validation.vi.md) và
[bằng chứng cấu trúc](ios-build-validation.json). Hai build giống nhau trong cùng
môi trường đã xác minh chỉ chứng minh phép so sánh đó; không giả định tái lập
byte giữa SDK, bản toolchain hoặc source commit khác nhau.

## Vì sao lúc cài cần `ldid`

iOS đã jailbreak vẫn yêu cầu pseudo-sign cho binary chạy trực tiếp tự biên. Sau khi chép lên máy:

```sh
chown root:wheel /usr/local/bin/beszel-agent /usr/local/bin/beszel-hub
chmod 755 /usr/local/bin/beszel-agent /usr/local/bin/beszel-hub
ldid -S /usr/local/bin/beszel-agent /usr/local/bin/beszel-hub
```

Cài dưới `/usr/local/bin`; chạy từ `$HOME` và `/tmp` trước đây từng lỗi thực thi/sandbox.

## Vì sao có bản vá runtime Apple A7

Môi trường Apple A7 / iOS 12 lịch sử gặp lỗi ở đường CNTVCT_EL0 của Go ARM64
`runtime.procyieldAsm` mới. Candidate P023 hiện dùng Go 1.27.3; script kiểm tra
SHA source runtime upstream và chỉ chấp nhận toàn bộ source thực thi đã review
của đúng phiên bản này rồi thay bằng vòng
YIELD có guard giá trị 0. Bản không hỗ trợ, cấu trúc lạ và lần gọi thứ hai thất
bại, không ghi. Patch chạy trước mọi compile iOS native trên bản sao cách ly.
Kiểm tra source/binary không chứng minh 0.21 chạy không có SIGILL: nghiệm thu A7
thật vẫn thuộc P018 / Gate F.

## Yêu cầu build frontend Hub

Hub nhúng web UI. Trước khi biên dịch binary Hub:

```sh
cd internal/site
bun install --frozen-lockfile
bun run sync
bun test
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
2. Commit gắn tag phải là ancestor của `origin/iOS`, nên tag lỡ tạo
   trên lịch sử `main` không liên quan sẽ bị từ chối mà không phát hành
   gì. Rebuild lịch sử (tag nằm sau HEAD `iOS` hiện tại) vẫn cho phép.
3. Job tải artifact `beszel-ios-arm64` từ chính lượt chạy của nó rồi
   kiểm tra lại: cả ba file tồn tại và khác rỗng, `SHA256SUMS`
   chứa đúng hai mục binary, và `sha256sum -c` đạt.
4. Nó phát hành bằng `gh release create <tag> --title "Beszel-iOS <tag>"
   --latest` kèm ba file — không draft, không prerelease, gắn Latest —
   rồi assert trạng thái release và `/releases/latest` trỏ đúng tag mới.

Path filter trong trigger workflow chỉ áp dụng cho push nhánh `iOS`;
GitHub không xét path filter cho push tag, nên tag phát hành luôn build. Automation
theo tag của bản gốc được loại trừ tag iOS: `release.yml`
(GoReleaser) và `docker-images.yml` đều loại `v*-ios.*`, nên bản phát hành iOS
không bao giờ dính asset bản gốc hay build Docker.

Mỗi bản phát hành vì vậy cung cấp đúng contract cho installer tương lai:

- `.../releases/latest/download/beszel-agent-ios-arm64`
- `.../releases/latest/download/beszel-hub-ios-arm64`
- `.../releases/latest/download/SHA256SUMS`

Engine vòng đời (`scripts/ios/install-beszel.sh`, do bootstrap `install.sh` thực thi, v1.0.0) đi theo redirect `/releases/latest`
một lần mỗi lượt chạy để resolve tag hiện tại (kiểm tra theo
`v<upstream>-ios.<rev>`), rồi ghim mọi lượt tải của lượt đó về
`.../releases/download/<tag>/...` để `SHA256SUMS` và cả hai binary luôn
từ cùng một bản bất biến. Cài đặt và cập nhật thành công ghi lại
tag bản phát hành cùng SHA-256 gốc của asset vào
`/var/lib/beszel-ios/install-state`; binary đã ký `ldid` trên máy không
bao giờ đem so hash với `SHA256SUMS`.
