package op

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/cache"
	"github.com/bestruirui/octopus/internal/utils/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var statsDailyCache model.StatsDaily
var statsDailyCacheLock sync.RWMutex

var statsTotalCache model.StatsTotal
var statsTotalCacheLock sync.RWMutex

var statsHourlyCache [24]model.StatsHourly
var statsHourlyCacheLock sync.RWMutex

var statsChannelCache = cache.New[int, model.StatsChannel](16)
var statsChannelCacheNeedUpdate = make(map[int]struct{})
var statsChannelCacheNeedUpdateLock sync.Mutex

var statsModelCache = cache.New[int, model.StatsModel](16)
var statsModelCacheNeedUpdate = make(map[int]struct{})
var statsModelCacheNeedUpdateLock sync.Mutex

var statsAPIKeyCache = cache.New[int, model.StatsAPIKey](16)
var statsAPIKeyCacheNeedUpdate = make(map[int]struct{})
var statsAPIKeyCacheNeedUpdateLock sync.Mutex

func StatsSaveDBTask() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	log.Debugf("stats save db task started")
	startTime := time.Now()
	defer func() {
		log.Debugf("stats save db task finished, save time: %s", time.Since(startTime))
	}()
	if err := StatsSaveDB(ctx); err != nil {
		log.Errorf("stats save db error: %v", err)
		return
	}
}

func StatsSaveDB(ctx context.Context) error {
	statsTotalCacheLock.RLock()
	totalSnap := statsTotalCache
	statsTotalCacheLock.RUnlock()
	if totalSnap.ID == 0 {
		totalSnap.ID = 1
	}

	statsDailyCacheLock.RLock()
	dailySnap := statsDailyCache
	statsDailyCacheLock.RUnlock()

	statsHourlyCacheLock.RLock()
	hourlyAll := statsHourlyCache
	statsHourlyCacheLock.RUnlock()

	statsChannelCacheNeedUpdateLock.Lock()
	channelIDs := make([]int, 0, len(statsChannelCacheNeedUpdate))
	for id := range statsChannelCacheNeedUpdate {
		channelIDs = append(channelIDs, id)
	}
	statsChannelCacheNeedUpdate = make(map[int]struct{})
	statsChannelCacheNeedUpdateLock.Unlock()

	statsModelCacheNeedUpdateLock.Lock()
	modelIDs := make([]int, 0, len(statsModelCacheNeedUpdate))
	for id := range statsModelCacheNeedUpdate {
		modelIDs = append(modelIDs, id)
	}
	statsModelCacheNeedUpdate = make(map[int]struct{})
	statsModelCacheNeedUpdateLock.Unlock()

	statsAPIKeyCacheNeedUpdateLock.Lock()
	apiKeyIDs := make([]int, 0, len(statsAPIKeyCacheNeedUpdate))
	for id := range statsAPIKeyCacheNeedUpdate {
		apiKeyIDs = append(apiKeyIDs, id)
	}
	statsAPIKeyCacheNeedUpdate = make(map[int]struct{})
	statsAPIKeyCacheNeedUpdateLock.Unlock()

	return persistStatsSnapshots(ctx, totalSnap, dailySnap, hourlyAll, channelIDs, modelIDs, apiKeyIDs)
}

