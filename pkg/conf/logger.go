package conf

import (
	"log"
	"sync"
)

// LogFunc writes a formatted diagnostic message.
type LogFunc func(format string, args ...any)

var (
	logMu sync.RWMutex
	logf  LogFunc = log.Printf
)

// SetLogFunc replaces the logger used for internal diagnostics: reload failures,
// reload success messages and LoadEnvironments errors.
//
// conf cannot import xlog (xlog depends on conf), so this hook is how a program
// routes configuration messages into its own logging pipeline:
//
//	conf.SetLogFunc(func(format string, args ...any) { xlog.Infof(format, args...) })
//
// Call it after the logger is initialized; passing nil restores the standard
// library logger.
func SetLogFunc(fn LogFunc) {
	logMu.Lock()
	if fn == nil {
		logf = log.Printf
	} else {
		logf = fn
	}
	logMu.Unlock()
}

func logMessage(format string, args ...any) {
	logMu.RLock()
	fn := logf
	logMu.RUnlock()
	fn(format, args...)
}
