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
- GitHub Actions CI, unit tests and race tests.

### Changed

- All code uses Go 1.27 idioms, including `any` instead of `interface{}`.
