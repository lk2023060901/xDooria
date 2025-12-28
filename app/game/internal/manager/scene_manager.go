package manager

import (
	"fmt"
	"sync"

	api "github.com/lk2023060901/xdooria-proto-api"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// Scene 场景实例
type Scene struct {
	MapID   int32
	Players map[int64]*PlayerInScene // roleID -> PlayerInScene
	mu      sync.RWMutex
}

// PlayerInScene 场景中的玩家信息
type PlayerInScene struct {
	RoleID   int64
	Position *api.Position
}

// SceneManager 场景管理器
type SceneManager struct {
	logger logger.Logger

	mu     sync.RWMutex
	scenes map[int32]*Scene // mapID -> Scene

	// 角色所在场景映射
	roleScenes map[int64]int32 // roleID -> mapID
}

// NewSceneManager 创建场景管理器
func NewSceneManager(l logger.Logger) *SceneManager {
	return &SceneManager{
		logger:     l.Named("manager.scene"),
		scenes:     make(map[int32]*Scene),
		roleScenes: make(map[int64]int32),
	}
}

// GetOrCreateScene 获取或创建场景实例
func (m *SceneManager) GetOrCreateScene(mapID int32) (*Scene, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	scene, exists := m.scenes[mapID]
	if !exists {
		scene = &Scene{
			MapID:   mapID,
			Players: make(map[int64]*PlayerInScene),
		}
		m.scenes[mapID] = scene
		m.logger.Info("scene created", "map_id", mapID)
	}

	return scene, nil
}

// EnterScene 角色进入场景
func (m *SceneManager) EnterScene(roleID int64, mapID int32, position *api.Position) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果角色已经在其他场景，先离开
	if oldMapID, exists := m.roleScenes[roleID]; exists {
		if oldScene, ok := m.scenes[oldMapID]; ok {
			oldScene.mu.Lock()
			delete(oldScene.Players, roleID)
			oldScene.mu.Unlock()
			m.logger.Debug("role left old scene",
				"role_id", roleID,
				"old_map_id", oldMapID,
			)
		}
	}

	// 获取目标场景
	scene, exists := m.scenes[mapID]
	if !exists {
		return fmt.Errorf("scene not found: %d", mapID)
	}

	// 加入新场景
	scene.mu.Lock()
	scene.Players[roleID] = &PlayerInScene{
		RoleID:   roleID,
		Position: position,
	}
	scene.mu.Unlock()

	// 更新映射
	m.roleScenes[roleID] = mapID

	m.logger.Info("role entered scene",
		"role_id", roleID,
		"map_id", mapID,
		"player_count", len(scene.Players),
	)

	return nil
}

// LeaveScene 角色离开场景
func (m *SceneManager) LeaveScene(roleID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mapID, exists := m.roleScenes[roleID]
	if !exists {
		return fmt.Errorf("role not in any scene: %d", roleID)
	}

	scene, ok := m.scenes[mapID]
	if !ok {
		delete(m.roleScenes, roleID)
		return fmt.Errorf("scene not found: %d", mapID)
	}

	scene.mu.Lock()
	delete(scene.Players, roleID)
	playerCount := len(scene.Players)
	scene.mu.Unlock()

	delete(m.roleScenes, roleID)

	m.logger.Info("role left scene",
		"role_id", roleID,
		"map_id", mapID,
		"remaining_players", playerCount,
	)

	return nil
}

// GetRoleScene 获取角色所在场景
func (m *SceneManager) GetRoleScene(roleID int64) (int32, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mapID, exists := m.roleScenes[roleID]
	return mapID, exists
}

// GetScene 获取场景实例
func (m *SceneManager) GetScene(mapID int32) (*Scene, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	scene, exists := m.scenes[mapID]
	return scene, exists
}

// GetPlayersInScene 获取场景中的所有玩家
func (m *SceneManager) GetPlayersInScene(mapID int32) []*PlayerInScene {
	m.mu.RLock()
	scene, exists := m.scenes[mapID]
	m.mu.RUnlock()

	if !exists {
		return nil
	}

	scene.mu.RLock()
	defer scene.mu.RUnlock()

	players := make([]*PlayerInScene, 0, len(scene.Players))
	for _, player := range scene.Players {
		players = append(players, player)
	}

	return players
}
