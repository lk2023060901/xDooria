package auth

import (
	"context"

	pb "github.com/lk2023060901/xdooria-proto-common"
)

// Identity 认证后的统一身份标识
type Identity struct {
	UID        string            // 平台唯一标识 (OpenID/UnionID)
	Nickname   string            // 平台昵称
	Avatar     string            // 平台头像
	Extra      map[string]string // 平台特定扩展数据
}

// Authenticator 认证器接口，每种平台实现一套
type Authenticator interface {
	// Type 获取认证器类型
	Type() pb.LoginType
	
	// Authenticate 执行认证逻辑
	// cred: 原始凭证 (通常是 SDK 拿到的 Token 或 Code)
	Authenticate(ctx context.Context, cred []byte) (*Identity, error)
}