func persistStatsSnapshots(
	ctx context.Context,
	totalSnap model.StatsTotal,
	dailySnap model.StatsDaily,
	hourlyAll [24]model.StatsHourly,
	channelIDs []int,
	modelIDs []int,
	apiKeyIDs []int,
) error {
	dbConn := db.GetDB().WithContext(ctx)

	if result := dbConn.Save(&totalSnap); result.Error != nil {
		return result.Error
	}
	if result := dbConn.Save(&dailySnap); result.Error != nil {
		return result.Error
	}

	todayDate := time.Now().Format("20060102")
	hourlyStats := make([]model.StatsHourly, 0, 24)
	for hour := 0; hour < 24; hour++ {
		if hourlyAll[hour].Date == todayDate {
			hourlyStats = append(hourlyStats, hourlyAll[hour])
		}
	}
	if len(hourlyStats) > 0 {
		if result := dbConn.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "hour"}},
			UpdateAll: true,
		}).Create(&hourlyStats); result.Error != nil {
			return result.Error
		}
	}

	for _, id := range channelIDs {
		ch, ok := statsChannelCache.Get(id)
		if !ok {
			continue
		}
		if result := dbConn.Save(&ch); result.Error != nil {
			return result.Error
		}
	}

	for _, id := range modelIDs {
		m, ok := statsModelCache.Get(id)
		if !ok {
			continue
		}
		if result := dbConn.Save(&m); result.Error != nil {
			return result.Error
		}
	}

	for _, id := range apiKeyIDs {
		ak, ok := statsAPIKeyCache.Get(id)
		if !ok {
			continue
		}
		if result := dbConn.Save(&ak); result.Error != nil {
			return result.Error
		}
	}

	return nil
}

func statsSaveDBWithDailyOverride(ctx context.Context, dailyOverride model.StatsDaily) error {
	statsTotalCacheLock.RLock()
	totalSnap := statsTotalCache
	statsTotalCacheLock.RUnlock()
	if totalSnap.ID == 0 {
		totalSnap.ID = 1
	}

	statsHourlyCacheLock.RLock()
	hourlyAll := statsHourlyCache
	statsHourlyCacheLock.RUnlock()

	statsChannelCacheNeedUpdateLock.Lock()
	channelIDs := make([]int, 0, len(statsChannelCacheNeedUpdate))
	for id := range statsChannelCacheNeedUpdate {
		channelIDs = append(channelIDs, id)
	}
	statsChannelCacheNeedUpdate = make(map[int]struct{})
	statsChannelCacheNeedUpdateLock.Unlock()

	statsModelCacheNeedUpdateLock.Lock()
	modelIDs := make([]int, 0, len(statsModelCacheNeedUpdate))
	for id := range statsModelCacheNeedUpdate {
		modelIDs = append(modelIDs, id)
	}
	statsModelCacheNeedUpdate = make(map[int]struct{})
	statsModelCacheNeedUpdateLock.Unlock()

	statsAPIKeyCacheNeedUpdateLock.Lock()
	apiKeyIDs := make([]int, 0, len(statsAPIKeyCacheNeedUpdate))
	for id := range statsAPIKeyCacheNeedUpdate {
		apiKeyIDs = append(apiKeyIDs, id)
	}
	statsAPIKeyCacheNeedUpdate = make(map[int]struct{})
	statsAPIKeyCacheNeedUpdateLock.Unlock()

	return persistStatsSnapshots(ctx, totalSnap, dailyOverride, hourlyAll, channelIDs, modelIDs, apiKeyIDs)
}

func StatsDailyUpdate(ctx context.Context, metrics model.StatsMetrics) error {
	today := time.Now().Format("20060102")

	statsDailyCacheLock.Lock()
	if statsDailyCache.Date == today {
		statsDailyCache.StatsMetrics.Add(metrics)
		statsDailyCacheLock.Unlock()
		return nil
	}

	prevDaily := statsDailyCache
	statsDailyCache = model.StatsDaily{Date: today}
	statsDailyCache.StatsMetrics.Add(metrics)
	statsDailyCacheLock.Unlock()

	return statsSaveDBWithDailyOverride(ctx, prevDaily)
}

func StatsTotalUpdate(metrics model.StatsMetrics) error {
	statsTotalCacheLock.Lock()
	defer statsTotalCacheLock.Unlock()
	if statsTotalCache.ID == 0 {
		statsTotalCache.ID = 1
	}
	statsTotalCache.StatsMetrics.Add(metrics)
	return nil
}

