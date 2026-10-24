package api

import (
	"github.com/gin-gonic/gin"
	"github.com/zl14229/pokemon-shiny-tracker/internal/service"
	"gorm.io/gorm"
)

func shinyRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	s := service.Shinies{DB: db}
	v1.GET("/shiny-records", func(c *gin.Context) {
		rows, total, err := s.List(c.Request.Context())
		if err != nil {
			personalError(c, err)
			return
		}
		c.JSON(200, gin.H{"data": rows, "meta": gin.H{"total": total, "limit": 100}})
	})
	v1.POST("/shiny-records", func(c *gin.Context) {
		in, ok := body[service.ShinyInput](c)
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
	v1.GET("/shiny-records/:id", func(c *gin.Context) {
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
	v1.PUT("/shiny-records/:id", func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		in, ok := body[service.ShinyInput](c)
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
	v1.DELETE("/shiny-records/:id", func(c *gin.Context) {
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
	v1.GET("/shiny-records/:id/history", func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		limit, offset, ok := referencePaging(c)
		if !ok {
			return
		}
		rows, deleted, total, err := s.History(c.Request.Context(), id, limit, offset)
		if err != nil {
			personalError(c, err)
			return
		}
		c.JSON(200, gin.H{"data": rows, "meta": gin.H{"record_deleted": deleted, "total": total, "limit": limit, "offset": offset}})
	})
}
