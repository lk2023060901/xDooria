-- 玩家抽卡记录表 (盲盒玩法)
CREATE TABLE IF NOT EXISTS player_gacha (
    role_id     BIGINT PRIMARY KEY,
    records     JSONB NOT NULL DEFAULT '[]', -- 存储玩法记录列表
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE player_gacha IS '玩家抽卡记录表';
COMMENT ON COLUMN player_gacha.role_id IS '角色ID';
COMMENT ON COLUMN player_gacha.records IS '抽卡玩法记录列表: [{id, type, total_count, last_time}]';
