package api

import (
	"github.com/gin-gonic/gin"
	"github.com/zl14229/pokemon-shiny-tracker/internal/service"
	"gorm.io/gorm"
	"strconv"
)

func plannerRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	s := service.Planner{DB: db}
	r := v1.Group("/planner", localOrigin)
	bingoRoutes(r, s)
	r.POST("/pool", func(c *gin.Context) {
		in, ok := body[struct {
			Rules service.PlannerRules `json:"rules"`
		}](c)
		if !ok {
			return
		}
		out, e := s.Pool(c.Request.Context(), in.Rules)
		if e != nil {
			personalError(c, e)
			return
		}
		c.JSON(200, gin.H{"data": out})
	})
	r.POST("/random", func(c *gin.Context) {
		in, ok := body[service.PlannerDrawInput](c)
		if !ok {
			return
		}
		out, e := s.Draw(c.Request.Context(), *in)
		if e != nil {
			personalError(c, e)
			return
		}
		c.JSON(201, gin.H{"data": out})
	})
	r.GET("/plans", func(c *gin.Context) {
		limit, offset, ok := referencePaging(c)
		if !ok {
			return
		}
		if e := service.ValidatePlannerPaging(limit, offset); e != nil {
			personalError(c, e)
			return
		}
		out, total, e := s.List(c.Request.Context(), limit, offset)
		if e != nil {
			personalError(c, e)
			return
		}
		c.JSON(200, gin.H{"data": out, "meta": gin.H{"total": total, "limit": limit, "offset": offset}})
	})
	r.GET("/plans/:id", func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		out, e := s.Get(c.Request.Context(), id)
		if e != nil {
			personalError(c, e)
			return
		}
		c.JSON(200, gin.H{"data": out})
	})
	r.POST("/plans/:id/save", func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		if _, ok = body[struct{}](c); !ok {
			return
		}
		out, e := s.Save(c.Request.Context(), id)
		if e != nil {
			personalError(c, e)
			return
		}
		c.JSON(200, gin.H{"data": out})
	})
	r.POST("/plans/:id/targets/:target/hunt", func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		target, e := strconv.ParseUint(c.Param("target"), 10, 63)
		if e != nil || target == 0 {
			abort(c, 400, "invalid_request", "目标ID无效")
			return
		}
		in, ok := body[struct {
			HuntID            *uint64 `json:"hunt_id"`
			ExpectedTargetKey *string `json:"expected_target_key"`
		}](c)
		if !ok {
			return
		}
		expected := []string{}
		if in.ExpectedTargetKey != nil {
			expected = append(expected, *in.ExpectedTargetKey)
		}
		out, e := s.StartHunt(c.Request.Context(), id, target, in.HuntID, expected...)
		if e != nil {
			personalError(c, e)
			return
		}
		c.JSON(200, gin.H{"data": out})
	})
}
