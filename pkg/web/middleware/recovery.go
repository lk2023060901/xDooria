package middleware

import (
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// Recovery 适配 pkg/logger 的异常恢复中间件
func Recovery(l logger.Logger, stack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 检查连接是否断开
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				if brokenPipe {
					l.Error("http broken pipe",
						"error", err,
						"request", string(httpRequest),
					)
					c.Error(err.(error))
					c.Abort()
					return
				}

				if stack {
					l.Error("http recovery from panic",
						"error", err,
						"request", string(httpRequest),
					)
				} else {
					l.Error("http recovery from panic",
						"error", err,
						"request", string(httpRequest),
					)
				}
				
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
