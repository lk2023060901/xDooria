// Package session 提供通用的会话管理框架，使用 UUID string 作为会话标识。
// 业务服务可以通过实现 Session 接口或嵌入 BaseSession 来扩展。
package session

// Session 定义会话的基础接口。
type Session interface {
	// ID 返回会话的唯一标识（UUID string）。
	ID() string
}

// BaseSession 提供 Session 接口的基础实现。
type BaseSession struct {
	id string
}

// NewBaseSession 创建一个新的基础会话。
func NewBaseSession(id string) *BaseSession {
	return &BaseSession{id: id}
}

// ID 返回会话 ID。
func (s *BaseSession) ID() string {
	return s.id
}
