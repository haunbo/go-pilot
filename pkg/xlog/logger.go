package xlog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

var defaultLogger = New(DefaultConfig())

// Logger wraps zap.Logger with context aware helpers.
type Logger struct {
	ctx  context.Context
	zlog *zap.Logger
	// level is the atomic level shared by the logger and all its children, which
	// allows SetLevel to take effect without rebuilding the zap core.
	level zap.AtomicLevel
	// cfg is the snapshot this logger was built from, kept so Reload can report
	// which settings actually changed.
	cfg *Config
}

// New builds a Logger from cfg. A nil cfg uses DefaultConfig.
func New(cfg *Config) *Logger {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	snapshot := *cfg
	level := zap.NewAtomicLevelAt(parseLevel(cfg.Level))
	core := zapcore.NewCore(newEncoder(cfg), zapcore.AddSync(newWriter(cfg)), level)
	opts := []zap.Option{
		zap.AddCaller(),
		zap.AddCallerSkip(1 + cfg.CallerSkip),
	}
	return &Logger{zlog: zap.New(core, opts...), level: level, cfg: &snapshot}
}

// SetDefault replaces the process-wide logger.
func SetDefault(l *Logger) {
	if l != nil {
		defaultLogger = l
	}
}

// Default returns the process-wide logger.
func Default() *Logger { return defaultLogger }

// Init builds cfg, installs it as default and returns it.
func Init(cfg *Config) *Logger {
	l := New(cfg)
	SetDefault(l)
	return l
}

func newWriter(cfg *Config) zapcore.WriteSyncer {
	if cfg.Output != nil {
		return zapcore.AddSync(cfg.Output)
	}
	if cfg.Dir == "" || cfg.Dir == "-" {
		return zapcore.AddSync(os.Stdout)
	}
	_ = os.MkdirAll(cfg.Dir, 0o755)
	return zapcore.AddSync(&lumberjack.Logger{
		Filename:   cfg.Filename(),
		Compress:   cfg.Compress,
		MaxBackups: cfg.MaxBackup,
		MaxSize:    cfg.MaxSize,
		MaxAge:     cfg.MaxAge,
		LocalTime:  true,
	})
}

func newEncoder(cfg *Config) zapcore.Encoder {
	encCfg := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "line",
		MessageKey:     "detail",
		StacktraceKey:  "stack",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}
	if strings.EqualFold(cfg.Format, FormatText) {
		return zapcore.NewConsoleEncoder(encCfg)
	}
	return zapcore.NewJSONEncoder(encCfg)
}

func parseLevel(level string) zapcore.Level {
	lvl, err := parseLevelStrict(level)
	if err != nil {
		return zap.InfoLevel
	}
	return lvl
}

// parseLevelStrict is parseLevel for runtime reload: an unknown level is an
// error instead of a silent fallback to info.
func parseLevelStrict(level string) (zapcore.Level, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return zap.DebugLevel, nil
	case "info":
		return zap.InfoLevel, nil
	case "warn", "warning":
		return zap.WarnLevel, nil
	case "error":
		return zap.ErrorLevel, nil
	case "dpanic":
		return zap.DPanicLevel, nil
	case "panic":
		return zap.PanicLevel, nil
	case "fatal":
		return zap.FatalLevel, nil
	default:
		return zap.InfoLevel, fmt.Errorf("xlog: unknown level %q (want debug/info/warn/error/dpanic/panic/fatal)", level)
	}
}

// Zap returns the underlying zap.Logger.
func (l *Logger) Zap() *zap.Logger { return l.zlog }

// SetLevel changes the level of this logger and of every child derived from it.
// It takes effect immediately and is safe to call while other goroutines log.
func (l *Logger) SetLevel(level string) error {
	if l == nil || l.zlog == nil {
		return errors.New("xlog: logger is not initialized")
	}
	lvl, err := parseLevelStrict(level)
	if err != nil {
		return err
	}
	l.level.SetLevel(lvl)
	return nil
}

// Level returns the current level name, for example "info".
func (l *Logger) Level() string {
	if l == nil || l.zlog == nil {
		return ""
	}
	return l.level.Level().String()
}

// SetLevel changes the level of the default logger.
func SetLevel(level string) error { return defaultLogger.SetLevel(level) }

