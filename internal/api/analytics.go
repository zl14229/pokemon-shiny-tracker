package api

import (
	"github.com/gin-gonic/gin"
	"github.com/zl14229/pokemon-shiny-tracker/internal/service"
	"gorm.io/gorm"
)

func analyticsRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	s := service.Analytics{DB: db}
	v1.GET("/statistics", func(c *gin.Context) {
		v, err := s.Statistics(c.Request.Context(), c.Request.URL.RawQuery)
		if err != nil {
			personalError(c, err)
			return
		}
		c.JSON(200, gin.H{"data": v})
	})
	v1.GET("/dashboard", func(c *gin.Context) {
		v, err := s.Dashboard(c.Request.Context(), c.Request.URL.RawQuery)
		if err != nil {
			personalError(c, err)
			return
		}
		c.JSON(200, gin.H{"data": v})
	})
}
