// Package nacos provides a Nacos backed configuration source.
package nacos

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/haunbo/go-pilot/pkg/conf"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// Scheme is the configuration scheme name.
const Scheme = "nacos"

// Config controls the Nacos config client.
type Config struct {
	ServerAddr           string `json:"serverAddr"`
	Port                 uint64 `json:"port"`
	ContextPath          string `json:"contextPath"`
	NamespaceID          string `json:"namespaceId"`
	Group                string `json:"group"`
	DataID               string `json:"dataId"`
	Username             string `json:"username"`
	Password             string `json:"password"`
	TimeoutMs            uint64 `json:"timeoutMs"`
	CacheDir             string `json:"cacheDir"`
	LogDir               string `json:"logDir"`
	LogLevel             string `json:"logLevel"`
	NotLoadCacheAtStart  bool   `json:"notLoadCacheAtStart"`
	UpdateCacheWhenEmpty bool   `json:"updateCacheWhenEmpty"`
}

// DefaultConfig returns Nacos defaults.
func DefaultConfig() Config {
	return Config{
		Port:                 8848,
		Group:                "DEFAULT_GROUP",
		TimeoutMs:            10000,
		LogLevel:             "info",
		NotLoadCacheAtStart:  true,
		UpdateCacheWhenEmpty: true,
	}
}

func (c Config) normalize() Config {
	defaults := DefaultConfig()
	if c.Port == 0 {
		c.Port = defaults.Port
	}
	if c.Group == "" {
		c.Group = defaults.Group
	}
	if c.TimeoutMs == 0 {
		c.TimeoutMs = defaults.TimeoutMs
	}
	if c.LogLevel == "" {
		c.LogLevel = defaults.LogLevel
	}
	return c
}

// Source is a Nacos configuration data source.
type Source struct {
	client    config_client.IConfigClient
	config    Config
	changed   chan struct{}
	done      chan struct{}
	closeOnce sync.Once
}

// New creates a Nacos data source. When watch is true the data source listens
// for server-side config changes.
func New(cfg Config, watch bool) (*Source, error) {
	cfg = cfg.normalize()
	cfg.ServerAddr = strings.TrimSpace(cfg.ServerAddr)
	cfg.DataID = strings.TrimSpace(cfg.DataID)
	cfg.Group = strings.TrimSpace(cfg.Group)
	if cfg.ServerAddr == "" {
		return nil, errors.New("nacos: serverAddr is required")
	}
	if cfg.DataID == "" {
		return nil, errors.New("nacos: dataID is required")
	}
	if cfg.Group == "" {
		return nil, errors.New("nacos: group is required")
	}

	clientConfig := constant.ClientConfig{
		TimeoutMs:            cfg.TimeoutMs,
		NamespaceId:          cfg.NamespaceID,
		Username:             cfg.Username,
		Password:             cfg.Password,
		CacheDir:             cfg.CacheDir,
		LogDir:               cfg.LogDir,
		LogLevel:             cfg.LogLevel,
		NotLoadCacheAtStart:  cfg.NotLoadCacheAtStart,
		UpdateCacheWhenEmpty: cfg.UpdateCacheWhenEmpty,
	}
	serverConfigs := []constant.ServerConfig{{
		IpAddr:      cfg.ServerAddr,
		Port:        cfg.Port,
		ContextPath: cfg.ContextPath,
	}}
	client, err := clients.NewConfigClient(vo.NacosClientParam{
		ClientConfig:  &clientConfig,
		ServerConfigs: serverConfigs,
	})
	if err != nil {
		return nil, fmt.Errorf("nacos: create config client: %w", err)
	}

	source := &Source{
		client: client,
		config: cfg,
		done:   make(chan struct{}),
	}
	if watch {
		source.changed = make(chan struct{}, 1)
		if err := client.ListenConfig(vo.ConfigParam{
			DataId: cfg.DataID,
			Group:  cfg.Group,
			OnChange: func(_, _, _, _ string) {
				select {
				case <-source.done:
				default:
					select {
					case source.changed <- struct{}{}:
					default:
					}
				}
			},
		}); err != nil {
			client.CloseClient()
			return nil, fmt.Errorf("nacos: listen config: %w", err)
		}
	}
	return source, nil
}

// ReadConfig fetches the current config content.
func (s *Source) ReadConfig() ([]byte, error) {
	content, err := s.client.GetConfig(vo.ConfigParam{
		DataId: s.config.DataID,
		Group:  s.config.Group,
	})
	if err != nil {
		return nil, fmt.Errorf("nacos: get config: %w", err)
	}
	return []byte(content), nil
}

// IsConfigChanged returns the change channel, or nil when watch was disabled.
func (s *Source) IsConfigChanged() <-chan struct{} {
	return s.changed
}

// Close stops listening and closes the Nacos client.
func (s *Source) Close() error {
	var errs []error
	s.closeOnce.Do(func() {
		close(s.done)
		if s.changed != nil {
			if err := s.client.CancelListenConfig(vo.ConfigParam{
				DataId: s.config.DataID,
				Group:  s.config.Group,
			}); err != nil {
				errs = append(errs, err)
			}
		}
		s.client.CloseClient()
	})
	return errors.Join(errs...)
}

var _ conf.DataSource = (*Source)(nil)
