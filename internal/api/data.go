package api

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/zl14229/pokemon-shiny-tracker/internal/backup"
	"gorm.io/gorm"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func dataRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	v1 = v1.Group("", localOrigin)
	for _, kind := range []string{"backup", "export"} {
		v1.GET("/data/"+kind, func(c *gin.Context) {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
			defer cancel()
			_ = http.NewResponseController(c.Writer).SetWriteDeadline(time.Now().Add(2 * time.Minute))
			var list []struct {
				Name string
				File string
			}
			if e := db.WithContext(ctx).Raw("PRAGMA database_list").Scan(&list).Error; e != nil {
				storageError(c, e)
				return
			}
			source := ""
			for _, r := range list {
				if r.Name == "main" {
					source = r.File
				}
			}
			dir, e := os.MkdirTemp("", "pst-download-")
			if e != nil {
				storageError(c, e)
				return
			}
			defer os.RemoveAll(dir)
			name := "shiny-backup.zip"
			if kind == "export" {
				name = "shiny-data.json"
			}
			path := filepath.Join(dir, name)
			if kind == "backup" {
				_, e = backup.Backup(ctx, source, path)
			} else {
				e = backup.Export(ctx, source, path)
			}
			if e != nil {
				storageError(c, e)
				return
			}
			c.Header("X-Content-Type-Options", "nosniff")
			c.FileAttachment(path, name)
		})
	}
}
