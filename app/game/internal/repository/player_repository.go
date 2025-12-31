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
}

// playerRepositoryImpl 玩家仓储实现
type playerRepositoryImpl struct {
	dollDAO  *dao.DollDAO
	cacheDAO *dao.CacheDAO
	logger   logger.Logger
}

// NewPlayerRepository 创建玩家仓储
func NewPlayerRepository(
	dollDAO *dao.DollDAO,
	cacheDAO *dao.CacheDAO,
	l logger.Logger,
) PlayerRepository {
	return &playerRepositoryImpl{
		dollDAO:  dollDAO,
		cacheDAO: cacheDAO,
		logger:   l.Named("repository.player"),
	}
}
