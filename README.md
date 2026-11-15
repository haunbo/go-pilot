# go-pilot

一个面向 Go 后端服务的通用工具库，只包含可复用的 `pkg` 能力。

- Module: `github.com/haunbo/go-pilot`
- Version: `v1.0.0`
- Go: **1.27+**（`go.mod` 的 `go` 指令为 `1.27.0`）

### Go 版本说明

本库明确面向 **Go 1.26 及以上** 的项目，不再兼容 1.22/1.23/1.24 等旧版本，也不会为了向下兼容而绕开新标准库 API。

## 包一览

| 包 | 用途 |
| --- | --- |
| `pkg/conf` | 配置核心：YAML/JSON、点号路径、结构体解码、环境变量覆盖 |
| `pkg/conf/datasource/file` | 文件配置源，支持 fsnotify 热更新 |
| `pkg/conf/datasource/nacos` | Nacos 配置源，支持监听配置变更 |
| `pkg/xlog` | zap 结构化日志、文件轮转、request_id 注入 |
| `pkg/xerr` | 通用业务码 + HTTP 状态码错误类型（面向 HTTP 接口；gRPC 侧的错误映射由业务模块自行实现） |
| `pkg/server/xecho` | Echo HTTP 服务：访问日志、panic 恢复、请求 ID、CORS、BodyLimit、中英文校验、统一响应 |
| `pkg/server/xgrpc` | gRPC 服务端：访问日志、panic 恢复、request_id、TLS/mTLS |
| `pkg/client/grpc` | gRPC 客户端：超时、访问日志、request_id、TLS、连接就绪等待 |
| `pkg/client/redis` | go-redis 连接池、启动 PING、访问日志 |
| `pkg/client/request` | 基于 net/http 的 JSON 客户端：重试、鉴权、慢请求日志、request_id 透传 |
| `pkg/store/xrdb` | GORM：MySQL/PostgreSQL/SQLServer/SQLite、连接池、慢 SQL、AutoMigrate |
| `pkg/store/xmongo` | 官方 MongoDB Driver v2 封装、连接池、启动 PING |
| `pkg/excel` | 基于 excelize 的 Excel 导出 |
| `pkg/util/xcast` | `[]int64` 转换（`conf.GetInt64Slice` 使用） |
| `pkg/util/xcrypto` | 密码学随机字节 |
| `pkg/util/xstring` | RFC 4122 v4 UUID 生成（请求 ID 使用） |
| `pkg/version` | 当前二进制所链接的 go-pilot 版本 |

## 安装

```bash
go get github.com/haunbo/go-pilot@v1.0.0
```

要求 Go 1.27 或更高版本（最低可用 1.26，见上文「Go 版本说明」）。项目遵循语义化版本，`v1.x` 内保持 API 兼容。

## 快速开始

最小可运行的服务骨架，四个步骤的顺序不要调换（先配置、再日志、再组件、最后 Serve）：

```go
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/haunbo/go-pilot/pkg/conf"
	"github.com/haunbo/go-pilot/pkg/server/xecho"
	"github.com/haunbo/go-pilot/pkg/xlog"
	"github.com/labstack/echo/v4"
)

func main() {
	// 1. 配置：watch=true 时监听文件变化
	if err := conf.LoadFromFile("config.yaml", true); err != nil {
		log.Fatal(err)
	}
	conf.LoadEnvironments("GOPILOT_") // GOPILOT_SERVER_HTTP_PORT -> server.http.port

	// 2. 日志必须在任何 StdConfig()/Build() 之前初始化
	logger := xlog.Init(xlog.RawConfig("logger.default"))
	defer logger.Sync()

	// 3. 组件
	server, err := xecho.StdConfig("http").Build()
	if err != nil {
		log.Fatal(err)
	}
	server.GET("/healthz", func(c echo.Context) error {
		return xecho.Success(c, map[string]string{"status": "ok"})
	})

	// 4. 起服务、等信号、优雅退出
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() { _ = server.Serve() }()
	<-ctx.Done()
	_ = server.GracefulStop(context.Background())
}
```

配套的最小 `config.yaml`：

```yaml
server:
  http:
    port: 8080
logger:
  default:
    level: info
    dir: "-"        # "-" 或 "" 输出到 stdout
```

