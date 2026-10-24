package api

import (
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"github.com/zl14229/pokemon-shiny-tracker/internal/reference"
	"gorm.io/gorm"
)

func routeID(c *gin.Context) (uint64, bool) {
	value, err := strconv.ParseUint(c.Param("id"), 10, 63)
	if err != nil || value == 0 {
		abort(c, 400, "invalid_request", "id 必须为正整数")
		return 0, false
	}
	return value, true
}

func storageError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		abort(c, 404, "not_found", "记录不存在")
		return
	}
	_ = c.Error(err)
}

func referencePaging(c *gin.Context) (int, int, bool) {
	limit, offset := 100, 0
	for key, destination := range map[string]*int{"limit": &limit, "offset": &offset} {
		if value, ok := c.GetQuery(key); ok {
			n, err := strconv.ParseUint(value, 10, 31)
			if err != nil || n > math.MaxInt32 {
				abort(c, 400, "invalid_request", "分页参数不合法")
				return 0, 0, false
			}
			*destination = int(n)
		}
	}
	if limit < 1 || limit > 500 {
		abort(c, 400, "invalid_request", "limit 必须在 1 至 500 之间")
		return 0, 0, false
	}
	return limit, offset, true
}

func referenceRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	registerReference[model.Species](v1, db, "species", "species")
	registerReference[model.Form](v1, db, "forms", "form")
	registerReference[model.Game](v1, db, "games", "game")
	v1.GET("/reference/coverage", func(c *gin.Context) {
		counts := map[string]int64{}
		for name, entity := range map[string]any{"species": &model.Species{}, "forms": &model.Form{}, "games": &model.Game{}, "source_mappings": &model.ExternalReference{}} {
			var count int64
			if err := db.Model(entity).Count(&count).Error; err != nil {
				storageError(c, err)
				return
			}
			counts[name] = count
		}
		c.JSON(200, gin.H{"data": gin.H{"coverage": "partial", "counts": counts, "shiny_availability": "not_verified", "configured_source": reference.Source, "chinese_names": "source_provided_or_unknown_or_conflict"}})
	})
}

func registerReference[T any](v1 *gin.RouterGroup, db *gorm.DB, path, kind string) {
	base := "/reference/" + path
	v1.GET(base, func(c *gin.Context) {
		limit, offset, ok := referencePaging(c)
		if !ok {
			return
		}
		query := db.Model(new(T))
		if value, ok := c.GetQuery("species_id"); ok && kind == "form" {
			id, err := strconv.ParseUint(value, 10, 63)
			if err != nil || id == 0 {
				abort(c, 400, "invalid_request", "species_id 不合法")
				return
			}
			query = query.Where("species_id = ?", id)
		}
		var count int64
		if err := query.Count(&count).Error; err != nil {
			storageError(c, err)
			return
		}
		rows := []T{}
		if err := query.Order("id ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
			storageError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": rows, "meta": gin.H{"total": count, "limit": limit, "offset": offset, "coverage": "partial"}})
	})
	v1.GET(base+"/:id", func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		var row T
		if err := db.First(&row, id).Error; err != nil {
			storageError(c, err)
			return
		}
		c.JSON(200, gin.H{"data": row})
	})
	v1.GET(base+"/:id/sources", func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		var row T
		if err := db.First(&row, id).Error; err != nil {
			storageError(c, err)
			return
		}
		column := map[string]string{"species": "species_id", "form": "form_id", "game": "game_id"}[kind]
		rows := []model.ExternalReference{}
		if err := db.Where("resource = ? AND "+column+" = ?", kind, id).Order("id ASC").Find(&rows).Error; err != nil {
			storageError(c, err)
			return
		}
		c.JSON(200, gin.H{"data": rows})
	})
}