func StatsChannelUpdate(channelID int, metrics model.StatsMetrics) error {
	channelCache, ok := statsChannelCache.Get(channelID)
	if !ok {
		channelCache = model.StatsChannel{
			ChannelID: channelID,
		}
	}
	channelCache.StatsMetrics.Add(metrics)
	statsChannelCache.Set(channelID, channelCache)
	statsChannelCacheNeedUpdateLock.Lock()
	statsChannelCacheNeedUpdate[channelID] = struct{}{}
	statsChannelCacheNeedUpdateLock.Unlock()
	return nil
}

func StatsHourlyUpdate(metrics model.StatsMetrics) error {
	now := time.Now()
	nowHour := now.Hour()
	todayDate := time.Now().Format("20060102")

	statsHourlyCacheLock.Lock()
	defer statsHourlyCacheLock.Unlock()

	if statsHourlyCache[nowHour].Date != todayDate {
		statsHourlyCache[nowHour] = model.StatsHourly{
			Hour: nowHour,
			Date: todayDate,
		}
	}

	statsHourlyCache[nowHour].StatsMetrics.Add(metrics)
	return nil
}

func StatsModelUpdate(stats model.StatsModel) error {
	modelCache, ok := statsModelCache.Get(stats.ID)
	if !ok {
		modelCache = model.StatsModel{
			ID: stats.ID,
		}
	}
	modelCache.StatsMetrics.Add(stats.StatsMetrics)
	statsModelCache.Set(stats.ID, modelCache)
	statsModelCacheNeedUpdateLock.Lock()
	statsModelCacheNeedUpdate[stats.ID] = struct{}{}
	statsModelCacheNeedUpdateLock.Unlock()
	return nil
}

func StatsAPIKeyUpdate(apiKeyID int, metrics model.StatsMetrics) error {
	apiKeyCache, ok := statsAPIKeyCache.Get(apiKeyID)
	if !ok {
		apiKeyCache = model.StatsAPIKey{
			APIKeyID: apiKeyID,
		}
	}
	apiKeyCache.StatsMetrics.Add(metrics)
	statsAPIKeyCache.Set(apiKeyID, apiKeyCache)
	statsAPIKeyCacheNeedUpdateLock.Lock()
	statsAPIKeyCacheNeedUpdate[apiKeyID] = struct{}{}
	statsAPIKeyCacheNeedUpdateLock.Unlock()
	return nil
}

func StatsChannelDel(id int) error {
	if _, ok := statsChannelCache.Get(id); !ok {
		return nil
	}
	statsChannelCache.Del(id)
	statsChannelCacheNeedUpdateLock.Lock()
	delete(statsChannelCacheNeedUpdate, id)
	statsChannelCacheNeedUpdateLock.Unlock()
	return db.GetDB().Delete(&model.StatsChannel{}, id).Error
}

func StatsAPIKeyDel(id int) error {
	if _, ok := statsAPIKeyCache.Get(id); !ok {
		return nil
	}
	statsAPIKeyCache.Del(id)
	statsAPIKeyCacheNeedUpdateLock.Lock()
	delete(statsAPIKeyCacheNeedUpdate, id)
	statsAPIKeyCacheNeedUpdateLock.Unlock()
	return db.GetDB().Delete(&model.StatsAPIKey{}, id).Error
}

func StatsTotalGet() model.StatsTotal {
	statsTotalCacheLock.RLock()
	defer statsTotalCacheLock.RUnlock()
	return statsTotalCache
}

func StatsTodayGet() model.StatsDaily {
	statsDailyCacheLock.RLock()
	defer statsDailyCacheLock.RUnlock()
	return statsDailyCache
}

func StatsChannelGet(id int) model.StatsChannel {
	stats, ok := statsChannelCache.Get(id)
	if !ok {
		tmp := model.StatsChannel{
			ChannelID: id,
		}
		statsChannelCache.Set(id, tmp)
		statsChannelCacheNeedUpdateLock.Lock()
		statsChannelCacheNeedUpdate[id] = struct{}{}
		statsChannelCacheNeedUpdateLock.Unlock()
		return tmp
	}
	return stats
}

