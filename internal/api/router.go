// Package api defines the versioned HTTP interface. Health needs no database.
package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// NewRouter installs process health and consistent JSON error handling.
func NewRouter(db *gorm.DB) *gin.Engine {
	router := gin.New()
	_ = router.SetTrustedProxies(nil)
	router.HandleMethodNotAllowed = true
	router.RedirectTrailingSlash = false
	router.Use(handleErrors())
	router.NoRoute(func(c *gin.Context) {
		abort(c, http.StatusNotFound, "not_found", "请求的接口不存在")
	})
	router.NoMethod(func(c *gin.Context) {
		abort(c, http.StatusMethodNotAllowed, "method_not_allowed", "请求方法不受支持")
	})
	v1 := router.Group("/api/v1")
	v1.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "ok"}})
	})
	if db != nil {
		referenceRoutes(v1, db)
		huntRoutes(v1, db)
	}
	return router
}

func abort(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{Error: ErrorDetail{Code: code, Message: message}})
}

// Internal error and panic details are never returned or logged as raw input.
// Once headers/body are committed, abort without appending a second response.
func handleErrors() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Error("request panic recovered")
				c.Abort()
				if !c.Writer.Written() {
					abort(c, http.StatusInternalServerError, "internal_error", "服务暂时不可用，请稍后重试")
				}
			}
		}()
		c.Next()
		if len(c.Errors) > 0 {
			slog.Error("request failed")
			c.Abort()
			if !c.Writer.Written() {
				abort(c, http.StatusInternalServerError, "internal_error", "服务暂时不可用，请稍后重试")
			}
		}
	}
}