// With returns a child logger with extra fields.
func (l *Logger) With(fields ...Field) *Logger {
	child := *l
	child.zlog = l.zlog.With(fields...)
	return &child
}

// WithContext returns a child logger carrying ctx.
func (l *Logger) WithContext(ctx context.Context, fields ...Field) *Logger {
	child := l.With(fields...)
	child.ctx = ctx
	return child
}

func (l *Logger) buildFields(fields []Field) []Field {
	if l.ctx == nil {
		return fields
	}
	if id := RequestIDFromContext(l.ctx); id != "" {
		fields = append(fields, String("request_id", id))
	}
	return fields
}

// Debug logs at debug level.
func (l *Logger) Debug(msg string, fields ...Field) { l.zlog.Debug(msg, l.buildFields(fields)...) }

// Debugf logs a formatted debug message.
func (l *Logger) Debugf(msg string, args ...any) {
	l.zlog.Debug(fmt.Sprintf(msg, args...), l.buildFields(nil)...)
}

// Info logs at info level.
func (l *Logger) Info(msg string, fields ...Field) { l.zlog.Info(msg, l.buildFields(fields)...) }

// Infof logs a formatted info message.
func (l *Logger) Infof(msg string, args ...any) {
	l.zlog.Info(fmt.Sprintf(msg, args...), l.buildFields(nil)...)
}

// Warn logs at warn level.
func (l *Logger) Warn(msg string, fields ...Field) { l.zlog.Warn(msg, l.buildFields(fields)...) }

// Warnf logs a formatted warn message.
func (l *Logger) Warnf(msg string, args ...any) {
	l.zlog.Warn(fmt.Sprintf(msg, args...), l.buildFields(nil)...)
}

// Error logs at error level.
func (l *Logger) Error(msg string, fields ...Field) { l.zlog.Error(msg, l.buildFields(fields)...) }

// Errorf logs a formatted error message.
func (l *Logger) Errorf(msg string, args ...any) {
	l.zlog.Error(fmt.Sprintf(msg, args...), l.buildFields(nil)...)
}

// Panic logs at panic level then panics.
func (l *Logger) Panic(msg string, fields ...Field) { l.zlog.Panic(msg, l.buildFields(fields)...) }

// Fatal logs at fatal level then exits.
func (l *Logger) Fatal(msg string, fields ...Field) { l.zlog.Fatal(msg, l.buildFields(fields)...) }

// Sync flushes buffered logs. It never returns an error.
func (l *Logger) Sync() {
	if l == nil || l.zlog == nil {
		return
	}
	_ = l.zlog.Sync()
}

// With returns a child of the default logger.
func With(fields ...Field) *Logger { return defaultLogger.With(fields...) }

// WithContext returns a child of the default logger carrying ctx.
func WithContext(ctx context.Context, fields ...Field) *Logger {
	return defaultLogger.WithContext(ctx, fields...)
}

// Debug logs with the default logger.
func Debug(msg string, fields ...Field) { defaultLogger.Debug(msg, fields...) }

// Debugf logs with the default logger.
func Debugf(msg string, args ...any) { defaultLogger.Debugf(msg, args...) }

// Info logs with the default logger.
func Info(msg string, fields ...Field) { defaultLogger.Info(msg, fields...) }

// Infof logs with the default logger.
func Infof(msg string, args ...any) { defaultLogger.Infof(msg, args...) }

// Warn logs with the default logger.
func Warn(msg string, fields ...Field) { defaultLogger.Warn(msg, fields...) }

// Warnf logs with the default logger.
func Warnf(msg string, args ...any) { defaultLogger.Warnf(msg, args...) }

// Error logs with the default logger.
func Error(msg string, fields ...Field) { defaultLogger.Error(msg, fields...) }

// Errorf logs with the default logger.
func Errorf(msg string, args ...any) { defaultLogger.Errorf(msg, args...) }

// Panic logs with the default logger then panics.
func Panic(msg string, fields ...Field) { defaultLogger.Panic(msg, fields...) }

// Fatal logs with the default logger then exits.
func Fatal(msg string, fields ...Field) { defaultLogger.Fatal(msg, fields...) }

// Sync flushes the default logger.
func Sync() { defaultLogger.Sync() }
