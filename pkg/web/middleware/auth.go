package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/security"
	"github.com/lk2023060901/xdooria/pkg/web/errors"
	"go.uber.org/zap"
)

const (
	// ClaimsKey Context 中存储 Claims 的 key
	ClaimsKey = "jwt_claims"
	// LoggerKey Context 中存储带用户信息的 Logger 的 key
	LoggerKey = "request_logger"
)

// AuthConfig 认证配置
type AuthConfig struct {
	// JWTManager JWT 管理器
	JWTManager *security.JWTManager
	// SkipPaths 跳过验证的路径
	SkipPaths []string
	// SkipPrefixes 跳过验证的路径前缀
	SkipPrefixes []string
	// ErrorHandler 自定义错误处理
	ErrorHandler func(*gin.Context, error)
	// Logger 用于创建带用户信息的 logger
	Logger logger.Logger
	// UserIDKey Payload 中用户 ID 的 key（默认 "uid"）
	UserIDKey string
	// UsernameKey Payload 中用户名的 key（默认 "username"）
	UsernameKey string
	// RolesKey Payload 中角色列表的 key（默认 "roles"）
	RolesKey string
	// PermissionsKey Payload 中权限列表的 key（默认 "permissions"）
	PermissionsKey string
}

// DefaultAuthConfig 返回默认配置
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		UserIDKey:      "uid",
		UsernameKey:    "username",
		RolesKey:       "roles",
		PermissionsKey: "permissions",
	}
}

// Auth JWT 认证中间件
func Auth(cfg *AuthConfig) gin.HandlerFunc {
	if cfg.UserIDKey == "" {
		cfg.UserIDKey = "uid"
	}
	if cfg.UsernameKey == "" {
		cfg.UsernameKey = "username"
	}
	if cfg.RolesKey == "" {
		cfg.RolesKey = "roles"
	}
	if cfg.PermissionsKey == "" {
		cfg.PermissionsKey = "permissions"
	}

	skipPaths := make(map[string]struct{})
	for _, path := range cfg.SkipPaths {
		skipPaths[path] = struct{}{}
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 检查跳过路径
		if _, skip := skipPaths[path]; skip {
			c.Next()
			return
		}

		// 检查跳过前缀
		for _, prefix := range cfg.SkipPrefixes {
			if strings.HasPrefix(path, prefix) {
				c.Next()
				return
			}
		}

		// 提取 Token
		token := extractToken(c, cfg.JWTManager.GetConfig())
		if token == "" {
			handleAuthError(c, cfg, security.ErrTokenMissing)
			return
		}

		// 验证 Token
		claims, err := cfg.JWTManager.ValidateToken(token)
		if err != nil {
			handleAuthError(c, cfg, err)
			return
		}

		// Claims 存入 Context
		c.Set(ClaimsKey, claims)

		// 创建带用户信息的 Logger 并存入 Context
		if cfg.Logger != nil {
			userID := getPayloadString(claims, cfg.UserIDKey)
			username := getPayloadString(claims, cfg.UsernameKey)
			userLogger := cfg.Logger.WithFields(
				zap.String("uid", userID),
				zap.String("username", username),
			)
			c.Set(LoggerKey, userLogger)
		}

		c.Next()
	}
}

// extractToken 从请求中提取 Token
func extractToken(c *gin.Context, cfg *security.JWTConfig) string {
	header := c.GetHeader(cfg.HeaderName)
	if header == "" {
		return ""
	}

	if cfg.TokenPrefix != "" && strings.HasPrefix(header, cfg.TokenPrefix) {
		return strings.TrimPrefix(header, cfg.TokenPrefix)
	}

	return header
}

// handleAuthError 处理认证错误
func handleAuthError(c *gin.Context, cfg *AuthConfig, err error) {
	if cfg.ErrorHandler != nil {
		cfg.ErrorHandler(c, err)
		return
	}

	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"code":    errors.CodeUnAuthorized,
		"message": err.Error(),
		"data":    nil,
	})
}

