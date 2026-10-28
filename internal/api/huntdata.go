package api

import (
	"github.com/gin-gonic/gin"
	"github.com/zl14229/pokemon-shiny-tracker/internal/huntdata"
	"gorm.io/gorm"
	"strconv"
)

func huntDataRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	v1.GET("/reference/hunt-data", func(c *gin.Context) {
		limit, offset, ok := referencePaging(c)
		if !ok {
			return
		}
		var gameID uint64
		if value, exists := c.GetQuery("game_id"); exists {
			var e error
			gameID, e = strconv.ParseUint(value, 10, 63)
			if e != nil || gameID == 0 {
				abort(c, 400, "invalid_request", "游戏 ID 无效")
				return
			}
		}
		rows := []huntdata.Row{}
		e := db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			all, e := huntdata.Read(tx)
			if e != nil {
				return e
			}
			for _, r := range all {
				if gameID == 0 || r.GameID == gameID {
					rows = append(rows, r)
				}
			}
			return nil
		})
		if e != nil {
			storageError(c, e)
			return
		}
		total := len(rows)
		end := offset + limit
		if end > total {
			end = total
		}
		if offset < total {
			rows = rows[offset:end]
		} else {
			rows = []huntdata.Row{}
		}
		c.JSON(200, gin.H{"data": rows, "meta": gin.H{"total": total, "offset": offset, "limit": limit, "coverage": "partial", "note": "仅限逐条核对的本地资料；缺失目标未知，遭遇率与条件闪光率独立。"}})
	})
}
