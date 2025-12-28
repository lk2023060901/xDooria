package redis

import (
	"context"
	"encoding/json"
	"fmt"

	common "github.com/lk2023060901/xdooria-proto-common"
	gwsession "github.com/lk2023060901/xdooria/app/gateway/internal/session"
	"github.com/lk2023060901/xdooria/pkg/database/redis"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
)

// KickMessage 踢人消息
type KickMessage struct {
	RoleID  int64  `json:"role_id"`
	Reason  uint32 `json:"reason"`
	Message string `json:"message"`
}

// KickListener Redis Pub/Sub 踢人监听器
type KickListener struct {
	logger    logger.Logger
	client    redis.Client
	sessMgr   *gwsession.Manager
	ctx       context.Context
	cancel    context.CancelFunc
	subFuture *conc.Future[struct{}]
}

// NewKickListener 创建踢人监听器
func NewKickListener(l logger.Logger, client redis.Client, sessMgr *gwsession.Manager) *KickListener {
	ctx, cancel := context.WithCancel(context.Background())
	return &KickListener{
		logger:  l.Named("redis.kick_listener"),
		client:  client,
		sessMgr: sessMgr,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start 启动监听
func (kl *KickListener) Start() error {
	// 订阅所有角色的踢人频道（使用模式匹配）
	pattern := "kick:role:*"

	pubsub, err := kl.client.PSubscribe(kl.ctx, pattern)
	if err != nil {
		return fmt.Errorf("redis psubscribe failed: %w", err)
	}

	kl.logger.Info("kick listener started", "pattern", pattern)

	// 启动消息处理循环
	kl.subFuture = conc.Go(func() (struct{}, error) {
		return struct{}{}, kl.messageLoop(pubsub)
	})

	return nil
}

// Stop 停止监听
func (kl *KickListener) Stop() error {
	kl.cancel()

	if kl.subFuture != nil {
		if err := kl.subFuture.Err(); err != nil {
			kl.logger.Warn("kick listener stopped with error", "error", err)
		}
	}

	kl.logger.Info("kick listener stopped")
	return nil
}

// messageLoop 消息处理循环
func (kl *KickListener) messageLoop(pubsub redis.PubSub) error {
	msgChan := pubsub.Channel()

	for {
		select {
		case <-kl.ctx.Done():
			return pubsub.Close()

		case msg, ok := <-msgChan:
			if !ok {
				kl.logger.Warn("pubsub channel closed")
				return nil
			}

			if err := kl.handleKickMessage(msg.Payload); err != nil {
				kl.logger.Error("handle kick message failed",
					"channel", msg.Channel,
					"payload", msg.Payload,
					"error", err,
				)
			}
		}
	}
}

// handleKickMessage 处理踢人消息
func (kl *KickListener) handleKickMessage(payload string) error {
	var kickMsg KickMessage
	if err := json.Unmarshal([]byte(payload), &kickMsg); err != nil {
		return fmt.Errorf("unmarshal kick message failed: %w", err)
	}

	kl.logger.Info("received kick command",
		"role_id", kickMsg.RoleID,
		"reason", kickMsg.Reason,
		"message", kickMsg.Message,
	)

	// 查找该角色对应的会话
	gwSess, ok := kl.sessMgr.GetByRoleID(kickMsg.RoleID)
	if !ok {
		kl.logger.Debug("role session not found on this gateway",
			"role_id", kickMsg.RoleID,
		)
		return nil
	}

	// 发送踢人通知给客户端
	kickNotice := &common.Envelope{
		Header: &common.MessageHeader{
			Op: uint32(common.OpCode_OP_KICK_NOTICE),
		},
		Payload: []byte(kickMsg.Message),
	}

	baseSess := gwSess.BaseSession()
	if err := baseSess.Send(kl.ctx, kickNotice); err != nil {
		kl.logger.Warn("send kick notice failed",
			"role_id", kickMsg.RoleID,
			"session_id", baseSess.ID(),
			"error", err,
		)
	}

	// 关闭会话
	if err := baseSess.Close(); err != nil {
		kl.logger.Warn("close session failed",
			"role_id", kickMsg.RoleID,
			"session_id", baseSess.ID(),
			"error", err,
		)
	}

	kl.logger.Info("kicked role",
		"role_id", kickMsg.RoleID,
		"session_id", baseSess.ID(),
		"reason", kickMsg.Reason,
	)

	return nil
}