## 配置

### 文件配置源

```go
import (
	"log"

	"github.com/haunbo/go-pilot/pkg/conf"
)

func main() {
	// watch=true 时监听文件变化
	if err := conf.LoadFromFile("config.yaml", true); err != nil {
		log.Fatal(err)
	}
	conf.LoadEnvironments("GOPILOT_") // GOPILOT_SERVER_HTTP_PORT -> server.http.port

	conf.OnChange(func(*conf.Configuration) {
		// conf 只负责重新解析并替换配置树，组件需要自己重新读取，
		// 见「热更新边界」一节。
		log.Print("config changed")
	})

	var httpCfg struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}
	if err := conf.UnmarshalKey("server.http", &httpCfg); err != nil {
		log.Fatal(err)
	}
}
```

也可以显式使用文件配置源：

```go
import (
	filesource "github.com/haunbo/go-pilot/pkg/conf/datasource/file"
	"gopkg.in/yaml.v3"
)

source, err := filesource.New("config.yaml", true)
if err != nil {
	panic(err)
}
if err := conf.LoadFromDataSource(source, yaml.Unmarshal); err != nil {
	panic(err)
}
```

### Nacos 配置源

```go
import (
	"github.com/haunbo/go-pilot/pkg/conf"
	nacos "github.com/haunbo/go-pilot/pkg/conf/datasource/nacos"
	"gopkg.in/yaml.v3"
)

source, err := nacos.New(nacos.Config{
	ServerAddr:  "127.0.0.1",
	Port:        8848,
	NamespaceID: "",
	Group:       "DEFAULT_GROUP",
	DataID:      "demo.yaml",
	Username:    "",
	Password:    "",
}, true)
if err != nil {
	panic(err)
}
// conf.Close() will close the source; call source.Close() yourself if you
// load the source without registering it with conf.
if err := conf.LoadFromDataSource(source, yaml.Unmarshal); err != nil {
	panic(err)
}
```

`nacos.New` 的 `watch=true` 会监听服务端配置变化；`conf.OnChange` 会收到重新加载通知。

### 热更新边界

`watch=true` 只做一件事：**重新解析文件、替换内存里的配置树、触发 `OnChange`**。它不会替你重建监听器、连接池或 zap core，所以"配置 reloaded"之后是否真的生效，取决于组件有没有重新读一遍配置。

go-pilot 里各组件的能力边界：

| 组件 | 热更新后立刻生效 | 需要重启 |
|---|---|---|
| `xlog` | `level` | `dir`、`name`、`format`、轮转参数、`callerSkip` |
| `xecho` | `enableAccessLog`、`slowQueryThresholdInMilli`、`validationLocale`、`serviceAddress`、`debug` | `host`、`port`、`enableTLS` 及证书路径、`readTimeout`/`writeTimeout`/`idleTimeout`、`allowOrigins`、`bodyLimit` |
| `xgrpc`、`client/redis`、`store/xrdb`、`client/request` | 无 | 全部（连接在 `Build()` 时建立） |

接线方式：`conf.OnChange` 是全局钩子，组件不会自动订阅，需要自己在 main 里接一次（约 10 行）。只有实现了 `Reload()` 的组件才需要接：

```go
conf.OnChange(func(*conf.Configuration) {
	if report, err := xlog.Reload("logger.default"); err != nil {
		logger.Warn("reload logger failed", xlog.FieldErr(err))
	} else if report.Changed() {
		logger.Info("logger reloaded", xlog.String("result", report.String()))
	}

	if report, err := httpServer.Reload(); err != nil {
		logger.Warn("reload http failed", xlog.FieldErr(err))
	} else if report.Changed() {
		logger.Info("http reloaded", xlog.String("result", report.String()))
		if len(report.Restart) > 0 {
			logger.Warn("settings need a restart", xlog.Strings("fields", report.Restart))
		}
	}
})
```

`Reload()` 返回 `conf.ReloadReport`，把变化拆成两组：`Applied`（已生效）和 `Restart`（改了，但要重启才生效）。这样"改了配置没生效"能直接从日志看出来，而不是靠猜：

```
info  config reloaded, 2 key(s) changed      ← conf 自身的诊断日志
info  logger reloaded  result=applied=level
info  http reloaded    result=applied=enableAccessLog restart=port
warn  settings need a restart  fields=[port]
```

