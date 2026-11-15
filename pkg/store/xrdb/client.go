package xrdb

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// ErrRecordNotFound is returned by GORM when a row does not exist.
var ErrRecordNotFound = gorm.ErrRecordNotFound

// Client embeds *gorm.DB and owns the underlying sql.DB.
type Client struct {
	*gorm.DB
	config *Config
}

func newClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("xrdb: nil config")
	}
	if cfg.DSN == "" {
		return nil, fmt.Errorf("xrdb: empty dsn for %q", cfg.Name)
	}

	var dialector gorm.Dialector
	switch cfg.Type {
	case MySQL:
		dialector = mysql.Open(cfg.DSN)
	case Postgres:
		dialector = postgres.Open(cfg.DSN)
	case SQLServer:
		dialector = sqlserver.Open(cfg.DSN)
	case SQLite:
		dialector = sqlite.Open(cfg.DSN)
	default:
		return nil, fmt.Errorf("xrdb: unsupported database type %q", cfg.Type)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger:                                   newGormLogger(cfg),
		DisableForeignKeyConstraintWhenMigrating: true,
		NowFunc:                                  func() time.Time { return time.Now().Local() },
	})
	if err != nil {
		return nil, fmt.Errorf("xrdb: open %s: %w", cfg.Name, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("xrdb: get sql db: %w", err)
	}
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("xrdb: ping %s: %w", cfg.Name, err)
	}

	return &Client{DB: db, config: cfg}, nil
}

func newGormLogger(cfg *Config) gormlogger.Interface {
	level := gormlogger.Warn
	switch strings.ToLower(cfg.LogLevel) {
	case "silent":
		level = gormlogger.Silent
	case "error":
		level = gormlogger.Error
	case "info":
		level = gormlogger.Info
	case "warn", "warning", "":
		level = gormlogger.Warn
	}
	if cfg.Debug {
		level = gormlogger.Info
	}
	slow := cfg.SlowThreshold
	if slow <= 0 {
		slow = 200 * time.Millisecond
	}
	return gormlogger.New(log.New(os.Stdout, "", log.LstdFlags), gormlogger.Config{
		SlowThreshold:             slow,
		LogLevel:                  level,
		IgnoreRecordNotFoundError: true,
		Colorful:                  false,
	})
}

// Gorm returns the underlying *gorm.DB.
func (c *Client) Gorm() *gorm.DB { return c.DB }

// Config returns the database config.
func (c *Client) Config() *Config { return c.config }

// Ping checks database connectivity.
func (c *Client) Ping(ctx context.Context) error {
	sqlDB, err := c.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Close closes the connection pool.
func (c *Client) Close() error {
	sqlDB, err := c.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// AutoMigrate runs GORM auto migration for the given models.
func (c *Client) AutoMigrate(models ...any) error {
	return c.DB.AutoMigrate(models...)
}
