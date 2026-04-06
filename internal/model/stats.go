package model

type StatsMetrics struct {
	InputToken      int64   `json:"input_token" gorm:"bigint"`
	OutputToken     int64   `json:"output_token" gorm:"bigint"`
	CacheReadToken  int64   `json:"cache_read_token" gorm:"bigint"`
	CacheWriteToken int64   `json:"cache_write_token" gorm:"bigint"`
	InputCost       float64 `json:"input_cost" gorm:"type:real"`
	OutputCost      float64 `json:"output_cost" gorm:"type:real"`
	WaitTime        int64   `json:"wait_time" gorm:"bigint"`
	RequestSuccess  int64   `json:"request_success" gorm:"bigint"`
	RequestFailed   int64   `json:"request_failed" gorm:"bigint"`
}

type StatsTotal struct {
	ID int `gorm:"primaryKey"`
	StatsMetrics
}

type StatsHourly struct {
	Hour int    `json:"hour" gorm:"primaryKey"`
	Date string `json:"date" gorm:"not null"` // 记录最后更新日期，格式：20060102
	StatsMetrics
}

type StatsDaily struct {
	Date string `json:"date" gorm:"primaryKey"`
	StatsMetrics
}

type StatsModel struct {
	ID        int    `json:"id" gorm:"primaryKey"`
	Name      string `json:"name" gorm:"not null"`
	ChannelID int    `json:"channel_id" gorm:"not null"`
	StatsMetrics
}

type StatsChannel struct {
	ChannelID int `json:"channel_id" gorm:"primaryKey"`
	StatsMetrics
}

type StatsAPIKey struct {
	APIKeyID int `json:"api_key_id" gorm:"primaryKey"`
	StatsMetrics
}

// ChannelRankItem represents per-channel aggregated stats for a date range.
type ChannelRankItem struct {
	ChannelID       int     `json:"channel_id"`
	ChannelName     string  `json:"channel_name"`
	InputToken      int64   `json:"input_token"`
	OutputToken     int64   `json:"output_token"`
	CacheReadToken  int64   `json:"cache_read_token"`
	CacheWriteToken int64   `json:"cache_write_token"`
	TotalCost       float64 `json:"total_cost"`
	RequestSuccess  int64   `json:"request_success"`
	RequestFailed   int64   `json:"request_failed"`
	WaitTime        int64   `json:"wait_time"`
}

// ModelRankItem represents per-channel-model aggregated stats for a date range.
type ModelRankItem struct {
	ChannelID       int     `json:"channel_id"`
	ChannelName     string  `json:"channel_name"`
	ModelName       string  `json:"model_name"`
	InputToken      int64   `json:"input_token"`
	OutputToken     int64   `json:"output_token"`
	CacheReadToken  int64   `json:"cache_read_token"`
	CacheWriteToken int64   `json:"cache_write_token"`
	TotalCost       float64 `json:"total_cost"`
	RequestSuccess  int64   `json:"request_success"`
	RequestFailed   int64   `json:"request_failed"`
	WaitTime        int64   `json:"wait_time"`
}

// Add aggregates another StatsMetrics into the current one.
func (s *StatsMetrics) Add(delta StatsMetrics) {
	s.InputToken += delta.InputToken
	s.OutputToken += delta.OutputToken
	s.CacheReadToken += delta.CacheReadToken
	s.CacheWriteToken += delta.CacheWriteToken
	s.InputCost += delta.InputCost
	s.OutputCost += delta.OutputCost
	s.WaitTime += delta.WaitTime
	s.RequestSuccess += delta.RequestSuccess
	s.RequestFailed += delta.RequestFailed
}
