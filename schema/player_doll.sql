-- 玩家玩偶表
CREATE TABLE IF NOT EXISTS player_doll (
    id          BIGINT PRIMARY KEY,                     -- 实例ID（雪花ID）
    player_id   BIGINT NOT NULL,                        -- 玩家ID，引用 roles.id
    doll_id     INT NOT NULL,                           -- 配置ID，引用 Doll.id
    quality     SMALLINT NOT NULL,                      -- 当前品质（可通过熔炼提升）
    is_locked   BOOLEAN NOT NULL DEFAULT FALSE,         -- 锁定状态（防误熔炼）
    is_redeemed BOOLEAN NOT NULL DEFAULT FALSE,         -- 是否已兑换实物
    created_at  TIMESTAMPTZ NOT NULL                    -- 创建时间
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_player_doll_player_id ON player_doll(player_id);
CREATE INDEX IF NOT EXISTS idx_player_doll_doll_id ON player_doll(player_id, doll_id);

-- 注释
COMMENT ON TABLE player_doll IS '玩家玩偶表';
COMMENT ON COLUMN player_doll.id IS '实例ID（雪花ID）';
COMMENT ON COLUMN player_doll.player_id IS '玩家ID，引用 roles.id';
COMMENT ON COLUMN player_doll.doll_id IS '配置ID，引用 Doll.id';
COMMENT ON COLUMN player_doll.quality IS '当前品质（可通过熔炼提升）';
COMMENT ON COLUMN player_doll.is_locked IS '锁定状态（防误熔炼）';
COMMENT ON COLUMN player_doll.is_redeemed IS '是否已兑换实物';
COMMENT ON COLUMN player_doll.created_at IS '创建时间';
