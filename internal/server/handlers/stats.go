package handlers

import (
	"net/http"

	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/gin-gonic/gin"
)

func init() {
	router.NewGroupRouter("/api/v1/stats").
		Use(middleware.Auth()).
		AddRoute(
			router.NewRoute("/today", http.MethodGet).
				Handle(getStatsToday),
		).
		AddRoute(
			router.NewRoute("/daily", http.MethodGet).
				Handle(getStatsDaily),
		).
		AddRoute(
			router.NewRoute("/hourly", http.MethodGet).
				Handle(getStatsHourly),
		).
		AddRoute(
			router.NewRoute("/total", http.MethodGet).
				Handle(getStatsTotal),
		).
		AddRoute(
			router.NewRoute("/apikey", http.MethodGet).
				Handle(getStatsAPIKey),
		).
		AddRoute(
			router.NewRoute("/range", http.MethodGet).
				Handle(getStatsRange),
		).
		AddRoute(
			router.NewRoute("/channel-rank", http.MethodGet).
				Handle(getStatsChannelRank),
		).
		AddRoute(
			router.NewRoute("/model-rank", http.MethodGet).
				Handle(getStatsModelRank),
		)
}

func getStatsToday(c *gin.Context) {
	resp.Success(c, op.StatsTodayGet())
}

func getStatsDaily(c *gin.Context) {
	start := c.Query("start")
	end := c.Query("end")

	if start != "" && end != "" {
		statsDaily, err := op.StatsGetDailyRange(c.Request.Context(), start, end)
		if err != nil {
			resp.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		resp.Success(c, statsDaily)
		return
	}

	statsDaily, err := op.StatsGetDaily(c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, statsDaily)
}

func getStatsHourly(c *gin.Context) {
	date := c.Query("date")

	if date != "" {
		hourlyStats, err := op.StatsGetHourlyByDate(c.Request.Context(), date)
		if err != nil {
			resp.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		resp.Success(c, hourlyStats)
		return
	}

	resp.Success(c, op.StatsHourlyGet())
}

func getStatsTotal(c *gin.Context) {
	resp.Success(c, op.StatsTotalGet())
}

func getStatsAPIKey(c *gin.Context) {
	resp.Success(c, op.StatsAPIKeyList())
}

func getStatsRange(c *gin.Context) {
	start := c.Query("start")
	end := c.Query("end")

	if start == "" || end == "" {
		resp.Success(c, op.StatsTotalGet())
		return
	}

	aggregated, err := op.StatsGetDailyRangeAggregated(c.Request.Context(), start, end)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, aggregated)
}

func getStatsChannelRank(c *gin.Context) {
	start := c.Query("start")
	end := c.Query("end")

	if start == "" || end == "" {
		resp.Success(c, op.StatsGetChannelRankAll())
		return
	}

	rankItems, err := op.StatsGetChannelRankByRange(c.Request.Context(), start, end)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, rankItems)
}

func getStatsModelRank(c *gin.Context) {
	start := c.Query("start")
	end := c.Query("end")

	if start == "" || end == "" {
		rankItems, err := op.StatsGetModelRankAll(c.Request.Context())
		if err != nil {
			resp.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		resp.Success(c, rankItems)
		return
	}

	rankItems, err := op.StatsGetModelRankByRange(c.Request.Context(), start, end)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, rankItems)
}