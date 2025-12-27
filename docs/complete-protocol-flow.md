# 完整协议流程：从客户端登录到游戏

本文档详细描述从客户端登录到进入游戏场景的完整协议流程，包括所有涉及的服务、协议、Token 流转和 RoleID 传递机制。

## 协议流程概览

```
┌─────────┐         ┌───────────┐         ┌──────────┐         ┌────────┐
│ Client  │         │ Login Svc │         │ Gateway  │         │ Game   │
└─────────┘         └───────────┘         └──────────┘         └────────┘
     │                    │                     │                    │
     │ 1. Login           │                     │                    │
     │ OP_LOGIN_REQ       │                     │                    │
     │ (1000)             │                     │                    │
     │───────────────────>│                     │                    │
     │                    │                     │                    │
     │                    │ 验证凭证            │                    │
     │                    │ 生成 LoginToken     │                    │
     │                    │ 分配 Gateway        │                    │
     │                    │                     │                    │
     │ 2. LoginResponse   │                     │                    │
     │ OP_LOGIN_RES       │                     │                    │
     │ (1001)             │                     │                    │
     │<───────────────────│                     │                    │
     │ {                  │                     │                    │
     │   token: "login-jwt-xxx",                │                    │
     │   uid: 100001,     │                     │                    │
     │   gateway_addr: "gw1.game.com:8080"      │                    │
     │ }                  │                     │                    │
     │                    │                     │                    │
     │ 3. 连接 Gateway    │                     │                    │
     │ (WebSocket/TCP)    │                     │                    │
     │─────────────────────────────────────────>│                    │
     │                    │                     │                    │
     │ 4. Auth            │                     │                    │
     │ OP_AUTH_REQ        │                     │                    │
     │ (1002)             │                     │                    │
     │─────────────────────────────────────────>│                    │
     │ {                  │                     │                    │
     │   login_token: "login-jwt-xxx"           │                    │
     │ }                  │                     │                    │
     │                    │                     │                    │
     │                    │                     │ 验证 LoginToken    │
     │                    │                     │ 提取 uid           │
     │                    │                     │ 生成 SessionToken  │
     │                    │                     │ 创建 GatewaySession│
     │                    │                     │                    │
     │ 5. AuthResponse    │                     │                    │
     │ OP_AUTH_RES        │                     │                    │
     │ (1003)             │                     │                    │
     │<─────────────────────────────────────────│                    │
     │ {                  │                     │                    │
     │   code: 0,         │                     │                    │
     │   token: "session-jwt-yyy",              │                    │
     │   uid: 100001      │                     │                    │
     │ }                  │                     │                    │
     │                    │                     │                    │
     │ 6. GetRoles        │                     │                    │
     │ OP_GET_ROLES_REQ   │                     │                    │
     │ (1010)             │                     │                    │
     │─────────────────────────────────────────>│                    │
     │ {}                 │                     │                    │
     │                    │                     │                    │
     │                    │                     │ SessionRouter 处理 │
     │                    │                     │ 从 Session 获取 uid│
     │                    │                     │ 查询角色列表       │
     │                    │                     │                    │
     │ 7. GetRolesResponse│                     │                    │
     │ OP_GET_ROLES_RES   │                     │                    │
     │ (1011)             │                     │                    │
     │<─────────────────────────────────────────│                    │
     │ {                  │                     │                    │
     │   code: 0,         │                     │                    │
     │   roles: [         │                     │                    │
     │     {role_id: 10001, nickname: "勇士", level: 50},            │
     │     {role_id: 10002, nickname: "法师", level: 30}             │
     │   ]                │                     │                    │
     │ }                  │                     │                    │
     │                    │                     │                    │
     │ 【客户端逻辑判断】 │                     │                    │
     │ if (roles.length > 0):                   │                    │
     │   选择第一个角色 (role_id = roles[0].role_id)                 │
     │ else:              │                     │                    │
     │   创建新角色       │                     │                    │
     │                    │                     │                    │
     │ 【情况A：有角色，直接选择第一个】       │                    │
     │                    │                     │                    │
     │ 8a. SelectRole     │                     │                    │
     │ OP_SELECT_ROLE_REQ │                     │                    │
     │ (1006)             │                     │                    │
     │─────────────────────────────────────────>│                    │
     │ {                  │                     │                    │
     │   role_id: 10001   │                     │                    │
     │ }                  │                     │                    │
     │                    │                     │                    │
     │                    │                     │ SessionRouter 处理 │
     │                    │                     │ 验证角色属于该 uid │
     │                    │                     │ GatewaySession 设置│
     │                    │                     │ roleID = 10001     │
     │                    │                     │                    │
     │ 9a. SelectRoleResponse                   │                    │
     │ OP_SELECT_ROLE_RES │                     │                    │
     │ (1007)             │                     │                    │
     │<─────────────────────────────────────────│                    │
     │ {                  │                     │                    │
     │   code: 0,         │                     │                    │
     │   role_id: 10001   │                     │                    │
     │ }                  │                     │                    │
     │                    │                     │                    │
     │ → 跳转到步骤 10    │                     │                    │
     │                    │                     │                    │
     │ 【情况B：无角色，需要创建】             │                    │
     │                    │                     │                    │
     │ 8b. CreateRole     │                     │                    │
     │ OP_CREATE_ROLE_REQ │                     │                    │
     │ (1008)             │                     │                    │
     │─────────────────────────────────────────>│                    │
     │ {                  │                     │                    │
     │   nickname: "新手",│                     │                    │
     │   gender: 1,       │                     │                    │
     │   appearance: "{}" │                     │                    │
     │ }                  │                     │                    │
     │                    │                     │                    │
     │                    │                     │ SessionRouter 处理 │
     │                    │                     │ 验证 uid 已认证    │
     │                    │                     │ 创建角色记录       │
     │                    │                     │                    │
     │ 9b. CreateRoleResponse                   │                    │
     │ OP_CREATE_ROLE_RES │                     │                    │
     │ (1009)             │                     │                    │
     │<─────────────────────────────────────────│                    │
     │ {                  │                     │                    │
     │   code: 0,         │                     │                    │
     │   role: {role_id: 10003, nickname: "新手", level: 1}          │
     │ }                  │                     │                    │
     │                    │                     │                    │
     │ 8c. SelectRole     │                     │                    │
     │ OP_SELECT_ROLE_REQ │                     │                    │
     │ (1006)             │                     │                    │
     │─────────────────────────────────────────>│                    │
     │ {                  │                     │                    │
     │   role_id: 10003   │                     │                    │
     │ }                  │                     │                    │
     │                    │                     │                    │
     │                    │                     │ SessionRouter 处理 │
     │                    │                     │ 验证角色属于该 uid │
     │                    │                     │ GatewaySession 设置│
     │                    │                     │ roleID = 10003     │
     │                    │                     │                    │
     │ 9c. SelectRoleResponse                   │                    │
     │ OP_SELECT_ROLE_RES │                     │                    │
     │ (1007)             │                     │                    │
     │<─────────────────────────────────────────│                    │
     │ {                  │                     │                    │
     │   code: 0,         │                     │                    │
     │   role_id: 10003   │                     │                    │
     │ }                  │                     │                    │
     │                    │                     │                    │
     │ 【现在可以发送游戏业务消息】             │                    │
     │                    │                     │                    │
     │ 10. EnterScene     │                     │                    │
     │ OP_ENTER_SCENE_REQ │                     │                    │
     │ (2000)             │                     │                    │
     │─────────────────────────────────────────>│                    │
     │ {}                 │                     │                    │
     │                    │                     │                    │
     │                    │                     │ OnMessage:         │
     │                    │                     │ 1. SessionRouter 查找失败│
     │                    │                     │ 2. 检查已选角色    │
     │                    │                     │ 3. 获取 roleID     │
     │                    │                     │ 4. ctx = WithRoleID(ctx, roleID)│
     │                    │                     │                    │
     │                    │                     │ 11. ForwardMessage │
     │                    │                     │ (gRPC)             │
     │                    │                     │ ctx 包含 roleID    │
     │                    │                     │───────────────────>│
     │                    │                     │                    │
     │                    │                     │          HandleMessage:
     │                    │                     │          1. 验证角色在线
     │                    │                     │          2. 检查状态
     │                    │                     │          3. ctx = WithRoleID()
     │                    │                     │          4. RoleRouter.Dispatch()
     │                    │                     │          5. 提取 roleID
     │                    │                     │          6. handler(ctx, roleID, req)
     │                    │                     │                    │
     │                    │                     │ 12. Response       │
     │                    │                     │<───────────────────│
     │                    │                     │ respPayload        │
     │                    │                     │                    │
     │ 13. EnterSceneResponse                   │                    │
     │ OP_ENTER_SCENE_RES │                     │                    │
     │ (2001)             │                     │                    │
     │<─────────────────────────────────────────│                    │
     │ {                  │                     │                    │
     │   code: 0,         │                     │                    │
     │   role: {role_id: xxx, nickname: "xxx", level: xx},           │
     │   scene: {map_id: 1001, pos: {x: 100, y: 0, z: 200}}          │
     │ }                  │                     │                    │
```

