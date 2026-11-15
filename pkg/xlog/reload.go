package xlog

import (
	"errors"
	"fmt"
	"strings"

	"github.com/haunbo/go-pilot/pkg/conf"
)

// ConfigKey returns the dotted config key this logger was built from, or "" when
// it was created by hand.
func (l *Logger) ConfigKey() string {
	if l == nil || l.cfg == nil {
		return ""
	}
	return l.cfg.key
}

// Reload re-reads the config key the logger was built from. Call it from a
// conf.OnChange hook to hot-swap the level of a logger created with StdConfig or
// RawConfig; loggers created by hand have no key and report an error.
func (l *Logger) Reload() (conf.ReloadReport, error) {
	var report conf.ReloadReport
	key := l.ConfigKey()
	if key == "" {
		return report, errors.New("xlog: logger was built without a config key, nothing to reload")
	}
	return Reload(key)
}

// Reload re-reads the logger config at key and applies the settings that can
// change at runtime.
//
// Only level is hot-swappable. dir, name, format, rotation, callerSkip and the
// output writer are baked into the zap core, so a change to them is reported in
// ReloadReport.Restart and only takes effect after the process restarts.
func Reload(key string) (conf.ReloadReport, error) {
	var report conf.ReloadReport
	if key == "" {
		return report, errors.New("xlog: empty config key")
	}
	logger := Default()
	if logger == nil || logger.cfg == nil {
		return report, errors.New("xlog: logger is not initialized")
	}
	next := DefaultConfig()
	next.key = key
	if err := conf.UnmarshalKey(key, next); err != nil {
		if errors.Is(err, conf.ErrInvalidKey) {
			return report, nil // 没有配置该节点，保持当前设置
		}
		return report, fmt.Errorf("xlog: read %s: %w", key, err)
	}
	return logger.ApplyConfig(next)
}

// ApplyConfig applies an already decoded config to the logger and reports which
// settings were applied and which need a restart.
func (l *Logger) ApplyConfig(next *Config) (conf.ReloadReport, error) {
	var report conf.ReloadReport
	if l == nil || l.cfg == nil {
		return report, errors.New("xlog: logger is not initialized")
	}
	if next == nil {
		return report, errors.New("xlog: nil config")
	}
	old := l.cfg

	if !strings.EqualFold(strings.TrimSpace(next.Level), strings.TrimSpace(old.Level)) {
		if err := l.SetLevel(next.Level); err != nil {
			return report, err
		}
		report.MarkApplied("level")
	}
	if next.Dir != old.Dir || next.Name != old.Name {
		report.MarkRestartNeeded("dir", "name")
	}
	if next.Format != old.Format {
		report.MarkRestartNeeded("format")
	}
	if next.MaxSize != old.MaxSize || next.MaxAge != old.MaxAge ||
		next.MaxBackup != old.MaxBackup || next.Compress != old.Compress {
		report.MarkRestartNeeded("rotation")
	}
	if next.CallerSkip != old.CallerSkip {
		report.MarkRestartNeeded("callerSkip")
	}
	if next.Output != nil && next.Output != old.Output {
		report.MarkRestartNeeded("output")
	}

	snapshot := *next
	snapshot.Output = old.Output // 输出目标未重建，保持与 core 一致
	l.cfg = &snapshot
	return report, nil
}
