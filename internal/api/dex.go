package api

import (
	"github.com/gin-gonic/gin"
	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"github.com/zl14229/pokemon-shiny-tracker/internal/service"
	"gorm.io/gorm"
	"strconv"
)

func dexRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	v1.GET("/reference/availability", func(c *gin.Context) {
		limit, offset, ok := referencePaging(c)
		if !ok {
			return
		}
		q := db.Model(&model.ShinyAvailability{})
		for _, key := range []string{"game_id", "form_id"} {
			if value, ok := c.GetQuery(key); ok {
				id, e := strconv.ParseUint(value, 10, 63)
				if e != nil || id == 0 {
					abort(c, 400, "invalid_request", "可获取性ID不合法")
					return
				}
				q = q.Where(key+" = ?", id)
			}
		}
		var count int64
		if e := q.Count(&count).Error; e != nil {
			storageError(c, e)
			return
		}
		rows := []model.ShinyAvailability{}
		if e := q.Order("id ASC").Limit(limit).Offset(offset).Find(&rows).Error; e != nil {
			storageError(c, e)
			return
		}
		c.JSON(200, gin.H{"data": rows, "meta": gin.H{"total": count, "limit": limit, "offset": offset, "coverage": "partial"}})
	})
	s := service.Dex{DB: db}
	v1.GET("/dex", func(c *gin.Context) {
		result, err := s.List(c.Request.Context(), c.Request.URL.RawQuery)
		if err != nil {
			personalError(c, err)
			return
		}
		c.JSON(200, result)
	})
}