## 详细协议说明

### 协议总结

完整流程共 **13 个步骤**：
1. OP_LOGIN_REQ/RES - 登录获取 LoginToken
2. 连接 Gateway
3. OP_AUTH_REQ/RES - 使用 LoginToken 获取 SessionToken
4. OP_GET_ROLES_REQ/RES - 获取角色列表
5. 【情况A】有角色 → OP_SELECT_ROLE_REQ/RES - 选择第一个角色
6. 【情况B】无角色 → OP_CREATE_ROLE_REQ/RES - 创建角色 → OP_SELECT_ROLE_REQ/RES - 选择新角色
7. OP_ENTER_SCENE_REQ/RES - 进入场景（Gateway → Game）

### 阶段 1: 登录 (Login Service)

#### 1.1 OP_LOGIN_REQ (1000)

**客户端 → Login 服务**

```protobuf
message LoginRequest {
    common.LoginType login_type = 1;  // 登录类型（本地账号、第三方等）
    bytes credentials = 2;             // 凭证（序列化后）
}

// 本地登录示例
message LocalCredentials {
    string username = 1;
    string password = 2;
}
```

**处理流程：**
1. Login 服务验证凭证
2. 查询或创建用户记录
3. 生成 **LoginToken** (JWT)
4. 从服务注册中心选择可用的 Gateway
5. 返回 LoginToken 和 Gateway 地址

