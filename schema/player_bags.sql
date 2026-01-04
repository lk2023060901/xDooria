-- 玩家背包表 (按类型切分，使用 Protobuf 序列化存储)
CREATE TABLE IF NOT EXISTS player_bags (
    role_id     BIGINT NOT NULL,
    bag_type    INT NOT NULL,        -- 背包类型
    data        BYTEA NOT NULL,      -- 序列化后的 PB 字节流 (ItemBag 或 DollBag)
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, bag_type)
);

CREATE INDEX IF NOT EXISTS idx_player_bags_role_id ON player_bags(role_id);

COMMENT ON TABLE player_bags IS '玩家背包表';
COMMENT ON COLUMN player_bags.role_id IS '角色ID';
COMMENT ON COLUMN player_bags.bag_type IS '背包类型';
COMMENT ON COLUMN player_bags.data IS 'Protobuf 序列化数据 (ItemBag 或 DollBag)';