// getPayloadString 从 Claims.Payload 获取字符串值
func getPayloadString(claims *security.Claims, key string) string {
	if v := claims.Get(key); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// getPayloadStringSlice 从 Claims.Payload 获取字符串切片
func getPayloadStringSlice(claims *security.Claims, key string) []string {
	if v := claims.Get(key); v != nil {
		switch val := v.(type) {
		case []string:
			return val
		case []any:
			result := make([]string, 0, len(val))
			for _, item := range val {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return nil
}

// RequireRoles 角色检查中间件（需要所有角色）
func RequireRoles(rolesKey string, roles ...string) gin.HandlerFunc {
	if rolesKey == "" {
		rolesKey = "roles"
	}
	return func(c *gin.Context) {
		claims, ok := GetClaims(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    errors.CodeUnAuthorized,
				"message": "unauthorized",
				"data":    nil,
			})
			return
		}

		userRoles := getPayloadStringSlice(claims, rolesKey)
		if !hasAllStrings(userRoles, roles) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    errors.CodeForbidden,
				"message": "forbidden: insufficient roles",
				"data":    nil,
			})
			return
		}

		c.Next()
	}
}

// RequireAnyRole 角色检查中间件（需要任意一个角色）
func RequireAnyRole(rolesKey string, roles ...string) gin.HandlerFunc {
	if rolesKey == "" {
		rolesKey = "roles"
	}
	return func(c *gin.Context) {
		claims, ok := GetClaims(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    errors.CodeUnAuthorized,
				"message": "unauthorized",
				"data":    nil,
			})
			return
		}

		userRoles := getPayloadStringSlice(claims, rolesKey)
		if !hasAnyString(userRoles, roles) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    errors.CodeForbidden,
				"message": "forbidden: insufficient roles",
				"data":    nil,
			})
			return
		}

		c.Next()
	}
}

// RequirePermissions 权限检查中间件（需要所有权限）
func RequirePermissions(permissionsKey string, permissions ...string) gin.HandlerFunc {
	if permissionsKey == "" {
		permissionsKey = "permissions"
	}
	return func(c *gin.Context) {
		claims, ok := GetClaims(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    errors.CodeUnAuthorized,
				"message": "unauthorized",
				"data":    nil,
			})
			return
		}

		userPerms := getPayloadStringSlice(claims, permissionsKey)
		if !hasAllStrings(userPerms, permissions) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    errors.CodeForbidden,
				"message": "forbidden: insufficient permissions",
				"data":    nil,
			})
			return
		}

		c.Next()
	}
}

// RequireAnyPermission 权限检查中间件（需要任意一个权限）
func RequireAnyPermission(permissionsKey string, permissions ...string) gin.HandlerFunc {
	if permissionsKey == "" {
		permissionsKey = "permissions"
	}
	return func(c *gin.Context) {
		claims, ok := GetClaims(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    errors.CodeUnAuthorized,
				"message": "unauthorized",
				"data":    nil,
			})
			return
		}

		userPerms := getPayloadStringSlice(claims, permissionsKey)
		if !hasAnyString(userPerms, permissions) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    errors.CodeForbidden,
				"message": "forbidden: insufficient permissions",
				"data":    nil,
			})
			return
		}

		c.Next()
	}
}

// hasAllStrings 检查是否包含所有目标字符串
func hasAllStrings(slice []string, targets []string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}
	for _, t := range targets {
		if _, ok := set[t]; !ok {
			return false
		}
	}
	return true
}

// hasAnyString 检查是否包含任意一个目标字符串
func hasAnyString(slice []string, targets []string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}
	for _, t := range targets {
		if _, ok := set[t]; ok {
			return true
		}
	}
	return false
}

// GetClaims 从 Context 获取 Claims
func GetClaims(c *gin.Context) (*security.Claims, bool) {
	if claims, exists := c.Get(ClaimsKey); exists {
		return claims.(*security.Claims), true
	}
	return nil, false
}

// MustGetClaims 从 Context 获取 Claims（不存在则 panic）
func MustGetClaims(c *gin.Context) *security.Claims {
	claims, ok := GetClaims(c)
	if !ok {
		panic("claims not found in context")
	}
	return claims
}

// GetUserID 从 Context 获取用户 ID（使用默认 key "uid"）
func GetUserID(c *gin.Context) string {
	return GetUserIDWithKey(c, "uid")
}

// GetUserIDWithKey 从 Context 获取用户 ID（指定 key）
func GetUserIDWithKey(c *gin.Context, key string) string {
	if claims, ok := GetClaims(c); ok {
		return getPayloadString(claims, key)
	}
	return ""
}

// GetLogger 从 Context 获取带用户信息的 Logger
func GetLogger(c *gin.Context) logger.Logger {
	if l, exists := c.Get(LoggerKey); exists {
		return l.(logger.Logger)
	}
	return nil
}

// MustGetLogger 从 Context 获取 Logger（不存在则 panic）
func MustGetLogger(c *gin.Context) logger.Logger {
	l := GetLogger(c)
	if l == nil {
		panic("logger not found in context")
	}
	return l
}
