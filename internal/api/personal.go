package api

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zl14229/pokemon-shiny-tracker/internal/service"
	"gorm.io/gorm"
)

// Decode a bounded, explicit DTO. null/unknown fields/trailing values are invalid.
func body[T any](c *gin.Context) (*T, bool) {
	media, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || media != "application/json" {
		abort(c, 415, "unsupported_media_type", "请使用 application/json")
		return nil, false
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	var input *T
	if err = decoder.Decode(&input); err == nil && input != nil {
		var extra any
		err = decoder.Decode(&extra)
		if err == io.EOF {
			return input, true
		}
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		abort(c, 413, "request_too_large", "请求内容过大")
	} else {
		abort(c, 400, "invalid_request", "请求内容不符合 JSON 字段要求")
	}
	return nil, false
}
func personalError(c *gin.Context, err error) {
	var problem *service.Error
	if errors.As(err, &problem) {
		status := 400
		if problem.Code == "conflict" {
			status = 409
		}
		abort(c, status, problem.Code, problem.Message)
		return
	}
	storageError(c, err)
}
func huntRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	s := service.Hunts{DB: db}
	v1.GET("/hunts", func(c *gin.Context) {
		rows, total, err := s.List(c.Request.Context())
		if err != nil {
			personalError(c, err)
			return
		}
		c.JSON(200, gin.H{"data": rows, "meta": gin.H{"total": total, "limit": 100}})
	})
	v1.POST("/hunts", func(c *gin.Context) {
		in, ok := body[service.HuntInput](c)
		if !ok {
			return
		}
		row, err := s.Create(c.Request.Context(), *in)
		if err != nil {
			personalError(c, err)
			return
		}
		c.JSON(201, gin.H{"data": row})
	})
	v1.GET("/hunts/:id", func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		row, err := s.Get(c.Request.Context(), id)
		if err != nil {
			personalError(c, err)
			return
		}
		c.JSON(200, gin.H{"data": row})
	})
	v1.PUT("/hunts/:id", func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		in, ok := body[service.HuntInput](c)
		if !ok {
			return
		}
		row, err := s.Update(c.Request.Context(), id, *in)
		if err != nil {
			personalError(c, err)
			return
		}
		c.JSON(200, gin.H{"data": row})
	})
	v1.DELETE("/hunts/:id", func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		if err := s.Delete(c.Request.Context(), id); err != nil {
			personalError(c, err)
			return
		}
		c.Status(204)
	})
}
