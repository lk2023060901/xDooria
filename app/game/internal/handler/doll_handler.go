package handler

import (
	"context"
	"fmt"

	api "github.com/lk2023060901/xdooria-proto-api"
	"github.com/lk2023060901/xdooria/app/game/internal/gameconfig"
	gamerouter "github.com/lk2023060901/xdooria/app/game/internal/router"
	"github.com/lk2023060901/xdooria/app/game/internal/service"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// DollHandler 玩偶协议处理器
type DollHandler struct {
	logger  logger.Logger
	dollSvc *service.DollService
}

// NewDollHandler 创建玩偶处理器
func NewDollHandler(
	l logger.Logger,
	dollSvc *service.DollService,
) *DollHandler {
	return &DollHandler{
		logger:  l.Named("handler.doll"),
		dollSvc: dollSvc,
	}
}

// RegisterHandlers 注册玩偶相关的消息处理器
func (h *DollHandler) RegisterHandlers(roleRouter *gamerouter.RoleRouter) {
	// 注册玩偶列表
	gamerouter.RegisterHandler(roleRouter,
		uint32(api.OpCode_OP_DOLL_LIST_REQ),
		uint32(api.OpCode_OP_DOLL_LIST_RES),
		h.HandleDollList)

	// 注册玩偶收藏
	gamerouter.RegisterHandler(roleRouter,
		uint32(api.OpCode_OP_DOLL_FAVORITE_REQ),
		uint32(api.OpCode_OP_DOLL_FAVORITE_RES),
		h.HandleDollFavorite)

	// 注册玩偶锁定
	gamerouter.RegisterHandler(roleRouter,
		uint32(api.OpCode_OP_DOLL_LOCK_REQ),
		uint32(api.OpCode_OP_DOLL_LOCK_RES),
		h.HandleDollLock)

	// 注册玩偶兑换
	gamerouter.RegisterHandler(roleRouter,
		uint32(api.OpCode_OP_DOLL_REDEEM_REQ),
		uint32(api.OpCode_OP_DOLL_REDEEM_RES),
		h.HandleDollRedeem)
}

// HandleDollList 处理获取玩偶列表请求
func (h *DollHandler) HandleDollList(ctx context.Context, roleID int64, req *api.DollListRequest) (*api.DollListResponse, error) {
	h.logger.Debug("handling doll list request",
		"role_id", roleID,
	)

	// 1. 从 service 获取玩偶列表（使用默认排序）
	dolls, err := h.dollSvc.GetDolls(ctx, roleID, gameconfig.DollSortType_Quality)
	if err != nil {
		h.logger.Error("failed to get dolls",
			"role_id", roleID,
			"error", err,
		)
		return &api.DollListResponse{
			Code: api.ErrorCode_ERR_INTERNAL,
		}, nil
	}

	// 2. 转换为 protobuf 格式
	dollItems := make([]*api.DollItem, 0, len(dolls))
	for _, doll := range dolls {
		dollItems = append(dollItems, &api.DollItem{
			Id:         doll.ID,
			DollId:     doll.DollID,
			Quality:    int32(doll.Quality),
			IsLocked:   doll.IsLocked,
			IsRedeemed: doll.IsRedeemed,
			// TODO: IsFavorite 字段需要在 model.Doll 中添加
			IsFavorite: false,
			CreatedAt:  doll.CreatedAt.Unix(),
		})
	}

	h.logger.Info("doll list response sent",
		"role_id", roleID,
		"count", len(dollItems),
	)

	return &api.DollListResponse{
		Code:  api.ErrorCode_ERR_SUCCESS,
		Dolls: dollItems,
	}, nil
}

// HandleDollFavorite 处理收藏/取消收藏请求
func (h *DollHandler) HandleDollFavorite(ctx context.Context, roleID int64, req *api.DollFavoriteRequest) (*api.DollFavoriteResponse, error) {
	h.logger.Debug("handling doll favorite request",
		"role_id", roleID,
		"doll_id", req.Id,
		"favorite", req.Favorite,
	)

	// TODO: 实现收藏功能（需要先在 model.Doll 中添加 IsFavorite 字段）
	// if req.Favorite {
	//     err = h.dollSvc.FavoriteDoll(ctx, roleID, req.Id)
	// } else {
	//     err = h.dollSvc.UnfavoriteDoll(ctx, roleID, req.Id)
	// }

	h.logger.Warn("doll favorite feature not implemented yet",
		"role_id", roleID,
		"doll_id", req.Id,
	)

	return &api.DollFavoriteResponse{
		Code:       api.ErrorCode_ERR_NOT_IMPLEMENTED,
		Id:         req.Id,
		IsFavorite: req.Favorite,
	}, nil
}

// HandleDollLock 处理锁定/取消锁定请求
func (h *DollHandler) HandleDollLock(ctx context.Context, roleID int64, req *api.DollLockRequest) (*api.DollLockResponse, error) {
	h.logger.Debug("handling doll lock request",
		"role_id", roleID,
		"doll_id", req.Id,
		"lock", req.Lock,
	)

	// 1. 调用 service 层处理
	var err error
	if req.Lock {
		err = h.dollSvc.LockDoll(ctx, roleID, req.Id)
	} else {
		err = h.dollSvc.UnlockDoll(ctx, roleID, req.Id)
	}

	if err != nil {
		h.logger.Error("failed to lock/unlock doll",
			"role_id", roleID,
			"doll_id", req.Id,
			"lock", req.Lock,
			"error", err,
		)

		// 根据错误类型返回相应的错误码
		errCode := api.ErrorCode_ERR_INTERNAL
		errMsg := err.Error()

		if errMsg == "doll not found" || errMsg == fmt.Sprintf("doll not found: player_id=%d, doll_id=%d", roleID, req.Id) {
			errCode = api.ErrorCode_ERR_NOT_FOUND
		} else if errMsg == "doll already locked" || errMsg == "doll already unlocked" {
			errCode = api.ErrorCode_ERR_ALREADY_EXISTS
		}

		return &api.DollLockResponse{
			Code:     errCode,
			Id:       req.Id,
			IsLocked: req.Lock,
		}, nil
	}

	h.logger.Info("doll lock status updated",
		"role_id", roleID,
		"doll_id", req.Id,
		"is_locked", req.Lock,
	)

	return &api.DollLockResponse{
		Code:     api.ErrorCode_ERR_SUCCESS,
		Id:       req.Id,
		IsLocked: req.Lock,
	}, nil
}

// HandleDollRedeem 处理兑换实体玩偶请求
func (h *DollHandler) HandleDollRedeem(ctx context.Context, roleID int64, req *api.DollRedeemRequest) (*api.DollRedeemResponse, error) {
	h.logger.Debug("handling doll redeem request",
		"role_id", roleID,
		"doll_id", req.Id,
	)

	// 1. 调用 service 层处理兑换
	err := h.dollSvc.RedeemRealDoll(ctx, roleID, req.Id)
	if err != nil {
		h.logger.Error("failed to redeem doll",
			"role_id", roleID,
			"doll_id", req.Id,
			"error", err,
		)

		// 根据错误类型返回相应的错误码
		errCode := api.ErrorCode_ERR_INTERNAL
		errMsg := err.Error()

		if errMsg == "doll not found" {
			errCode = api.ErrorCode_ERR_NOT_FOUND
		} else if errMsg == "doll already redeemed" {
			errCode = api.ErrorCode_ERR_ALREADY_EXISTS
		} else if errMsg == "doll config not found" {
			errCode = api.ErrorCode_ERR_CONFIG_NOT_FOUND
		} else if errMsg == "doll cannot be redeemed" {
			errCode = api.ErrorCode_ERR_PERMISSION_DENIED
		} else if errMsg == "insufficient currency" {
			errCode = api.ErrorCode_ERR_INSUFFICIENT_CURRENCY
		}

		return &api.DollRedeemResponse{
			Code: errCode,
		}, nil
	}

	h.logger.Info("doll redeemed successfully",
		"role_id", roleID,
		"doll_id", req.Id,
	)

	return &api.DollRedeemResponse{
		Code: api.ErrorCode_ERR_SUCCESS,
	}, nil
}