func StatsAPIKeyGet(id int) model.StatsAPIKey {
	stats, ok := statsAPIKeyCache.Get(id)
	if !ok {
		tmp := model.StatsAPIKey{
			APIKeyID: id,
		}
		statsAPIKeyCache.Set(id, tmp)
		statsAPIKeyCacheNeedUpdateLock.Lock()
		statsAPIKeyCacheNeedUpdate[id] = struct{}{}
		statsAPIKeyCacheNeedUpdateLock.Unlock()
		return tmp
	}
	return stats
}

func StatsAPIKeyList() []model.StatsAPIKey {
	apiKeys := make([]model.StatsAPIKey, 0, statsAPIKeyCache.Len())
	for _, v := range statsAPIKeyCache.GetAll() {
		apiKeys = append(apiKeys, v)
	}
	return apiKeys
}

func StatsHourlyGet() []model.StatsHourly {
	now := time.Now()
	currentHour := now.Hour()
	todayDate := time.Now().Format("20060102")

	statsHourlyCacheLock.RLock()
	defer statsHourlyCacheLock.RUnlock()

	result := make([]model.StatsHourly, 0, currentHour+1)

	for hour := 0; hour <= currentHour; hour++ {
		if statsHourlyCache[hour].Date == todayDate {
			result = append(result, statsHourlyCache[hour])
		} else {
			result = append(result, model.StatsHourly{
				Hour: hour,
				Date: todayDate,
			})
		}
	}

	return result
}

func StatsGetDaily(ctx context.Context) ([]model.StatsDaily, error) {
	var statsDaily []model.StatsDaily
	result := db.GetDB().WithContext(ctx).Find(&statsDaily)
	if result.Error != nil {
		return nil, result.Error
	}
	return statsDaily, nil
}

// StatsGetDailyRange returns daily stats within the given date range (inclusive).
// If the range includes today, the in-memory cache is merged since today's data may not be flushed.
func StatsGetDailyRange(ctx context.Context, startDate, endDate string) ([]model.StatsDaily, error) {
	var statsDaily []model.StatsDaily
	result := db.GetDB().WithContext(ctx).Where("date >= ? AND date <= ?", startDate, endDate).Find(&statsDaily)
	if result.Error != nil {
		return nil, result.Error
	}

	today := time.Now().Format("20060102")
	if today >= startDate && today <= endDate {
		statsDailyCacheLock.RLock()
		todayCache := statsDailyCache
		statsDailyCacheLock.RUnlock()

		if todayCache.Date == today {
			found := false
			for i, d := range statsDaily {
				if d.Date == today {
					statsDaily[i] = todayCache
					found = true
					break
				}
			}
			if !found {
				statsDaily = append(statsDaily, todayCache)
			}
		}
	}

	return statsDaily, nil
}

// StatsGetDailyRangeAggregated returns a single aggregated StatsMetrics for the given date range.
func StatsGetDailyRangeAggregated(ctx context.Context, startDate, endDate string) (model.StatsMetrics, error) {
	dailyStats, err := StatsGetDailyRange(ctx, startDate, endDate)
	if err != nil {
		return model.StatsMetrics{}, err
	}

	var aggregated model.StatsMetrics
	for _, d := range dailyStats {
		aggregated.Add(d.StatsMetrics)
	}
	return aggregated, nil
}

// StatsGetHourlyByDate returns hourly stats for a specific date.
// If the date is today, it returns data from the in-memory cache.
func StatsGetHourlyByDate(ctx context.Context, date string) ([]model.StatsHourly, error) {
	today := time.Now().Format("20060102")
	if date == today {
		return StatsHourlyGet(), nil
	}

	var hourlyStats []model.StatsHourly
	result := db.GetDB().WithContext(ctx).Where("date = ?", date).Order("hour ASC").Find(&hourlyStats)
	if result.Error != nil {
		return nil, result.Error
	}
	return hourlyStats, nil
}

