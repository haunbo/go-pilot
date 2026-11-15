# go-pilot

一个面向 Go 后端服务的通用工具库，从内部 `pilot` 工具集中提炼而来。只包含可复用的 `pkg` 能力，不包含任何后台管理业务逻辑。

- Module: `github.com/haunbo/go-pilot`
- Version: `v1.0.0`
- Go: `1.27+`
- 代码风格：Go 1.27 新写法（`any`、泛型、`errors.Join`、context 等）

## 包一览

| 包 | 用途 |
| --- | --- |
| `pkg/conf` | 配置核心：YAML/JSON、点号路径、结构体解码、环境变量覆盖 |
| `pkg/conf/datasource/file` | 文件配置源，支持 fsnotify 热更新 |
| `pkg/conf/datasource/nacos` | Nacos 配置源，支持监听配置变更 |
| `pkg/xlog` | zap 结构化日志、文件轮转、request_id 注入 |
| `pkg/xerr` | 通用业务码 + HTTP 状态码错误类型 |
| `pkg/server/xecho` | Echo HTTP 服务：访问日志、panic 恢复、请求 ID、CORS、BodyLimit、中英文校验、统一响应 |
| `pkg/server/xgrpc` | gRPC 服务端：访问日志、panic 恢复、request_id、TLS/mTLS |
| `pkg/client/grpc` | gRPC 客户端：超时、访问日志、request_id、TLS、连接就绪等待 |
| `pkg/client/redis` | go-redis 连接池、启动 PING、访问日志 |
| `pkg/client/request` | 基于 net/http 的 JSON 客户端：重试、鉴权、慢请求日志、request_id 透传 |
| `pkg/store/xrdb` | GORM：MySQL/PostgreSQL/SQLServer/SQLite、连接池、慢 SQL、AutoMigrate |
| `pkg/store/xmongo` | 官方 MongoDB Driver v2 封装、连接池、启动 PING |
| `pkg/excel` | 基于 excelize 的 Excel 导出 |
| `pkg/util/xcast` | 类型转换辅助 |
| `pkg/util/xcrypto` | 随机数/随机字符串、MD5、SHA-256、HMAC-SHA256 |
| `pkg/util/xfile` | 原子写入、复制、哈希、目录遍历、跨进程文件锁 |
| `pkg/util/xgo` | goroutine、panic 恢复、并行/串行执行 |
| `pkg/util/xmap` | 嵌套 map 合并、扁平化、反扁平化、大小写不敏感处理 |
| `pkg/util/xmath` | 泛型 Abs/Clamp/Percent |
| `pkg/util/xnet` | 本机 IP/MAC、监听端口、TCP/HTTP 健康检查 |
| `pkg/util/xretry` | 支持 context 的指数退避重试 |
| `pkg/util/xstring` | UUID、截断/拼接、函数名、反转、地址转换等 |
| `pkg/util/xtime` | 相对时间、当天起止时间 |
| `pkg/version` | 版本号与构建信息 |

## 安装

```bash
go get github.com/haunbo/go-pilot@v1.0.0
```

要求 Go 1.27 或更高版本。项目遵循语义化版本，`v1.x` 内保持 API 兼容。

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
		log.Print("config reloaded")
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

## 工具包示例

```go
import (
	"github.com/haunbo/go-pilot/pkg/util/xcrypto"
	"github.com/haunbo/go-pilot/pkg/util/xfile"
	"github.com/haunbo/go-pilot/pkg/util/xgo"
	"github.com/haunbo/go-pilot/pkg/util/xretry"
)

id := xcrypto.RandomString(24)
_ = xfile.WriteFileAtomic("data/id.txt", []byte(id), 0o644)
xgo.Parallel(taskA, taskB)
_ = xretry.Do(ctx, 3, 100*time.Millisecond, func(ctx context.Context) error {
	return callRemote(ctx)
})
```

## 配置约定

- 配置使用点号路径：`server.http.port`、`logger.default.level`。
- `RawConfig(key)` / `StdConfig(name)` 会从全局 `conf` 读取对应节点，缺失时使用包内默认值。
- 环境变量前缀由调用方决定，推荐 `GOPILOT_`；`LoadEnvironments` 会把前缀后的下划线转成点号。
- 结构体默认按 `json` tag 解码，也可以使用 `conf.UnmarshalKey(key, &out, conf.TagNameMapstructure())`。
- 文件配置源和 Nacos 配置源都实现了 `conf.DataSource`，可以按需替换。

## 设计说明

- 只包含通用 `pkg`，不包含任何用户/角色/菜单/登录等业务逻辑。
- 配置源保留 **file** 和 **nacos**；数据库保留 **MySQL/PostgreSQL/SQLServer/SQLite/MongoDB**；gRPC 服务端和客户端均保留。
- 不引入 etcd 服务发现、服务注册、OpenTelemetry、SSE、gRPC gateway 等额外能力，保持公共库精简。
- 所有包均可单独引用，不需要一次性初始化全部组件。
- 代码使用 Go 1.27 写法：以 `any` 替代 `interface{}`，泛型、`errors.Join`、context 取消等。

## 开发

```bash
go test ./...                 # 单元测试
go test -race ./...           # 竞态检测
go vet ./...                  # 静态检查
go build ./...                # 编译全部包
CGO_ENABLED=0 go build ./...  # 无 CGO 构建（SQLite 使用纯 Go 驱动）
make test                     # 等价命令
```

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
