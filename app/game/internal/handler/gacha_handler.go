package handler

import (
	"context"

	api "github.com/lk2023060901/xdooria-proto-api"
	gamerouter "github.com/lk2023060901/xdooria/app/game/internal/router"
	"github.com/lk2023060901/xdooria/app/game/internal/service"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

type GachaHandler struct {
	logger   logger.Logger
	gachaSvc *service.GachaService
}

func NewGachaHandler(l logger.Logger, gachaSvc *service.GachaService) *GachaHandler {
	return &GachaHandler{
		logger:   l.Named("handler.gacha"),
		gachaSvc: gachaSvc,
	}
}

func (h *GachaHandler) RegisterHandlers(roleRouter *gamerouter.RoleRouter) {
	gamerouter.RegisterHandler(roleRouter,
		uint32(api.OpCode_OP_GACHA_DRAW_REQ),
		uint32(api.OpCode_OP_GACHA_DRAW_RES),
		h.HandleDraw)
}

func (h *GachaHandler) HandleDraw(ctx context.Context, roleID int64, req *api.GachaDrawRequest) (*api.GachaDrawResponse, error) {
	results, totalCount, err := h.gachaSvc.Draw(ctx, roleID, req.PoolId, req.Count)
	if err != nil {
		h.logger.Error("draw failed", "role_id", roleID, "pool_id", req.PoolId, "error", err)
		return &api.GachaDrawResponse{Code: api.ErrorCode_ERR_INTERNAL}, nil
	}

	rewards := make([]*api.GachaReward, 0, len(results))
	for _, r := range results {
		rewards = append(rewards, &api.GachaReward{
			Id:    r.ItemID,
			Count: r.Count,
		})
	}

	return &api.GachaDrawResponse{
		Code:      api.ErrorCode_ERR_SUCCESS,
		Rewards:   rewards,
		DrawCount: totalCount,
	}, nil
}
