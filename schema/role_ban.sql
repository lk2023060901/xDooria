-- 角色封禁记录表
CREATE TABLE IF NOT EXISTS role_bans (
    id              BIGSERIAL PRIMARY KEY,              -- 记录ID
    role_id         BIGINT NOT NULL,                    -- 角色ID
    ban_type        SMALLINT NOT NULL DEFAULT 3,        -- 封禁类型: 1禁言 2禁赛 3全封禁
    reason          VARCHAR(256) NOT NULL DEFAULT '',   -- 封禁原因
    operator_id     BIGINT NOT NULL DEFAULT 0,          -- 操作人ID（0=系统）
    start_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- 封禁开始时间
    expire_at       TIMESTAMPTZ,                        -- 到期时间，NULL=永久
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()  -- 创建时间
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_role_bans_role_id ON role_bans(role_id);
CREATE INDEX IF NOT EXISTS idx_role_bans_expire_at ON role_bans(expire_at) WHERE expire_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_role_bans_created_at ON role_bans(created_at);

-- 注释
COMMENT ON TABLE role_bans IS '角色封禁记录表';
COMMENT ON COLUMN role_bans.id IS '记录ID';
COMMENT ON COLUMN role_bans.role_id IS '角色ID';
COMMENT ON COLUMN role_bans.ban_type IS '封禁类型: 1禁言 2禁赛 3全封禁';
COMMENT ON COLUMN role_bans.reason IS '封禁原因';
COMMENT ON COLUMN role_bans.operator_id IS '操作人ID，0表示系统自动';
COMMENT ON COLUMN role_bans.start_at IS '封禁开始时间';
COMMENT ON COLUMN role_bans.expire_at IS '到期时间，NULL表示永久封禁';
COMMENT ON COLUMN role_bans.created_at IS '创建时间';
