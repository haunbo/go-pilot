package xecho

import (
	"errors"
	"fmt"
	"slices"

	"github.com/haunbo/go-pilot/pkg/conf"
)

// Reload re-reads the config key this server was built from and applies the
// settings that support hot updates.
//
// Call it from a conf.OnChange hook:
//
//	httpServer, _ := xecho.StdConfig("http").Build()
//	conf.OnChange(func(*conf.Configuration) {
//		report, err := httpServer.Reload()
//		...
//	})
//
// The returned report lists what took effect immediately and what still needs a
// restart, because listeners, TLS material, http.Server timeouts, the CORS
// origin set and the body limit middleware are all fixed at build time.
func (s *Server) Reload() (conf.ReloadReport, error) {
	var report conf.ReloadReport
	old := s.Config()
	if old == nil || old.key == "" {
		return report, errors.New("xecho: server has no config key, build it with StdConfig/RawConfig or call ApplyConfig directly")
	}
	next := DefaultConfig()
	next.key = old.key
	if err := conf.UnmarshalKey(old.key, next); err != nil && !errors.Is(err, conf.ErrInvalidKey) {
		// 配置写错时必须保留旧配置继续服务，而不是退回默认值。
		return report, fmt.Errorf("xecho: read %s: %w", old.key, err)
	}
	return s.ApplyConfig(next), nil
}

// ApplyConfig stores next as the live configuration and reports the outcome.
func (s *Server) ApplyConfig(next *Config) conf.ReloadReport {
	var report conf.ReloadReport
	old := s.Config()
	if next == nil || old == nil {
		return report
	}

	// —— 可热更新：所有读写都发生在请求路径上，读的是当前快照 ——
	if next.EnableAccessLog != old.EnableAccessLog {
		report.MarkApplied("enableAccessLog")
	}
	if next.SlowQueryThresholdInMilli != old.SlowQueryThresholdInMilli {
		report.MarkApplied("slowQueryThresholdInMilli")
	}
	if next.Debug != old.Debug {
		// 只影响启动时的路由打印，这里不改 echo.Debug 避免和 Serve 并发写。
		report.MarkApplied("debug")
	}
	if next.ServiceAddress != old.ServiceAddress {
		report.MarkApplied("serviceAddress")
	}
	if next.ValidationLocale != old.ValidationLocale {
		s.validator.Store(NewValidator(next.ValidationLocale))
		report.MarkApplied("validationLocale")
	}

	// 监听时 port: 0 会被替换成实际端口，重新读取到的 0 不应覆盖它，
	// 也不应被当成"端口变了"。必须在比对之前修正。
	if next.Port == 0 && old.Port != 0 {
		next.Port = old.Port
	}

	// —— 需重启：构建期已固化 ——
	if next.Host != old.Host {
		report.MarkRestartNeeded("host")
	}
	if next.Port != old.Port {
		report.MarkRestartNeeded("port")
	}
	if next.EnableTLS != old.EnableTLS || next.CertFile != old.CertFile || next.PrivateFile != old.PrivateFile {
		report.MarkRestartNeeded("enableTLS", "certFile", "privateFile")
	}
	if next.ReadTimeout != old.ReadTimeout || next.WriteTimeout != old.WriteTimeout || next.IdleTimeout != old.IdleTimeout {
		report.MarkRestartNeeded("readTimeout", "writeTimeout", "idleTimeout")
	}
	if !slices.Equal(next.AllowOrigins, old.AllowOrigins) {
		// CORS 的 allowOrigins 与 allowCredentials 是绑定的（"*" 时不能带凭证），
		// 构建期已编译成正则，运行期改会带来凭证语义漂移，因此归入重启项。
		report.MarkRestartNeeded("allowOrigins")
	}
	if next.BodyLimit != old.BodyLimit {
		report.MarkRestartNeeded("bodyLimit")
	}

	// 监听时 port: 0 会被替换成实际端口，重新读取的 0 不应覆盖它。
	next.key = old.key
	next.logger = old.logger // 日志由 xlog 统一管理，reload 不替换 logger
	s.config.Store(next)
	return report
}
