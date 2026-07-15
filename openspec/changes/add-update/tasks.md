## 1. Update package

- [x] 1.1 Add `internal/update/update.go`: `LatestRelease`, `AssetName`, `ChecksumFor`, `CompareVersions`, download + sha256, `extractBinary`, `verifyCosign`, `Apply`, and the `Run` orchestration
- [x] 1.2 Bound decompression (LimitReader), preserve existing binary permissions, resolve symlinks, Windows move-aside path

## 2. Command

- [x] 2.1 Add `cmd/update.go` with `--check` / `--force`, wired to `internal/update.Run` and `internal/version.Version`

## 3. Tests

- [x] 3.1 Unit tests: `CompareVersions`, `ChecksumFor`, `AssetName`, `extractBinary`
- [x] 3.2 `httptest`-backed tests for `LatestRelease`, `Run` (already-latest, `--check`)

## 4. Docs

- [x] 4.1 Document `update` in the README commands section

## 5. Validation

- [x] 5.1 `just test` passes with race detector
- [x] 5.2 `go vet` and `gosec` clean on the new code
- [x] 5.3 `gofmt -s -l` clean
- [x] 5.4 Manual `sshroute update --check` against the live API
