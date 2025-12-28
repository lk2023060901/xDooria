package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/xdooria/app/portal/internal/client"
	"github.com/lk2023060901/xdooria/app/portal/internal/metrics"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/web"
	api "github.com/lk2023060901/xdooria-proto-api"
	pb "github.com/lk2023060901/xdooria-proto-common"
)

// LoginHandler 登录处理器
type LoginHandler struct {
	client  *client.LoginClient
	metrics *metrics.PortalMetrics
	logger  logger.Logger
}

// NewLoginHandler 创建登录处理器
func NewLoginHandler(c *client.LoginClient, m *metrics.PortalMetrics, l logger.Logger) *LoginHandler {
	return &LoginHandler{
		client:  c,
		metrics: m,
		logger:  l.Named("handler.login"),
	}
}

// LoginRequest HTTP 登录请求
type LoginRequest struct {
	LoginType   int32             `json:"login_type" binding:"required"`
	Credentials map[string]string `json:"credentials" binding:"required"`
}

// LoginResponse HTTP 登录响应
type LoginResponse struct {
	Token    string `json:"token"`
	UID      uint64 `json:"uid"`
	Nickname string `json:"nickname"`
}

// Register 注册路由
func (h *LoginHandler) Register(r *gin.Engine) {
	api := r.Group("/api/v1")
	{
		api.POST("/login", h.Login)
	}
}

// Login 登录接口
// @Summary 用户登录
// @Description 支持多种登录方式
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "登录请求"
// @Success 200 {object} web.Response{data=LoginResponse}
// @Failure 400 {object} web.Response
// @Failure 500 {object} web.Response
// @Router /api/v1/login [post]
func (h *LoginHandler) Login(c *gin.Context) {
	start := time.Now()

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid login request", "error", err)
		web.Error(c, http.StatusBadRequest, 400, "invalid request: "+err.Error())
		return
	}

	// 序列化凭证
	credentials, err := json.Marshal(req.Credentials)
	if err != nil {
		h.logger.Warn("failed to marshal credentials", "error", err)
		web.Error(c, http.StatusBadRequest, 400, "invalid credentials")
		return
	}

	// 构建 gRPC 请求
	grpcReq := &api.LoginRequest{
		LoginType:   pb.LoginType(req.LoginType),
		Credentials: credentials,
	}

	// 调用 Login 服务
	resp, err := h.client.Login(c.Request.Context(), grpcReq)
	if err != nil {
		duration := time.Since(start).Seconds()
		h.metrics.RecordRequest("login", false, duration)
		h.logger.Error("login failed", "error", err)
		web.Error(c, http.StatusInternalServerError, 500, "login failed: "+err.Error())
		return
	}

	// 记录成功指标
	duration := time.Since(start).Seconds()
	h.metrics.RecordRequest("login", true, duration)

	// 返回响应
	web.Success(c, LoginResponse{
		Token:    resp.Token,
		UID:      resp.Uid,
		Nickname: resp.Nickname,
	})
}
