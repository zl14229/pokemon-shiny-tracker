package api

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"github.com/zl14229/pokemon-shiny-tracker/internal/service"
	"github.com/zl14229/pokemon-shiny-tracker/internal/workbook"
	"gorm.io/gorm"
)

func uploadWorkbook(c *gin.Context) ([]byte, bool) {
	media, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || media != "multipart/form-data" {
		abort(c, 415, "unsupported_media_type", "请上传XLSX文件")
		return nil, false
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, workbook.MaxFile+65536)
	if err := c.Request.ParseMultipartForm(workbook.MaxFile + 65536); err != nil {
		abort(c, 413, "request_too_large", "文件过大或格式无效")
		return nil, false
	}
	defer c.Request.MultipartForm.RemoveAll()
	if len(c.Request.MultipartForm.File["file"]) != 1 {
		abort(c, 400, "invalid_request", "须上传一个文件")
		return nil, false
	}
	f, err := c.Request.MultipartForm.File["file"][0].Open()
	if err != nil {
		storageError(c, err)
		return nil, false
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, workbook.MaxFile+1))
	if err != nil || len(data) > workbook.MaxFile {
		abort(c, 413, "request_too_large", "XLSX上限8MiB")
		return nil, false
	}
	return data, true
}
func importRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	v1 = v1.Group("", localOrigin)
	s := service.NewImports(db)
	v1.POST("/imports/inspect", func(c *gin.Context) {
		data, ok := uploadWorkbook(c)
		if !ok {
			return
		}
		book, err := workbook.Read(data)
		if err != nil {
			abort(c, 400, "invalid_request", err.Error())
			return
		}
		c.JSON(200, gin.H{"data": book})
	})
	v1.POST("/imports/preview", func(c *gin.Context) {
		data, ok := uploadWorkbook(c)
		if !ok {
			return
		}
		var m service.ImportMapping
		if err := json.Unmarshal([]byte(c.PostForm("mapping")), &m); err != nil {
			abort(c, 400, "invalid_request", "映射JSON无效")
			return
		}
		p, err := s.Preview(c.Request.Context(), data, m)
		if err != nil {
			personalError(c, err)
			return
		}
		c.JSON(200, gin.H{"data": p})
	})
	v1.POST("/imports/confirm", func(c *gin.Context) {
		in, ok := body[service.ImportConfirm](c)
		if !ok {
			return
		}
		r, err := s.Confirm(c.Request.Context(), *in)
		if err != nil {
			personalError(c, err)
			return
		}
		c.JSON(200, gin.H{"data": r})
	})
	v1.POST("/imports/discard", func(c *gin.Context) {
		in, ok := body[struct {
			Token string `json:"token"`
		}](c)
		if !ok {
			return
		}
		s.Discard(in.Token)
		c.JSON(200, gin.H{"data": true})
	})
	v1.GET("/imports/batches/:id", func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		var batch model.ImportBatch
		if err := db.WithContext(c.Request.Context()).First(&batch, id).Error; err != nil {
			storageError(c, err)
			return
		}
		rows := []model.ImportRow{}
		if err := db.WithContext(c.Request.Context()).Where("import_batch_id = ?", id).Order("row_number").Find(&rows).Error; err != nil {
			storageError(c, err)
			return
		}
		c.JSON(200, gin.H{"data": service.ImportResult{Batch: batch, Rows: rows}})
	})
	v1.GET("/imports/batches", func(c *gin.Context) {
		rows := []model.ImportBatch{}
		err := db.WithContext(c.Request.Context()).Order("id DESC").Limit(100).Find(&rows).Error
		if err != nil {
			storageError(c, err)
			return
		}
		c.JSON(200, gin.H{"data": rows})
	})
}
