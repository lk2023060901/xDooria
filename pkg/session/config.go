package session

import (
	"errors"
)

// Config 会话配置。
type Config struct {
	// SendChannelSize 发送队列大小。
	SendChannelSize int `mapstructure:"send_channel_size" json:"send_channel_size" yaml:"send_channel_size"`
	// RecvChannelSize 接收队列大小。
	RecvChannelSize int `mapstructure:"recv_channel_size" json:"recv_channel_size" yaml:"recv_channel_size"`
}

// ServerConfig 服务端配置。
type ServerConfig struct {
	// Session 会话相关参数配置。
	Session *Config `mapstructure:"session" json:"session" yaml:"session"`

	// 以下为运行时组件，由代码注入
	Acceptor Acceptor       `json:"-" yaml:"-"`
	Handler  SessionHandler `json:"-" yaml:"-"`
	Manager  SessionManager `json:"-" yaml:"-"`
}

// DefaultConfig 返回默认会话参数。
func DefaultConfig() *Config {
	return &Config{
		SendChannelSize: 1024,
		RecvChannelSize: 1024,
	}
}

// DefaultServerConfig 返回包含默认组件的服务端配置。
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Session: DefaultConfig(),
		Handler: &NopSessionHandler{},
		Manager: NewBaseSessionManager(),
	}
}

// Validate 验证配置。
func (c *Config) Validate() error {
	if c.SendChannelSize <= 0 {
		return errors.New("send_channel_size must be greater than 0")
	}
	if c.RecvChannelSize <= 0 {
		return errors.New("recv_channel_size must be greater than 0")
	}
	return nil
}

// Validate 验证服务端配置。
func (c *ServerConfig) Validate() error {
	if c.Session == nil {
		c.Session = DefaultConfig()
	}
	return c.Session.Validate()
}