// StatsGetChannelRankByRange returns per-channel aggregated stats from relay_logs for the given date range.
func StatsGetChannelRankByRange(ctx context.Context, startDate, endDate string) ([]model.ChannelRankItem, error) {
	startTime, err := time.ParseInLocation("20060102", startDate, time.Now().Location())
	if err != nil {
		return nil, fmt.Errorf("invalid start date: %w", err)
	}
	endTime, err := time.ParseInLocation("20060102", endDate, time.Now().Location())
	if err != nil {
		return nil, fmt.Errorf("invalid end date: %w", err)
	}
	// endTime should cover the entire day
	endTimestamp := endTime.Add(24*time.Hour - time.Second).Unix()
	startTimestamp := startTime.Unix()

	type dbResult struct {
		ChannelID       int     `gorm:"column:channel_id"`
		ChannelName     string  `gorm:"column:channel_name"`
		InputToken      int64   `gorm:"column:input_token"`
		OutputToken     int64   `gorm:"column:output_token"`
		CacheReadToken  int64   `gorm:"column:cache_read_token"`
		CacheWriteToken int64   `gorm:"column:cache_write_token"`
		TotalCost       float64 `gorm:"column:total_cost"`
		RequestSuccess  int64   `gorm:"column:request_success"`
		RequestFailed   int64   `gorm:"column:request_failed"`
		WaitTime        int64   `gorm:"column:wait_time"`
	}

	var results []dbResult
	err = db.GetDB().WithContext(ctx).
		Model(&model.RelayLog{}).
		Select(`channel_id AS channel_id,
			channel_name AS channel_name,
			COALESCE(SUM(input_tokens), 0) AS input_token,
			COALESCE(SUM(output_tokens), 0) AS output_token,
			COALESCE(SUM(cache_read_tokens), 0) AS cache_read_token,
			COALESCE(SUM(cache_write_tokens), 0) AS cache_write_token,
			COALESCE(SUM(cost), 0) AS total_cost,
			COALESCE(SUM(CASE WHEN error = '' THEN 1 ELSE 0 END), 0) AS request_success,
			COALESCE(SUM(CASE WHEN error != '' THEN 1 ELSE 0 END), 0) AS request_failed,
			COALESCE(SUM(use_time), 0) AS wait_time`).
		Where("time >= ? AND time <= ?", startTimestamp, endTimestamp).
		Group("channel_id").
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	// Build a map to merge in-memory relay log cache entries
	rankMap := make(map[int]*model.ChannelRankItem, len(results))
	for i := range results {
		r := &results[i]
		rankMap[r.ChannelID] = &model.ChannelRankItem{
			ChannelID:       r.ChannelID,
			ChannelName:     r.ChannelName,
			InputToken:      r.InputToken,
			OutputToken:     r.OutputToken,
			CacheReadToken:  r.CacheReadToken,
			CacheWriteToken: r.CacheWriteToken,
			TotalCost:       r.TotalCost,
			RequestSuccess:  r.RequestSuccess,
			RequestFailed:   r.RequestFailed,
			WaitTime:        r.WaitTime,
		}
	}

	// Merge unflushed relay log cache entries
	relayLogCacheLock.Lock()
	for _, rl := range relayLogCache {
		if rl.Time >= startTimestamp && rl.Time <= endTimestamp {
			item, ok := rankMap[rl.ChannelId]
			if !ok {
				item = &model.ChannelRankItem{
					ChannelID:   rl.ChannelId,
					ChannelName: rl.ChannelName,
				}
				rankMap[rl.ChannelId] = item
			}
			item.InputToken += int64(rl.InputTokens)
			item.OutputToken += int64(rl.OutputTokens)
			item.CacheReadToken += int64(rl.CacheReadTokens)
			item.CacheWriteToken += int64(rl.CacheWriteTokens)
			item.TotalCost += rl.Cost
			item.WaitTime += int64(rl.UseTime)
			if rl.Error == "" {
				item.RequestSuccess++
			} else {
				item.RequestFailed++
			}
		}
	}
	relayLogCacheLock.Unlock()

	// Convert map to slice
	rankItems := make([]model.ChannelRankItem, 0, len(rankMap))
	for _, item := range rankMap {
		rankItems = append(rankItems, *item)
	}

	return rankItems, nil
}