几个约定：

- **只有 `xlog` 和 `xecho` 实现了 `Reload()`**。`redis`、`xrdb`、`xmongo`、`xgrpc`、`client/request` 的连接在 `Build()` 时建立，改了配置必须重启；它们没有 `Reload` 方法，也不会假装能热更。
- 配置解析失败时 `Reload()` 返回 error 并**保留旧配置**，不会退回默认值继续跑。
- 派生 logger（`logger.With(...)`、组件内部的 `mod=...`）与默认 logger 共用同一个原子级别，所以 `xlog.Reload` 一处生效、全局生效。
- `conf` 自身的诊断日志走标准库 `log`。想让它进 xlog（带级别和格式），初始化日志后调用 `conf.SetLogFunc(func(format string, args ...any) { xlog.Infof(format, args...) })`。
- 运行期临时调级别可以直接 `xlog.SetLevel("debug")`，不必改文件。

### 配置示例

```yaml
server:
  http:
    host: 0.0.0.0
    port: 8080
  grpc:
    host: 0.0.0.0
    port: 9092

logger:
  default:
    level: info
    format: json
    dir: "-"

redis:
  default:
    addr: 127.0.0.1:6379

database:
  mysql:
    type: mysql
    dsn: "root:root@tcp(127.0.0.1:3306)/demo?charset=utf8mb4&parseTime=True&loc=Local"
  sqlserver:
    type: sqlserver
    dsn: "sqlserver://sa:Password@127.0.0.1:1433?database=demo"
  sqlite:
    type: sqlite
    dsn: "file:demo.db?cache=shared&_fk=1"

mongo:
  default:
    dsn: "mongodb://127.0.0.1:27017"
    maxPoolSize: 100

grpc:
  default:
    addr: 127.0.0.1:9092
```

## 日志

```go
import (
	"context"
	"time"

	"github.com/haunbo/go-pilot/pkg/xlog"
)

logger := xlog.Init(xlog.RawConfig("logger.default"))
defer logger.Sync()

logger.Info("server started", xlog.String("addr", ":8080"))
logger.Error("request failed", xlog.FieldErr(err), xlog.FieldCost(12*time.Millisecond))

ctx := xlog.WithRequestID(context.Background(), "req-123")
logger.WithContext(ctx).Info("request handled") // 自动带上 request_id
```

`xlog.Config` 支持文件输出、`maxSize`/`maxAge`/`maxBackup` 轮转。

级别可以运行期调整：`logger.SetLevel("debug")`（子 logger 共用同一个原子级别，一处改全局生效），或 `xlog.Reload("logger.default")` 从配置热切；其余字段在下次重启生效。

## HTTP 服务

```go
import (
	"context"
	"net/http"

	"github.com/haunbo/go-pilot/pkg/server/xecho"
	"github.com/labstack/echo/v4"
)

server, err := xecho.StdConfig("http").Build()
if err != nil {
	panic(err)
}

server.GET("/healthz", func(c echo.Context) error {
	return xecho.Success(c, map[string]string{"status": "ok"})
})
server.GET("/users/:id", func(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"id": c.Param("id")})
})

go func() { _ = server.Serve() }()

_ = server.GracefulStop(context.Background())
```

默认监听 `:8080`。内置请求 ID、访问日志、panic 恢复、慢请求标记、CORS 和 BodyLimit。参数校验默认英文，配置 `validationLocale: zh` 可输出中文提示。

`server.Reload()` 会重新读取 `server.http` 并热更新访问日志开关、慢请求阈值、校验语言和 `serviceAddress`；端口、TLS、超时、CORS 与 BodyLimit 属于构建期参数，只会在报告的 `Restart` 里列出。

## gRPC

### 服务端

```go
import (
	"context"

	"github.com/haunbo/go-pilot/pkg/server/xgrpc"
)

server, err := xgrpc.StdConfig("grpc").Build()
if err != nil {
	panic(err)
}
// pb.RegisterYourServiceServer(server.Server, &impl{})
go func() { _ = server.Serve() }()

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
_ = server.GracefulStop(ctx)
```

支持访问日志、panic 恢复、request_id 透传、最大消息大小和 TLS/mTLS：

