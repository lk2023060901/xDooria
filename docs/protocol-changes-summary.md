# 协议修改总结

## 修改内容

### 1. 添加了获取角色列表协议

**文件：** `xDooria-proto-api/op_code.proto`

```protobuf
OP_GET_ROLES_REQ = 1010;   // 获取角色列表请求
OP_GET_ROLES_RES = 1011;   // 获取角色列表响应
```

**文件：** `xDooria-proto-api/gateway/gateway.proto`

```protobuf
// GetRolesRequest 获取角色列表请求 (OP_GET_ROLES_REQ)
message GetRolesRequest {
    // 无需参数，从 Session 中获取 uid
}

// GetRolesResponse 获取角色列表响应 (OP_GET_ROLES_RES)
message GetRolesResponse {
    uint32 code = 1;              // 错误码
    repeated RoleInfo roles = 2;  // 该账号下的所有角色
}
```

### 2. 修改了 AuthResponse

**原来：**
```protobuf
message AuthResponse {
    uint32 code = 1;
    string token = 2;
    uint64 uid = 3;
    repeated RoleInfo roles = 4;  // ← 移除了这个字段
}
```

**现在：**
```protobuf
message AuthResponse {
    uint32 code = 1;    // 错误码
    string token = 2;   // SessionToken
    uint64 uid = 3;     // 用户ID
}
```

## 修改原因

### 职责分离

- **OP_AUTH_REQ/RES**：只负责**认证**，返回 SessionToken 和 uid
- **OP_GET_ROLES_REQ/RES**：负责**查询角色列表**

### 灵活性

客户端可以在需要时随时刷新角色列表，例如：
- 在其他设备创建了新角色
- 角色被删除或封禁
- 需要重新加载角色信息

## 新的协议流程

```
1. OP_LOGIN_REQ/RES
   ↓
2. 连接 Gateway
   ↓
3. OP_AUTH_REQ/RES          ← 只返回 token 和 uid
   ↓
4. OP_GET_ROLES_REQ/RES     ← 新增：获取角色列表
   ↓
5. 判断：
   - 有角色 → OP_SELECT_ROLE_REQ/RES
   - 无角色 → OP_CREATE_ROLE_REQ/RES → OP_SELECT_ROLE_REQ/RES
   ↓
6. OP_ENTER_SCENE_REQ/RES
```

## 客户端逻辑

```javascript
// 伪代码
async function loginFlow() {
    // 1. 登录
    const loginRes = await login(username, password);

    // 2. 连接 Gateway
    await connectGateway(loginRes.gateway_addr);

    // 3. 认证
    const authRes = await auth(loginRes.token);

    // 4. 获取角色列表
    const rolesRes = await getRoles();

    // 5. 选择或创建角色
    let selectedRoleId;
    if (rolesRes.roles.length > 0) {
        // 有角色，默认选择第一个
        selectedRoleId = rolesRes.roles[0].role_id;
    } else {
        // 没有角色，创建一个
        const createRes = await createRole("新角色", gender, appearance);
        selectedRoleId = createRes.role.role_id;
    }

    // 6. 选择角色
    await selectRole(selectedRoleId);

    // 7. 进入游戏
    await enterScene();
}
```

## 需要实现的 Gateway Handler

在 `app/gateway/internal/handler/handler.go` 中需要添加：

