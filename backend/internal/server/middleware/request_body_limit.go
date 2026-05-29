package middleware

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/gin-gonic/gin"
)

// RequestBodyLimit 使用 MaxBytesReader 限制请求体大小。
func RequestBodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request = httputil.ApplyRequestBodyLimit(c.Writer, c.Request, maxBytes)
		c.Next()
	}
}