// StatsGetChannelRankAll returns per-channel cumulative stats from the in-memory cache.
func StatsGetChannelRankAll() []model.ChannelRankItem {
	allChannels := statsChannelCache.GetAll()
	items := make([]model.ChannelRankItem, 0, len(allChannels))
	for _, ch := range allChannels {
		items = append(items, model.ChannelRankItem{
			ChannelID:       ch.ChannelID,
			InputToken:      ch.InputToken,
			OutputToken:     ch.OutputToken,
			CacheReadToken:  ch.CacheReadToken,
			CacheWriteToken: ch.CacheWriteToken,
			TotalCost:       ch.InputCost + ch.OutputCost,
			RequestSuccess:  ch.RequestSuccess,
			RequestFailed:   ch.RequestFailed,
			WaitTime:        ch.WaitTime,
		})
	}
	return items
}

// StatsGetModelRankByRange returns per-channel-model aggregated stats from relay_logs for the given date range.
func StatsGetModelRankByRange(ctx context.Context, startDate, endDate string) ([]model.ModelRankItem, error) {
	startTime, err := time.ParseInLocation("20060102", startDate, time.Now().Location())
	if err != nil {
		return nil, fmt.Errorf("invalid start date: %w", err)
	}
	endTime, err := time.ParseInLocation("20060102", endDate, time.Now().Location())
	if err != nil {
		return nil, fmt.Errorf("invalid end date: %w", err)
	}
	endTimestamp := endTime.Add(24*time.Hour - time.Second).Unix()
	startTimestamp := startTime.Unix()

	type dbResult struct {
		ChannelID       int     `gorm:"column:channel_id"`
		ChannelName     string  `gorm:"column:channel_name"`
		ModelName       string  `gorm:"column:model_name"`
		InputToken      int64   `gorm:"column:input_token"`
		OutputToken     int64   `gorm:"column:output_token"`
		CacheReadToken  int64   `gorm:"column:cache_read_token"`
		CacheWriteToken int64   `gorm:"column:cache_write_token"`
		TotalCost       float64 `gorm:"column:total_cost"`
		RequestSuccess  int64   `gorm:"column:request_success"`
		RequestFailed   int64   `gorm:"column:request_failed"`
		WaitTime        int64   `gorm:"column:wait_time"`
	}

	var results []dbResult
	err = db.GetDB().WithContext(ctx).
		Model(&model.RelayLog{}).
		Select(`channel_id AS channel_id,
			channel_name AS channel_name,
			actual_model_name AS model_name,
			COALESCE(SUM(input_tokens), 0) AS input_token,
			COALESCE(SUM(output_tokens), 0) AS output_token,
			COALESCE(SUM(cache_read_tokens), 0) AS cache_read_token,
			COALESCE(SUM(cache_write_tokens), 0) AS cache_write_token,
			COALESCE(SUM(cost), 0) AS total_cost,
			COALESCE(SUM(CASE WHEN error = '' THEN 1 ELSE 0 END), 0) AS request_success,
			COALESCE(SUM(CASE WHEN error != '' THEN 1 ELSE 0 END), 0) AS request_failed,
			COALESCE(SUM(use_time), 0) AS wait_time`).
		Where("time >= ? AND time <= ? AND actual_model_name != ''", startTimestamp, endTimestamp).
		Group("channel_id, actual_model_name").
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	rankMap := make(map[string]*model.ModelRankItem, len(results))
	for i := range results {
		r := &results[i]
		key := fmt.Sprintf("%d:%s", r.ChannelID, r.ModelName)
		rankMap[key] = &model.ModelRankItem{
			ChannelID:       r.ChannelID,
			ChannelName:     r.ChannelName,
			ModelName:       r.ModelName,
			InputToken:      r.InputToken,
			OutputToken:     r.OutputToken,
			CacheReadToken:  r.CacheReadToken,
			CacheWriteToken: r.CacheWriteToken,
			TotalCost:       r.TotalCost,
			RequestSuccess:  r.RequestSuccess,
			RequestFailed:   r.RequestFailed,
			WaitTime:        r.WaitTime,
		}
	}

	// Merge unflushed relay log cache entries
	relayLogCacheLock.Lock()
	for _, rl := range relayLogCache {
		if rl.ActualModelName != "" && rl.Time >= startTimestamp && rl.Time <= endTimestamp {
			key := fmt.Sprintf("%d:%s", rl.ChannelId, rl.ActualModelName)
			item, ok := rankMap[key]
			if !ok {
				item = &model.ModelRankItem{
					ChannelID:   rl.ChannelId,
					ChannelName: rl.ChannelName,
					ModelName:   rl.ActualModelName,
				}
				rankMap[key] = item
			}
			item.InputToken += int64(rl.InputTokens)
			item.OutputToken += int64(rl.OutputTokens)
			item.CacheReadToken += int64(rl.CacheReadTokens)
			item.CacheWriteToken += int64(rl.CacheWriteTokens)
			item.TotalCost += rl.Cost
			item.WaitTime += int64(rl.UseTime)
			if rl.Error == "" {
				item.RequestSuccess++
			} else {
				item.RequestFailed++
			}
		}
	}
	relayLogCacheLock.Unlock()

	rankItems := make([]model.ModelRankItem, 0, len(rankMap))
	for _, item := range rankMap {
		rankItems = append(rankItems, *item)
	}

	return rankItems, nil
}