```go
func (h *GatewayHandler) registerHandlers() {
    // ... 现有的注册 ...

    // 新增：注册获取角色列表处理器
    RegisterHandler(h.sessionRouter,
        uint32(api.OpCode_OP_GET_ROLES_REQ),
        uint32(api.OpCode_OP_GET_ROLES_RES),
        h.handleGetRoles)
}

// handleGetRoles 处理获取角色列表请求
func (h *GatewayHandler) handleGetRoles(
    ctx context.Context,
    s session.Session,
    req *gateway.GetRolesRequest,
) (*gateway.GetRolesResponse, error) {
    // 1. 从 Session 获取 uid
    gwSess, ok := h.sessMgr.Get(s.ID())
    if !ok || !gwSess.IsAuthenticated() {
        return &gateway.GetRolesResponse{
            Code: uint32(api.ErrorCode_ERR_NOT_AUTHENTICATED),
        }, nil
    }

    uid := gwSess.GetUID()

    // 2. 查询角色列表（从数据库或缓存）
    roles, err := h.roleDAO.GetRolesByUID(ctx, uid)
    if err != nil {
        h.logger.Error("failed to get roles", "uid", uid, "error", err)
        return &gateway.GetRolesResponse{
            Code: uint32(api.ErrorCode_ERR_INTERNAL),
        }, nil
    }

    // 3. 转换为 proto RoleInfo
    protoRoles := make([]*gateway.RoleInfo, 0, len(roles))
    for _, role := range roles {
        protoRoles = append(protoRoles, &gateway.RoleInfo{
            RoleId:   role.RoleID,
            Nickname: role.Nickname,
            Level:    role.Level,
            Status:   role.Status,
        })
    }

    // 4. 返回响应
    return &gateway.GetRolesResponse{
        Code:  uint32(api.ErrorCode_ERR_SUCCESS),
        Roles: protoRoles,
    }, nil
}
```

## 数据库查询

需要在 Gateway 的 DAO 层添加：

```go
// app/gateway/internal/dao/role_dao.go

type RoleDAO struct {
    db     *postgres.Client
    logger logger.Logger
}

func NewRoleDAO(db *postgres.Client, l logger.Logger) *RoleDAO {
    return &RoleDAO{
        db:     db,
        logger: l.Named("dao.role"),
    }
}

// GetRolesByUID 查询用户的所有角色
func (d *RoleDAO) GetRolesByUID(ctx context.Context, uid uint64) ([]*model.Role, error) {
    query := `
        SELECT role_id, uid, nickname, level, status, created_at, updated_at
        FROM roles
        WHERE uid = $1
        ORDER BY created_at DESC
    `

    rows, err := d.db.Query(ctx, query, uid)
    if err != nil {
        return nil, fmt.Errorf("query roles failed: %w", err)
    }
    defer rows.Close()

    var roles []*model.Role
    for rows.Next() {
        role := &model.Role{}
        err := rows.Scan(
            &role.RoleID,
            &role.UID,
            &role.Nickname,
            &role.Level,
            &role.Status,
            &role.CreatedAt,
            &role.UpdatedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("scan role failed: %w", err)
        }
        roles = append(roles, role)
    }

    return roles, rows.Err()
}
```

## 影响范围

### 需要修改的文件

1. ✅ **xDooria-proto-api/op_code.proto** - 已添加 OpCode
2. ✅ **xDooria-proto-api/gateway/gateway.proto** - 已添加消息定义，已修改 AuthResponse
3. ⚠️ **app/gateway/internal/handler/handler.go** - 需要移除 handleAuth 中返回角色列表的逻辑
4. ⚠️ **app/gateway/internal/handler/handler.go** - 需要添加 handleGetRoles
5. ⚠️ **app/gateway/internal/dao/role_dao.go** - 需要添加 GetRolesByUID 方法（如果不存在）

### 需要重新生成的代码

在 `xDooria-proto-api` 仓库中运行：
```bash
make generate
```

这将重新生成：
- `gateway/gateway.pb.go`
- `op_code.pb.go`

### 客户端影响

客户端需要修改登录流程，在 Auth 成功后添加 GetRoles 请求。

## 兼容性

### 破坏性变更

- ❌ **AuthResponse 移除了 roles 字段**
  - 现有客户端如果依赖 `AuthResponse.roles`，会失败
  - 需要客户端同步更新

### 建议

如果需要保持兼容，可以考虑：
1. 保留 AuthResponse.roles 字段（标记为 deprecated）
2. 同时支持新旧两种方式
3. 给客户端一个迁移期

但根据你的要求，这是一个**新的流程设计**，所以直接使用新方式即可。

## 下一步

1. 在 `xDooria-proto-api` 仓库中运行 `make generate` 重新生成 Go 代码
2. 在 Gateway 中实现 `handleGetRoles`
3. 修改 `handleAuth`，移除返回角色列表的逻辑
4. 更新客户端代码，添加 GetRoles 请求
5. 测试完整流程

## 参考文档

- [完整协议流程文档](./complete-protocol-flow.md)
- [Router 实现总结](./router-implementation-summary.md)
- [RoleRouter 使用指南](./role-router-usage.md)