#### 1.2 OP_LOGIN_RES (1001)

**Login 服务 → 客户端**

```protobuf
message LoginResponse {
    string token = 1;         // LoginToken (JWT)
    uint64 uid = 2;           // 用户ID
    string nickname = 3;      // 用户昵称
    string gateway_addr = 4;  // 分配的 Gateway 地址
}
```

**Token 内容（LoginToken）：**
```json
{
  "uid": 100001,
  "exp": 1704038400,  // 过期时间（通常 30 分钟）
  "iat": 1704036600,
  "iss": "login-service"
}
```

### 阶段 2: 网关认证 (Gateway)

客户端使用 LoginToken 连接到分配的 Gateway。

#### 2.1 OP_AUTH_REQ (1002)

**客户端 → Gateway**

```protobuf
message AuthRequest {
    string login_token = 1;  // Login 服务签发的 token
}
```

**处理流程：**
1. Gateway 验证 LoginToken 签名
2. 从 Token 中提取 `uid`
3. 查询该 `uid` 下的所有角色列表（从数据库或缓存）
4. 生成 **SessionToken** (新的 JWT)
5. 创建 `GatewaySession` 对象，存储 `uid` 和 Session ID
6. 将 Session 注册到 SessionManager

**代码位置：** [handler.go:handleAuth](app/gateway/internal/handler/handler.go#L200-L250)

#### 2.2 OP_AUTH_RES (1003)

**Gateway → 客户端**

```protobuf
message AuthResponse {
    uint32 code = 1;    // 错误码
    string token = 2;   // SessionToken (Gateway 签发)
    uint64 uid = 3;     // 用户ID
}
```

**Token 内容（SessionToken）：**
```json
{
  "uid": 100001,
  "session_id": "sess_abc123",
  "exp": 1704124800,  // 过期时间（通常 24 小时）
  "iat": 1704038400,
  "iss": "gateway-service"
}
```

**此时 GatewaySession 状态：**
```go
GatewaySession {
    uid: 100001,
    roleID: 0,           // 尚未选择角色
    sessionID: "sess_abc123",
    authenticated: true,
    roleSelected: false  // 关键标志！
}
```

### 阶段 3: 获取角色列表 (Gateway)

#### 3.1 OP_GET_ROLES_REQ (1010)

**客户端 → Gateway**

```protobuf
message GetRolesRequest {
    // 无需参数，从 Session 中获取 uid
}
```

**处理流程：**
1. SessionRouter 接收请求
2. 从 Session 获取 `uid`（已认证）
3. 查询该 `uid` 下的所有角色（从数据库或缓存）
4. 返回角色列表

**代码位置：** 需要在 Gateway handler 中实现 `handleGetRoles`

#### 3.2 OP_GET_ROLES_RES (1011)

**Gateway → 客户端**

```protobuf
message GetRolesResponse {
    uint32 code = 1;              // 错误码
    repeated RoleInfo roles = 2;  // 该账号下的所有角色
}

message RoleInfo {
    int64 role_id = 1;
    string nickname = 2;
    int32 level = 3;
    int32 status = 4;  // 0=正常, 1=封禁
}
```

**客户端逻辑：**
```javascript
// 伪代码
if (roles.length > 0) {
    // 有角色，默认选择第一个
    selectRole(roles[0].role_id);
} else {
    // 没有角色，创建新角色
    createRole("新角色", gender, appearance);
}
```

### 阶段 4: 角色操作 (Gateway)

#### 4.1 创建角色（可选）

如果 `roles` 列表为空，客户端需要先创建角色。

##### OP_CREATE_ROLE_REQ (1008)

**客户端 → Gateway**

```protobuf
message CreateRoleRequest {
    string nickname = 1;
    int32 gender = 2;
    string appearance = 3;  // 外观数据
}
```

**处理流程：**
1. SessionRouter 接收请求
2. 从 Session 获取 `uid`
3. 验证昵称合法性（长度、敏感词等）
4. 在数据库创建角色记录
5. 返回新角色信息

**代码位置：** [handler.go:handleCreateRole](app/gateway/internal/handler/handler.go)

##### OP_CREATE_ROLE_RES (1009)

**Gateway → 客户端**

```protobuf
message CreateRoleResponse {
    uint32 code = 1;
    RoleInfo role = 2;  // 创建成功的角色
}
```

#### 4.2 选择角色（必须）

##### OP_SELECT_ROLE_REQ (1006)

**客户端 → Gateway**

```protobuf
message SelectRoleRequest {
    int64 role_id = 1;
}
```

**处理流程：**
1. SessionRouter 接收请求
2. 从 Session 获取 `uid`
3. 验证 `role_id` 属于该 `uid`
4. 验证角色状态（是否封禁）
5. **关键步骤：** 在 `GatewaySession` 中设置 `roleID`
6. 标记 `roleSelected = true`

**代码位置：** [handler.go:handleSelectRole](app/gateway/internal/handler/handler.go)

##### OP_SELECT_ROLE_RES (1007)

**Gateway → 客户端**

```protobuf
message SelectRoleResponse {
    uint32 code = 1;
    int64 role_id = 2;
}
```

**此时 GatewaySession 状态：**
```go
GatewaySession {
    uid: 100001,
    roleID: 10001,       // ✅ 已设置！
    sessionID: "sess_abc123",
    authenticated: true,
    roleSelected: true   // ✅ 已标记！
}
```

### 阶段 5: 游戏消息转发 (Gateway → Game)

此时客户端可以发送游戏业务消息。

#### 5.1 OP_ENTER_SCENE_REQ (2000)

**客户端 → Gateway**

```protobuf
message EnterSceneRequest {
    // 预留扩展（通常进入默认场景）
}
```

**Gateway 处理流程（OnMessage）：**

**代码位置：** [handler.go:OnMessage](app/gateway/internal/handler/handler.go#L105-L131)

```go
func (h *GatewayHandler) OnMessage(s session.Session, env *common.Envelope) {
    op := env.Header.Op
    payload := env.Payload

    // 1. 优先尝试 SessionRouter（Gateway 特定消息）
    respOp, respPayload, err := h.sessionRouter.Dispatch(s.Context(), s, op, payload)
    if err == nil {
        // SessionRouter 处理成功，直接返回
        s.Send(...)
        return
    }

    // 2. SessionRouter 未找到 → 业务消息，转发到 Game

    // 2.1 检查是否已选择角色
    gwSess, ok := h.sessMgr.Get(s.ID())
    if !ok || !gwSess.IsRoleSelected() {
        h.logger.Warn("message before role selection")
        return
    }

    // 2.2 【关键】从 GatewaySession 提取 roleID
    roleID := gwSess.GetRoleID()  // → 10001

    // 2.3 【关键】将 roleID 放入 Context
    ctx := router.WithRoleID(s.Context(), roleID)

    // 2.4 使用 Processor 转发到 Game（通过 gRPC）
    respOp, respPayload, err = h.processor.Process(ctx, op, payload)

    // 2.5 发送响应
    s.Send(...)
}
```

**Context 传递的 roleID：**
```go
ctx := context.WithValue(ctx, router.RoleIDKey, int64(10001))
```

#### 5.2 Gateway → Game (gRPC 转发)

**gRPC 调用：**

```protobuf
// 假设的 proto 定义（待实现）
message ForwardMessageRequest {
    int64 role_id = 1;      // roleID（也在 Context 中）
    uint32 op_code = 2;     // 2000 (OP_ENTER_SCENE_REQ)
    bytes payload = 3;      // EnterSceneRequest 序列化
}
```

**注意：** roleID 既在 Context 中传递，也可以在消息体中。但 **Game 服务应优先信任 Context 中的 roleID**，因为它由 Gateway 控制，无法被客户端伪造。

#### 5.3 Game 服务处理

**代码位置：** [message_service.go:HandleMessage](app/game/internal/service/message_service.go#L59-L109)

```go
func (s *MessageService) HandleMessage(
    ctx context.Context,
    roleID int64,
    opCode uint32,
    payload []byte,
) ([]byte, error) {
    // 1. 验证角色是否在线
    role, ok := s.roleMgr.GetRole(roleID)
    if !ok {
        return nil, fmt.Errorf("role not online")
    }

    // 2. 检查角色状态
    if role.IsBanned() {
        return nil, fmt.Errorf("role is banned")
    }

    // 3. 【关键】将 roleID 放入 Context
    ctx = router.WithRoleID(ctx, roleID)

    // 4. 使用 RoleRouter 路由到具体 handler
    _, respPayload, err := s.roleRouter.Dispatch(ctx, opCode, payload)

    return respPayload, err
}
```

**RoleRouter 内部：**

**代码位置：** [role_router.go:RegisterHandler](app/game/internal/router/role_router.go#L20-L46)

```go
// 注册时的包装函数
func RegisterHandler[...](
    rr *RoleRouter,
    reqOp uint32,
    respOp uint32,
    handler func(ctx context.Context, roleID int64, req PReq) (PResp, error),
) {
    router.RegisterHandler(rr.router, reqOp, respOp,
        func(ctx context.Context, req PReq) (PResp, error) {
            // 1. 从 Context 提取 roleID
            roleID, ok := router.GetRoleIDFromContext(ctx)
            if !ok {
                return nil, fmt.Errorf("role_id not found in context")
            }

            // 2. 调用业务 handler（自动传入 roleID）
            return handler(ctx, roleID, req)
        })
}
```

**实际的业务 handler（假设已注册）：**

```go
func (s *MessageService) handleEnterScene(
    ctx context.Context,
    roleID int64,  // ← 自动注入！= 10001
    req *scene.EnterSceneRequest,
) (*scene.EnterSceneResponse, error) {
    s.logger.Info("role entering scene", "role_id", roleID)

    // 1. 加载角色数据
    role, ok := s.roleMgr.GetRole(roleID)
    if !ok {
        return &scene.EnterSceneResponse{
            Code: common.ErrCode_ERR_INTERNAL,
        }, nil
    }

    // 2. 加载或初始化场景数据
    sceneInfo := &scene.SceneInfo{
        MapId: 1001,
        Pos: &scene.Position{
            X: 100.0,
            Y: 0.0,
            Z: 200.0,
            Rotation: 0.0,
        },
    }

    // 3. 返回响应
    return &scene.EnterSceneResponse{
        Code: common.ErrCode_ERR_SUCCESS,
        Role: &scene.RoleInfo{
            RoleId:   roleID,
            Nickname: role.Nickname,
            Level:    role.Level,
        },
        Scene: sceneInfo,
    }, nil
}
```

#### 5.4 OP_ENTER_SCENE_RES (2001)

**Game → Gateway → 客户端**

```protobuf
message EnterSceneResponse {
    common.ErrCode code = 1;
    RoleInfo role = 2;
    SceneInfo scene = 3;
}

message RoleInfo {
    int64 role_id = 1;
    string nickname = 2;
    string avatar_url = 3;
    int32 level = 4;
    int32 vip_exp = 5;
}

message SceneInfo {
    int32 map_id = 1;
    Position pos = 2;
}

message Position {
    float x = 1;
    float y = 2;
    float z = 3;
    float rotation = 4;
}
```

## 关键概念总结

### Token 流转

1. **LoginToken** (Login 服务签发)
   - 用途：首次网关认证
   - 有效期：30 分钟（短期）
   - 包含：uid
   - 使用：仅用于 OP_AUTH_REQ

2. **SessionToken** (Gateway 签发)
   - 用途：维持会话连接
   - 有效期：24 小时（长期）
   - 包含：uid, session_id
   - 使用：重连时使用 (OP_RECONNECT_REQ)

### RoleID 流转

```
┌─────────────────────────────────────────────────────────────────┐
│ RoleID 的完整生命周期                                            │
└─────────────────────────────────────────────────────────────────┘

1. 客户端发送: SelectRoleRequest{role_id: 10001}
   ↓
2. Gateway handleSelectRole():
   - 验证 role_id 属于 uid
   - gwSession.SetRoleID(10001)  ← 存储在 GatewaySession
   - gwSession.SetRoleSelected(true)
   ↓
3. 客户端发送业务消息: EnterSceneRequest{}
   ↓
4. Gateway OnMessage():
   - roleID := gwSession.GetRoleID()  → 10001
   - ctx := router.WithRoleID(ctx, roleID)  ← 放入 Context
   - processor.Process(ctx, op, payload)
   ↓
5. Game HandleMessage():
   - 接收 ctx（包含 roleID）
   - ctx = router.WithRoleID(ctx, roleID)  ← 确保 Context 中有 roleID
   - roleRouter.Dispatch(ctx, op, payload)
   ↓
6. RoleRouter RegisterHandler 包装:
   - roleID, ok := router.GetRoleIDFromContext(ctx)  → 10001
   - handler(ctx, roleID, req)  ← 自动注入
   ↓
7. 业务 handler:
   - func handleEnterScene(ctx, roleID, req)  ← roleID = 10001
```

### Router 分层

| Router | 使用位置 | Handler 签名 | 用途 |
|--------|---------|-------------|------|
| **SessionRouter** | Gateway | `(ctx, session, *Req) (*Resp, error)` | Gateway 特定消息（认证、角色选择） |
| **基础 Router** | 框架层 | `(ctx, *Req) (*Resp, error)` | 通用路由能力 |
| **RoleRouter** | Game | `(ctx, roleID, *Req) (*Resp, error)` | Game 业务消息（自动注入 roleID） |

### 消息分类

**Gateway 特定消息（SessionRouter 处理）：**
- OP_AUTH_REQ/RES (1002/1003)
- OP_RECONNECT_REQ/RES (1004/1005)
- OP_SELECT_ROLE_REQ/RES (1006/1007)
- OP_CREATE_ROLE_REQ/RES (1008/1009)

**Game 业务消息（转发到 Game）：**
- OP_ENTER_SCENE_REQ/RES (2000/2001)
- 所有其他游戏逻辑消息（战斗、背包、社交等）

### 安全性保证

1. **LoginToken 验证**：Gateway 验证 Login 服务签名
2. **角色归属验证**：SelectRole 时验证 role_id 属于 uid
3. **RoleID 防伪造**：
   - RoleID 由 Gateway 从 GatewaySession 提取
   - 通过 Context 传递到 Game
   - 客户端无法在消息体中伪造 roleID
4. **状态检查**：
   - Gateway 检查 `roleSelected` 标志
   - Game 检查角色是否在线、是否封禁

## 完整 OpCode 列表

| OpCode | 值 | 服务 | 说明 |
|--------|---|------|-----|
| OP_LOGIN_REQ | 1000 | Login | 登录请求 |
| OP_LOGIN_RES | 1001 | Login | 登录响应 |
| OP_AUTH_REQ | 1002 | Gateway | 网关认证请求 |
| OP_AUTH_RES | 1003 | Gateway | 网关认证响应 |
| OP_RECONNECT_REQ | 1004 | Gateway | 重连请求 |
| OP_RECONNECT_RES | 1005 | Gateway | 重连响应 |
| OP_SELECT_ROLE_REQ | 1006 | Gateway | 选择角色请求 |
| OP_SELECT_ROLE_RES | 1007 | Gateway | 选择角色响应 |
| OP_CREATE_ROLE_REQ | 1008 | Gateway | 创建角色请求 |
| OP_CREATE_ROLE_RES | 1009 | Gateway | 创建角色响应 |
| OP_GET_ROLES_REQ | 1010 | Gateway | 获取角色列表请求 |
| OP_GET_ROLES_RES | 1011 | Gateway | 获取角色列表响应 |
| OP_ENTER_SCENE_REQ | 2000 | Game | 进入场景请求 |
| OP_ENTER_SCENE_RES | 2001 | Game | 进入场景响应 |

## 参考文件

- [op_code.proto](../xDooria-proto-api/op_code.proto) - 所有 OpCode 定义
- [login.proto](../xDooria-proto-api/login/login.proto) - Login 服务协议
- [gateway.proto](../xDooria-proto-api/gateway/gateway.proto) - Gateway 服务协议
- [scene.proto](../xDooria-proto-api/scene/scene.proto) - 场景相关协议
- [handler.go](../xDooria/app/gateway/internal/handler/handler.go) - Gateway 消息处理
- [message_service.go](../xDooria/app/game/internal/service/message_service.go) - Game 消息服务
- [role_router.go](../xDooria/app/game/internal/router/role_router.go) - RoleRouter 实现
- [context.go](../xDooria/pkg/router/context.go) - Context RoleID 工具

## 下一步扩展

后续可以添加更多游戏业务协议：

- **战斗系统**：OP_ATTACK_REQ/RES, OP_USE_SKILL_REQ/RES
- **背包系统**：OP_USE_ITEM_REQ/RES, OP_EQUIP_REQ/RES
- **社交系统**：OP_CHAT_REQ/RES, OP_ADD_FRIEND_REQ/RES
- **任务系统**：OP_ACCEPT_QUEST_REQ/RES, OP_SUBMIT_QUEST_REQ/RES

所有这些协议都遵循相同的流程：
1. 客户端 → Gateway
2. Gateway 提取 roleID，放入 Context
3. Gateway → Game (gRPC)
4. Game RoleRouter 自动注入 roleID
5. 业务 handler 处理
6. 原路返回响应
