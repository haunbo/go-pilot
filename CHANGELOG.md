# Changelog

All notable changes to this project are documented in this file.

## [1.0.0] - 2026-11-15

### Added

- `pkg/conf`: YAML/JSON configuration loading, dotted keys, struct decoding,
  environment overrides and change hooks.
- `pkg/conf/datasource/file`: filesystem config source with optional fsnotify
  hot reload.
- `pkg/conf/datasource/nacos`: Nacos config source with change listening.
- `pkg/xlog`: structured zap logging, file rotation and request-id context
  fields.
- `pkg/xerr`: HTTP-aware application errors and generic API error codes.
- `pkg/server/xecho`: Echo server wrapper with request IDs, access logs, panic
  recovery, CORS, body limits, English/Chinese validation and response helpers.
- `pkg/server/xgrpc`: gRPC server with access logs, panic recovery, request-id
  propagation, message size limits and TLS/mTLS.
- `pkg/client/grpc`: gRPC client with timeouts, access logs, request-id
  propagation, TLS, keepalive and readiness waiting.
- `pkg/client/redis`: go-redis client with pool settings, startup ping and
  optional access logs.
- `pkg/client/request`: JSON HTTP client with timeout, retry, auth,
  slow-request logs and request-id propagation.
- `pkg/store/xrdb`: GORM client for MySQL, PostgreSQL, SQLServer and SQLite
  (pure-Go SQLite driver) with pool settings, slow SQL logging and AutoMigrate.
- `pkg/store/xmongo`: MongoDB driver v2 client with pool settings, timeouts and
  startup ping.
- `pkg/excel`: Excel export helpers built on excelize.
- `pkg/util/*`: xcast, xcrypto, xfile, xgo, xmap, xmath, xnet, xretry, xstring
  and xtime helpers.
- `pkg/version`: toolkit version and build information.
- 单元测试与 `-race` 竞态测试。

### Changed

- All code uses Go 1.27 idioms, including `any` instead of `interface{}`.
- 本库面向 Go 1.26+ 项目，不再兼容 1.22~1.24。

## [Unreleased]

### Added

- `conf.ReloadReport`：热更新结果分为 `Applied`（已生效）与 `Restart`（需重启），
  解决"日志说 reloaded、实际没生效"无法定位的问题。
- `xlog.SetLevel` / `Logger.SetLevel` / `Logger.Level` / `xlog.Reload(key)`：
  日志级别改为 zap `AtomicLevel`，运行期可热切，子 logger 同步生效。
- `xecho.(*Server).Reload()` / `ApplyConfig()` / `Config()`：HTTP 服务可热更新
  `enableAccessLog`、`slowQueryThresholdInMilli`、`validationLocale`、
  `serviceAddress`、`debug`；端口/TLS/超时/CORS/BodyLimit 折入 `Restart`。
- `conf.SetLogFunc`：把 conf 自身的诊断日志（含 "config reloaded"）接入 xlog。

### Changed

- README 明确声明 Go 版本要求（1.27 指令 / 1.26 最低可用）与初始化顺序、
  时间字段格式等约定；「快速开始」改为内联 main 骨架。
- 访问日志中间件改为按请求读取配置快照，`enableAccessLog: false` 现在真正关闭
  访问日志（错误日志不受影响）。
- `conf` 的 reload 日志改为 `config reloaded, N key(s) changed` 并可通过
  `SetLogFunc` 重定向。
- README 新增「热更新边界」章节，按组件列出可热更新与需重启的字段。

### Fixed

- `version.String()`：被当作依赖引用时返回的是**宿主程序**的版本（`info.Main.Version`），
  不是 go-pilot 的版本。改为在 `info.Deps` 中按 module path 匹配；`replace` 指向本地
  目录、版本不可知时回退到 `Version` 常量，不再返回 `v0.0.0` 这类占位值。

### Removed

- 裁剪 `pkg/util`：删除 `xfile`、`xgo`、`xmap`、`xmath`、`xnet`、`xretry`、`xtime`
  七个子包，以及 `xstring` 的 conv/hash/json/string、`xstring.GenerateShortUUID`、
  `xcrypto` 的 Md5/RandomString/SHA256/HmacSHA256。**只保留被本库自身引用的三个函数**：
  `xcast.ToInt64Slice`（`conf.GetInt64Slice`）、`xcrypto.Bytes`、
  `xstring.GenerateUUID`（`xecho` 请求 ID）。文件锁依赖 `gofrs/flock` 随之移除。
- 删除 `pkg/app`（`ReloadManager` / `Reloader`）。热更新分发改由业务在
  `conf.OnChange` 里显式调用 `xlog.Reload` / `server.Reload`，`ReloadReport`
  仍保留在 `conf` 作为这两个方法的返回值。
- 删除 `cmd/example` 与 `Makefile` 的 `run-example` 目标。README「快速开始」
  改为内联的 main 骨架，不再依赖可运行目录。
- 移除 `.github/workflows/ci.yml`。本项目不依赖 CI，发布前在本地执行
  `go vet ./...`、`go test -race ./...`、`go build ./...` 自检。