// StatsGetModelRankAll returns per-channel-model aggregated stats from all relay_logs.
func StatsGetModelRankAll(ctx context.Context) ([]model.ModelRankItem, error) {
	type dbResult struct {
		ChannelID       int     `gorm:"column:channel_id"`
		ChannelName     string  `gorm:"column:channel_name"`
		ModelName       string  `gorm:"column:model_name"`
		InputToken      int64   `gorm:"column:input_token"`
		OutputToken     int64   `gorm:"column:output_token"`
		CacheReadToken  int64   `gorm:"column:cache_read_token"`
		CacheWriteToken int64   `gorm:"column:cache_write_token"`
		TotalCost       float64 `gorm:"column:total_cost"`
		RequestSuccess  int64   `gorm:"column:request_success"`
		RequestFailed   int64   `gorm:"column:request_failed"`
		WaitTime        int64   `gorm:"column:wait_time"`
	}

	var results []dbResult
	err := db.GetDB().WithContext(ctx).
		Model(&model.RelayLog{}).
		Select(`channel_id AS channel_id,
			channel_name AS channel_name,
			actual_model_name AS model_name,
			COALESCE(SUM(input_tokens), 0) AS input_token,
			COALESCE(SUM(output_tokens), 0) AS output_token,
			COALESCE(SUM(cache_read_tokens), 0) AS cache_read_token,
			COALESCE(SUM(cache_write_tokens), 0) AS cache_write_token,
			COALESCE(SUM(cost), 0) AS total_cost,
			COALESCE(SUM(CASE WHEN error = '' THEN 1 ELSE 0 END), 0) AS request_success,
			COALESCE(SUM(CASE WHEN error != '' THEN 1 ELSE 0 END), 0) AS request_failed,
			COALESCE(SUM(use_time), 0) AS wait_time`).
		Where("actual_model_name != ''").
		Group("channel_id, actual_model_name").
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	rankMap := make(map[string]*model.ModelRankItem, len(results))
	for i := range results {
		r := &results[i]
		key := fmt.Sprintf("%d:%s", r.ChannelID, r.ModelName)
		rankMap[key] = &model.ModelRankItem{
			ChannelID:       r.ChannelID,
			ChannelName:     r.ChannelName,
			ModelName:       r.ModelName,
			InputToken:      r.InputToken,
			OutputToken:     r.OutputToken,
			CacheReadToken:  r.CacheReadToken,
			CacheWriteToken: r.CacheWriteToken,
			TotalCost:       r.TotalCost,
			RequestSuccess:  r.RequestSuccess,
			RequestFailed:   r.RequestFailed,
			WaitTime:        r.WaitTime,
		}
	}

	// Merge unflushed relay log cache entries
	relayLogCacheLock.Lock()
	for _, rl := range relayLogCache {
		if rl.ActualModelName != "" {
			key := fmt.Sprintf("%d:%s", rl.ChannelId, rl.ActualModelName)
			item, ok := rankMap[key]
			if !ok {
				item = &model.ModelRankItem{
					ChannelID:   rl.ChannelId,
					ChannelName: rl.ChannelName,
					ModelName:   rl.ActualModelName,
				}
				rankMap[key] = item
			}
			item.InputToken += int64(rl.InputTokens)
			item.OutputToken += int64(rl.OutputTokens)
			item.CacheReadToken += int64(rl.CacheReadTokens)
			item.CacheWriteToken += int64(rl.CacheWriteTokens)
			item.TotalCost += rl.Cost
			item.WaitTime += int64(rl.UseTime)
			if rl.Error == "" {
				item.RequestSuccess++
			} else {
				item.RequestFailed++
			}
		}
	}
	relayLogCacheLock.Unlock()

	rankItems := make([]model.ModelRankItem, 0, len(rankMap))
	for _, item := range rankMap {
		rankItems = append(rankItems, *item)
	}

	return rankItems, nil
}

