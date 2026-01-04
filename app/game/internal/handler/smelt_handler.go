package handler

import (
	"context"

	api "github.com/lk2023060901/xdooria-proto-api"
	gamerouter "github.com/lk2023060901/xdooria/app/game/internal/router"
	"github.com/lk2023060901/xdooria/app/game/internal/service"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

type SmeltHandler struct {
	logger   logger.Logger
	smeltSvc *service.SmeltService
}

func NewSmeltHandler(l logger.Logger, smeltSvc *service.SmeltService) *SmeltHandler {
	return &SmeltHandler{
		logger:   l.Named("handler.smelt"),
		smeltSvc: smeltSvc,
	}
}

func (h *SmeltHandler) RegisterHandlers(roleRouter *gamerouter.RoleRouter) {
	gamerouter.RegisterHandler(roleRouter,
		uint32(api.OpCode_OP_SMELT_DO_REQ),
		uint32(api.OpCode_OP_SMELT_DO_RES),
		h.HandleSmelt)
}

func (h *SmeltHandler) HandleSmelt(ctx context.Context, roleID int64, req *api.SmeltDoRequest) (*api.SmeltDoResponse, error) {
	newDoll, upgraded, err := h.smeltSvc.Smelt(ctx, roleID, req.DollIds)
	if err != nil {
		h.logger.Error("smelt failed", "role_id", roleID, "error", err)
		return &api.SmeltDoResponse{Code: api.ErrorCode_ERR_INTERNAL}, nil
	}

	var result *api.SmeltResult
	if newDoll != nil {
		result = &api.SmeltResult{
			Doll: &api.DollItem{
				Id:         newDoll.ID,
				DollId:     newDoll.DollID,
				Quality:    int32(newDoll.Quality),
				IsLocked:   newDoll.IsLocked,
				IsRedeemed: newDoll.IsRedeemed,
				CreatedAt:  newDoll.CreatedAt.Unix(),
			},
			Upgraded: upgraded,
		}
	}

	return &api.SmeltDoResponse{
		Code:   api.ErrorCode_ERR_SUCCESS,
		Result: result,
	}, nil
}
