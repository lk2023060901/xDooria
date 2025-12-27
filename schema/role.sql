-- 角色表
CREATE TABLE IF NOT EXISTS roles (
    -- 基础信息
    id              BIGSERIAL PRIMARY KEY,              -- 角色ID
    uid             BIGINT NOT NULL,                    -- 关联用户ID
    nickname        VARCHAR(32) NOT NULL,               -- 昵称
    gender          SMALLINT NOT NULL DEFAULT 0,        -- 性别: 0未知 1男 2女
    signature       VARCHAR(128) NOT NULL DEFAULT '',   -- 个性签名
    avatar_id       INT NOT NULL DEFAULT 0,             -- 头像ID

    -- 形象装扮
    appearance      JSONB NOT NULL DEFAULT '{}',        -- 角色外观配置（脸型、发型、肤色等）
    outfit          JSONB NOT NULL DEFAULT '{}',        -- 当前穿戴装扮（衣服、裤子、鞋子、配饰等）

    -- 经济系统
    gold            BIGINT NOT NULL DEFAULT 0,          -- 金币（普通货币）
    diamond         BIGINT NOT NULL DEFAULT 0,          -- 钻石（付费货币）

    -- 等级成长
    level           INT NOT NULL DEFAULT 1,             -- 角色等级
    exp             BIGINT NOT NULL DEFAULT 0,          -- 当前经验值
    vip_level       SMALLINT NOT NULL DEFAULT 0,        -- VIP等级
    vip_exp         INT NOT NULL DEFAULT 0,             -- VIP经验

    -- 封禁状态（冗余字段，快速判断）
    status          SMALLINT NOT NULL DEFAULT 0,        -- 状态: 0正常 1封禁
    ban_expire_at   TIMESTAMPTZ,                        -- 封禁到期时间，NULL=永久封禁

    -- 时间戳
    last_login_at   TIMESTAMPTZ,                        -- 最后登录时间
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()  -- 更新时间
);

-- 索引
CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_uid ON roles(uid);
CREATE INDEX IF NOT EXISTS idx_roles_nickname ON roles(nickname);
CREATE INDEX IF NOT EXISTS idx_roles_status ON roles(status) WHERE status != 0;
CREATE INDEX IF NOT EXISTS idx_roles_created_at ON roles(created_at);

-- 更新时间触发器
CREATE OR REPLACE FUNCTION update_roles_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_roles_updated_at ON roles;
CREATE TRIGGER trigger_roles_updated_at
    BEFORE UPDATE ON roles
    FOR EACH ROW
    EXECUTE FUNCTION update_roles_updated_at();

-- 注释
COMMENT ON TABLE roles IS '角色表';
COMMENT ON COLUMN roles.id IS '角色ID';
COMMENT ON COLUMN roles.uid IS '关联用户ID';
COMMENT ON COLUMN roles.nickname IS '昵称';
COMMENT ON COLUMN roles.gender IS '性别: 0未知 1男 2女';
COMMENT ON COLUMN roles.signature IS '个性签名';
COMMENT ON COLUMN roles.avatar_id IS '头像ID';
COMMENT ON COLUMN roles.appearance IS '角色外观配置（脸型、发型、肤色等）';
COMMENT ON COLUMN roles.outfit IS '当前穿戴装扮（衣服、裤子、鞋子、配饰等）';
COMMENT ON COLUMN roles.gold IS '金币（普通货币）';
COMMENT ON COLUMN roles.diamond IS '钻石（付费货币）';
COMMENT ON COLUMN roles.level IS '角色等级';
COMMENT ON COLUMN roles.exp IS '当前经验值';
COMMENT ON COLUMN roles.vip_level IS 'VIP等级';
COMMENT ON COLUMN roles.vip_exp IS 'VIP经验';
COMMENT ON COLUMN roles.status IS '状态: 0正常 1封禁';
COMMENT ON COLUMN roles.ban_expire_at IS '封禁到期时间，NULL表示永久封禁';
COMMENT ON COLUMN roles.last_login_at IS '最后登录时间';
COMMENT ON COLUMN roles.created_at IS '创建时间';
COMMENT ON COLUMN roles.updated_at IS '更新时间';