func statsRefreshCache(ctx context.Context) error {
	dbConn := db.GetDB().WithContext(ctx)
	today := time.Now().Format("20060102")

	var loadedDaily model.StatsDaily
	result := dbConn.Last(&loadedDaily)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to get daily stats: %v", result.Error)
	}
	if result.RowsAffected == 0 || loadedDaily.Date != today {
		loadedDaily = model.StatsDaily{Date: today}
	}

	var loadedTotal model.StatsTotal
	result = dbConn.First(&loadedTotal)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to get total stats: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		loadedTotal = model.StatsTotal{ID: 1}
	} else if loadedTotal.ID == 0 {
		loadedTotal.ID = 1
	}

	var loadedChannels []model.StatsChannel
	result = dbConn.Find(&loadedChannels)
	if result.Error != nil {
		return fmt.Errorf("failed to get channels: %v", result.Error)
	}

	var loadedHourly []model.StatsHourly
	result = dbConn.Find(&loadedHourly)
	if result.Error != nil {
		return fmt.Errorf("failed to get hourly stats: %v", result.Error)
	}

	statsDailyCacheLock.Lock()
	statsDailyCache = loadedDaily
	statsDailyCacheLock.Unlock()

	statsTotalCacheLock.Lock()
	statsTotalCache = loadedTotal
	statsTotalCacheLock.Unlock()

	statsChannelCache.Clear()
	statsChannelCacheNeedUpdateLock.Lock()
	statsChannelCacheNeedUpdate = make(map[int]struct{})
	statsChannelCacheNeedUpdateLock.Unlock()
	for _, v := range loadedChannels {
		statsChannelCache.Set(v.ChannelID, v)
	}

	var loadedAPIKeys []model.StatsAPIKey
	result = dbConn.Find(&loadedAPIKeys)
	if result.Error != nil {
		return fmt.Errorf("failed to get api key stats: %v", result.Error)
	}

	statsAPIKeyCache.Clear()
	statsAPIKeyCacheNeedUpdateLock.Lock()
	statsAPIKeyCacheNeedUpdate = make(map[int]struct{})
	statsAPIKeyCacheNeedUpdateLock.Unlock()
	for _, v := range loadedAPIKeys {
		statsAPIKeyCache.Set(v.APIKeyID, v)
	}

	statsHourlyCacheLock.Lock()
	statsHourlyCache = [24]model.StatsHourly{}
	for _, v := range loadedHourly {
		if v.Hour >= 0 && v.Hour < 24 {
			statsHourlyCache[v.Hour] = v
		}
	}
	statsHourlyCacheLock.Unlock()

	return nil
}
