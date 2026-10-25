package api

import (
	"github.com/gin-gonic/gin"
	"github.com/zl14229/pokemon-shiny-tracker/internal/files"
	"github.com/zl14229/pokemon-shiny-tracker/internal/service"
	"gorm.io/gorm"
	"io"
	"mime"
	"net/http"
)

func attachmentRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	v1 = v1.Group("", localOrigin)
	s := service.Attachments{DB: db}
	v1.GET("/shiny-records/:id/attachments", func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		rows, e := s.List(c.Request.Context(), id)
		if e != nil {
			personalError(c, e)
			return
		}
		c.JSON(200, gin.H{"data": rows})
	})
	v1.POST("/shiny-records/:id/attachments", func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, files.MaxSize+65536)
		reader, e := c.Request.MultipartReader()
		if e != nil {
			abort(c, 415, "unsupported_media_type", "请上传multipart文件")
			return
		}
		var name string
		var data []byte
		count := 0
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				abort(c, 413, "request_too_large", "上传内容过大或无效")
				return
			}
			_, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
			if err != nil || params["name"] != "file" || !files.ValidName(params["filename"]) {
				_ = part.Close()
				abort(c, 400, "invalid_request", "附件名称无效")
				return
			}
			count++
			if count > 1 {
				_ = part.Close()
				abort(c, 400, "invalid_request", "每次只能上传一个文件")
				return
			}
			name = params["filename"]
			data, err = io.ReadAll(io.LimitReader(part, files.MaxSize+1))
			_ = part.Close()
			if err != nil || len(data) > files.MaxSize {
				abort(c, 413, "request_too_large", "附件上限10MiB")
				return
			}
		}
		if count != 1 {
			abort(c, 400, "invalid_request", "请选择一个文件")
			return
		}
		a, e := s.Upload(c.Request.Context(), id, name, data)
		if e != nil {
			personalError(c, e)
			return
		}
		c.JSON(201, gin.H{"data": a})
	})
	v1.POST("/shiny-records/:id/links", func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		in, ok := body[service.LinkInput](c)
		if !ok {
			return
		}
		a, e := s.Link(c.Request.Context(), id, *in)
		if e != nil {
			personalError(c, e)
			return
		}
		c.JSON(201, gin.H{"data": a})
	})
	v1.GET("/attachments/:id/content", func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		a, data, e := s.Content(c.Request.Context(), id)
		if e != nil {
			personalError(c, e)
			return
		}
		kind := "attachment"
		if a.MediaType == "image/png" || a.MediaType == "image/jpeg" || a.MediaType == "image/gif" {
			kind = "inline"
		}
		c.Header("Content-Disposition", mime.FormatMediaType(kind, map[string]string{"filename": a.Name}))
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "no-referrer")
		c.Data(200, a.MediaType, data)
	})
	v1.DELETE("/attachments/:id", func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		if e := s.Delete(c.Request.Context(), id); e != nil {
			personalError(c, e)
			return
		}
		c.JSON(200, gin.H{"data": true})
	})
}
