package gamerouter

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/lk2023060901/xdooria/pkg/router"
)

// RoleRouter 支持 Role 的消息路由器
// 在基础 router 之上，自动从 Context 提取 roleID 并传递给 handler
type RoleRouter struct {
	router router.Router
}

// NewRoleRouter 创建 RoleRouter
func NewRoleRouter(r router.Router) *RoleRouter {
	return &RoleRouter{
		router: r,
	}
}

// RegisterHandler 注册带 roleID 参数的处理器
// Handler 签名: func(ctx context.Context, roleID int64, req PReq) (PResp, error)
// RoleID 会自动从 Context 中提取并传入
func RegisterHandler[TReq any, TResp any, PReq interface {
	*TReq
	proto.Message
}, PResp interface {
	*TResp
	proto.Message
}](
	rr *RoleRouter,
	reqOp uint32,
	respOp uint32,
	handler func(ctx context.Context, roleID int64, req PReq) (PResp, error),
) {
	// 使用底层 router.RegisterHandler，但在调用前从 context 提取 roleID
	router.RegisterHandler(rr.router, reqOp, respOp,
		func(ctx context.Context, req PReq) (PResp, error) {
			// 1. 从 Context 提取 roleID
			roleID, ok := router.GetRoleIDFromContext(ctx)
			if !ok {
				return nil, fmt.Errorf("role_id not found in context")
			}

			// 2. 调用业务 handler（自动传入 roleID）
			return handler(ctx, roleID, req)
		})
}

// Dispatch 路由消息到对应的处理器
func (rr *RoleRouter) Dispatch(ctx context.Context, op uint32, payload []byte) (uint32, []byte, error) {
	return rr.router.Dispatch(ctx, op, payload)
}