```yaml
server:
  grpc:
    host: 0.0.0.0
    port: 9092
    enableAccessLog: true
    slowQueryThresholdInMilli: 500
    maxRecvMsgSize: 4194304
    maxSendMsgSize: 4194304
    enableTLS: false
    certFile: server.pem
    privateFile: server.key
    caFile: ca.pem
```

### 客户端

```go
import (
	"context"

	grpcclient "github.com/haunbo/go-pilot/pkg/client/grpc"
)

cfg := grpcclient.DefaultConfig()
cfg.Addr = "127.0.0.1:9092"
cfg.DialTimeout = 3 * time.Second // 0 表示懒连接
conn, err := cfg.Build()
if err != nil {
	panic(err)
}
defer conn.Close()

// pb.NewYourServiceClient(conn)
ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()
```

客户端支持轮询负载均衡（默认 `round_robin`）、访问日志、慢调用日志、request_id 透传、TLS 和自定义 DialOption。服务发现不在本库范围内，可通过 `WithDialOption(grpc.WithResolvers(...))` 接入任意 resolver。

## Redis

```go
import (
	"context"
	"time"

	"github.com/haunbo/go-pilot/pkg/client/redis"
)

client, err := redis.StdConfig("default").Build()
if err != nil {
	panic(err)
}
defer client.Close()

if err := client.RDB().Set(context.Background(), "demo:key", "value", time.Minute).Err(); err != nil {
	panic(err)
}
```

## 关系型数据库

```go
import "github.com/haunbo/go-pilot/pkg/store/xrdb"

db, err := xrdb.StdConfig("mysql").Build()
if err != nil {
	panic(err)
}
defer db.Close()

if err := db.AutoMigrate(&User{}); err != nil {
	panic(err)
}
err = db.Gorm().WithContext(ctx).Find(&users).Error
```

支持的类型：

| type | 驱动 | DSN 示例 |
| --- | --- | --- |
| `mysql` | `gorm.io/driver/mysql` | `root:root@tcp(127.0.0.1:3306)/demo?charset=utf8mb4&parseTime=True&loc=Local` |
| `postgres` | `gorm.io/driver/postgres` | `host=127.0.0.1 user=postgres password=postgres dbname=demo port=5432 sslmode=disable` |
| `sqlserver` | `gorm.io/driver/sqlserver` | `sqlserver://sa:Password@127.0.0.1:1433?database=demo` |
| `sqlite` | `github.com/glebarez/sqlite`（纯 Go） | `file:demo.db?cache=shared&_fk=1` |

`sqlite` 使用纯 Go 驱动，不依赖 CGO，适合本地开发和轻量部署。

## MongoDB

```go
import (
	"context"

	"github.com/haunbo/go-pilot/pkg/store/xmongo"
)

client, err := xmongo.StdConfig("default").Build()
if err != nil {
	panic(err)
}
defer client.Close(context.Background())

users := client.Collection("demo", "users")
_, err = users.InsertOne(context.Background(), map[string]any{"name": "alice"})
```

使用官方 `go.mongodb.org/mongo-driver/v2`，支持最小/最大连接池、连接超时和服务选择超时。

## HTTP 客户端

```go
import (
	"context"
	"time"

	"github.com/haunbo/go-pilot/pkg/client/request"
)

cfg := request.DefaultConfig()
cfg.BaseURL = "https://api.example.com"
cfg.Timeout = 3 * time.Second
cfg.RetryCount = 2
cfg.Auth = request.AuthConfig{Type: "bearer", Token: "token"}
client := cfg.MustBuild()

var out struct {
	ID int `json:"id"`
}
resp, err := client.Get(context.Background(), "/api/v1/users/1", &out)
_ = resp
```

GET/HEAD/PUT/DELETE/OPTIONS 默认会在网络错误、429、5xx 时重试；POST/PATCH 默认不重试，避免重复写入。`RetryNonIdempotent: true` 可强制开启。

## Excel

```go
import "github.com/haunbo/go-pilot/pkg/excel"

err := excel.Save("users.xlsx", "users",
	[]string{"ID", "Name"},
	[][]any{{1, "alice"}, {2, "bob"}},
)
```

## 配置约定

