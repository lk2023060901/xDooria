package repository

import (
	"context"

	"github.com/lk2023060901/xdooria/app/game/internal/dao"
	"github.com/lk2023060901/xdooria/app/game/internal/model"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// PlayerRepository 玩家聚合仓储接口
type PlayerRepository interface {
	// ===== 玩偶相关 =====
	GetDolls(ctx context.Context, playerID int64) ([]*model.Doll, error)
	GetDollByID(ctx context.Context, playerID int64, dollID int64) (*model.Doll, error)
	AddDoll(ctx context.Context, doll *model.Doll) error
	AddDolls(ctx context.Context, dolls []*model.Doll) error
	UpdateDollLock(ctx context.Context, playerID int64, dollID int64, isLocked bool) error
	UpdateDollRedeem(ctx context.Context, playerID int64, dollID int64, isRedeemed bool) error
	UpdateDollQuality(ctx context.Context, playerID int64, dollID int64, quality int16) error
	DeleteDoll(ctx context.Context, playerID int64, dollID int64) error
	DeleteDolls(ctx context.Context, playerID int64, dollIDs []int64) error

	// ===== 抽卡相关 =====
	GetGachaRecords(ctx context.Context, roleID int64) (*model.PlayerGacha, error)
	SaveGachaRecords(ctx context.Context, gacha *model.PlayerGacha) error

	// ===== 背包(堆叠道具)相关 =====
	GetBag(ctx context.Context, roleID int64, bagType int32) (*model.PlayerBag, error)
	SaveBag(ctx context.Context, bag *model.PlayerBag) error
}

// playerRepositoryImpl 玩家仓储实现
type playerRepositoryImpl struct {
	dollDAO  *dao.DollDAO
	gachaDAO *dao.GachaDAO
	bagDAO   *dao.BagDAO
	cacheDAO *dao.CacheDAO
	logger   logger.Logger
}

// NewPlayerRepository 创建玩家仓储
func NewPlayerRepository(
	dollDAO *dao.DollDAO,
	gachaDAO *dao.GachaDAO,
	bagDAO *dao.BagDAO,
	cacheDAO *dao.CacheDAO,
	l logger.Logger,
) PlayerRepository {
	return &playerRepositoryImpl{
		dollDAO:  dollDAO,
		gachaDAO: gachaDAO,
		bagDAO:   bagDAO,
		cacheDAO: cacheDAO,
		logger:   l.Named("repository.player"),
	}
}
