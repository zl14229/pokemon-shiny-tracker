package api

import (
	"github.com/gin-gonic/gin"
	"github.com/zl14229/pokemon-shiny-tracker/internal/service"
	"strconv"
)

func bingoTargetID(c *gin.Context) (uint64, bool) {
	id, e := strconv.ParseUint(c.Param("target"), 10, 63)
	if e != nil || id == 0 {
		abort(c, 400, "invalid_request", "格子ID无效")
		return 0, false
	}
	return id, true
}
func bingoRoutes(r *gin.RouterGroup, s service.Planner) {
	r.POST("/bingo", func(c *gin.Context) {
		in, ok := body[service.BingoCreateInput](c)
		if !ok {
			return
		}
		v, e := s.CreateBingo(c.Request.Context(), *in)
		if e != nil {
			personalError(c, e)
			return
		}
		c.JSON(201, gin.H{"data": v})
	})
	r.POST("/plans/:id/refresh", func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		if _, ok = body[struct{}](c); !ok {
			return
		}
		v, e := s.RefreshBingo(c.Request.Context(), id)
		if e != nil {
			personalError(c, e)
			return
		}
		c.JSON(200, gin.H{"data": v})
	})
	r.POST("/plans/:id/close", func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		in, ok := body[struct {
			Status string `json:"status"`
		}](c)
		if !ok {
			return
		}
		v, e := s.CloseBingo(c.Request.Context(), id, in.Status)
		if e != nil {
			personalError(c, e)
			return
		}
		c.JSON(200, gin.H{"data": v})
	})
	r.POST("/plans/:id/targets/:target/reroll", func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		target, ok := bingoTargetID(c)
		if !ok {
			return
		}
		in, ok := body[service.BingoRerollInput](c)
		if !ok {
			return
		}
		v, e := s.RerollBingo(c.Request.Context(), id, target, *in)
		if e != nil {
			personalError(c, e)
			return
		}
		c.JSON(200, gin.H{"data": v})
	})
	r.POST("/plans/:id/targets/:target/verify-date", func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		target, ok := bingoTargetID(c)
		if !ok {
			return
		}
		in, ok := body[service.BingoDateInput](c)
		if !ok {
			return
		}
		v, e := s.VerifyBingoDate(c.Request.Context(), id, target, *in)
		if e != nil {
			personalError(c, e)
			return
		}
		c.JSON(200, gin.H{"data": v})
	})
}