- 配置使用点号路径：`server.http.port`、`logger.default.level`。
- `RawConfig(key)` / `StdConfig(name)` 会从全局 `conf` 读取对应节点，缺失时使用包内默认值。
- **所有时间字段必须写成带单位的字符串**（`15s`、`200ms`、`1h`）。解码器带 `WeaklyTypedInput`，写成裸数字 `timeout: 500` 会被当成 **500 纳秒**且不报任何错误。
- **`xlog.Init` 必须在任何 `StdConfig()` / `DefaultConfig()` / `Build()` 之前调用**。各组件的 `DefaultConfig()` 会在构造时就从当前默认 logger 派生子 logger，晚于构造再 `Init`，组件会继续写旧 logger（表现为日志写不到指定文件）。
- 环境变量前缀由调用方决定，推荐 `GOPILOT_`；`LoadEnvironments` 会把前缀后的下划线转成点号。
- 结构体默认按 `json` tag 解码，也可以使用 `conf.UnmarshalKey(key, &out, conf.TagNameMapstructure())`。
- 文件配置源和 Nacos 配置源都实现了 `conf.DataSource`，可以按需替换。
- 热更新：`conf.OnChange` 里手动调用 `xlog.Reload("logger.default")` 与 `server.Reload()`，按返回的 `ReloadReport` 打日志。`xlog` 支持热切 `level`，`xecho` 支持热切访问日志开关/慢请求阈值/校验语言；端口、TLS、超时、连接池等构建期参数只能重启。详见「热更新边界」。

## 设计说明

- 只包含通用 `pkg`，不包含任何用户/角色/菜单/登录等业务逻辑。
- 配置源保留 **file** 和 **nacos**；数据库保留 **MySQL/PostgreSQL/SQLServer/SQLite/MongoDB**；gRPC 服务端和客户端均保留。
- 不引入 etcd 服务发现、服务注册、OpenTelemetry、SSE、gRPC gateway 等额外能力，保持公共库精简。
- `pkg/util` 只保留被本库自身引用的三个小工具：`xcast.ToInt64Slice`（`conf.GetInt64Slice`）、`xcrypto.Bytes`、`xstring.GenerateUUID`（请求 ID）。字符串、文件、重试、泛型数学等通用需求请直接用标准库或成熟第三方库，本库不扩张 util。
- 不提供示例工程与组装层：进程怎么启动、怎么组装组件属于业务项目的选择，本库只提供可独立引用的组件。
- 所有包均可单独引用，不需要一次性初始化全部组件。
- `pkg/xerr` 只服务 HTTP 场景（业务码 + HTTP 状态码）。gRPC 服务端/客户端的错误码映射不做统一抽象，由各业务模块按自己的 code 规范自行实现。
- 代码使用 Go 1.27 写法：以 `any` 替代 `interface{}`，泛型、`errors.Join`、`context` 取消等。

## 开发

```bash
go test ./...                 # 单元测试
go test -race ./...           # 竞态检测
go vet ./...                  # 静态检查
go build ./...                # 编译全部包
CGO_ENABLED=0 go build ./...  # 无 CGO 构建（SQLite 使用纯 Go 驱动）
make test                     # 等价命令
```

本仓库不配置 GitHub Actions / 其他 CI，发布前请在本地跑完上面四条命令。

## 版本

当前版本 `v1.0.0`，见 [CHANGELOG.md](CHANGELOG.md)。发布新版本时使用语义化 tag：

```bash
git tag v1.0.1
git push origin v1.0.1
```

## 推送到 GitHub / Gitee

先在 GitHub 和 Gitee 各创建一个**空仓库**（不要初始化 README、License 或 .gitignore），然后执行：

```bash
cd /Users/haunbo/project/golang/go-pilot

# GitHub
git remote add origin git@github.com:haunbo/go-pilot.git
git push -u origin main
git push origin v1.0.0

# Gitee
git remote add gitee git@gitee.com:haunbo/go-pilot.git
git push -u gitee main
git push gitee v1.0.0
```

使用 HTTPS 时，把 SSH 地址换成：

```bash
git remote add origin https://github.com/haunbo/go-pilot.git
git remote add gitee https://gitee.com/haunbo/go-pilot.git
# 用户名填账号，密码填 Personal Access Token / Gitee 私人令牌
```

## License

[MIT](LICENSE)
