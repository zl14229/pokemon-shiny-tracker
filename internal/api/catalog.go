package api

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"github.com/zl14229/pokemon-shiny-tracker/internal/reference"
	"gorm.io/gorm"
)

type catalogView struct {
	model.CatalogEntry
	Names       map[string]string `json:"names"`
	DisplayName string            `json:"display_name"`
	NationalDex int               `json:"national_dex"`
	Slug        string            `json:"slug"`
}

func catalogRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	v1.GET("/reference/catalog", func(c *gin.Context) {
		limit, offset, ok := referencePaging(c)
		if !ok {
			return
		}
		q := db.Model(&model.CatalogEntry{})
		if kind := c.Query("kind"); kind != "" {
			if kind != "species" && kind != "form" {
				abort(c, 400, "invalid_request", "kind 不合法")
				return
			}
			q = q.Where("kind = ?", kind)
		}
		if inclusion := c.Query("inclusion"); inclusion != "" {
			if inclusion != "included" && inclusion != "excluded" && inclusion != "unknown" {
				abort(c, 400, "invalid_request", "inclusion 不合法")
				return
			}
			q = q.Where("inclusion = ?", inclusion)
		}
		var total int64
		if err := q.Count(&total).Error; err != nil {
			storageError(c, err)
			return
		}
		rows := []model.CatalogEntry{}
		if err := q.Order("kind DESC, external_id ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
			storageError(c, err)
			return
		}
		views := []catalogView{}
		for _, row := range rows {
			var s model.Species
			if err := db.First(&s, row.SpeciesID).Error; err != nil {
				storageError(c, err)
				return
			}
			slug := s.Slug
			if row.FormID != nil {
				var f model.Form
				if err := db.First(&f, *row.FormID).Error; err != nil {
					storageError(c, err)
					return
				}
				slug = f.Slug
			}
			names := map[string]string{}
			if err := json.Unmarshal([]byte(row.NamesJSON), &names); err != nil {
				storageError(c, err)
				return
			}
			views = append(views, catalogView{row, names, reference.DisplayCatalogName(names, row.NameStatus, slug), s.NationalDex, slug})
		}
		c.JSON(200, gin.H{"data": views, "meta": gin.H{"total": total, "limit": limit, "offset": offset, "coverage": "snapshot_partial_verification"}})
	})
	v1.GET("/reference/catalog/coverage", func(c *gin.Context) {
		counts := map[string]int64{}
		for _, kind := range []string{"species", "form"} {
			var n int64
			if err := db.Model(&model.CatalogEntry{}).Where("kind = ?", kind).Count(&n).Error; err != nil {
				storageError(c, err)
				return
			}
			counts[kind] = n
		}
		for _, state := range []string{"included", "excluded", "unknown"} {
			var n int64
			if err := db.Model(&model.CatalogEntry{}).Where("kind = ? AND inclusion = ?", "form", state).Count(&n).Error; err != nil {
				storageError(c, err)
				return
			}
			counts["forms_"+state] = n
		}
		var verified int64
		if err := db.Model(&model.CatalogEntry{}).Where("cross_check_status = ?", "verified").Count(&verified).Error; err != nil {
			storageError(c, err)
			return
		}
		counts["cross_checked"] = verified
		var batches []model.ReferenceBatch
		if err := db.Where("source = ?", reference.CatalogSource).Find(&batches).Error; err != nil {
			storageError(c, err)
			return
		}
		versions := []string{}
		for _, batch := range batches {
			versions = append(versions, batch.Checksum)
		}
		c.JSON(200, gin.H{"data": gin.H{"counts": counts, "versions": versions, "source": reference.CatalogSource, "coverage": "snapshot_partial_verification", "chinese_names": "cross_check_pending", "shiny_availability": "not_verified", "note": "仅表示本地来源快照覆盖，不代表全部持久形态、官方中文核验或游戏合法闪光覆盖完整。"}})
	})
}
