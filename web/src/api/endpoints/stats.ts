import { useQuery } from '@tanstack/react-query';
import { apiClient } from '../client';
import { formatCount, formatMoney, formatTime } from '@/lib/utils';
import { useSettingStore } from '@/stores/setting';

/**
 * 统计数据
 */
interface StatsMetrics {
    input_token: number;
    output_token: number;
    cache_read_token: number;
    cache_write_token: number;
    input_cost: number;
    output_cost: number;
    wait_time: number;
    request_success: number;
    request_failed: number;
}

export interface StatsMetricsFormatted {
    input_token: ReturnType<typeof formatCount>;
    output_token: ReturnType<typeof formatCount>;
    cache_read_token: ReturnType<typeof formatCount>;
    cache_write_token: ReturnType<typeof formatCount>;
    input_cost: ReturnType<typeof formatMoney>;
    output_cost: ReturnType<typeof formatMoney>;
    wait_time: ReturnType<typeof formatTime>;
    request_success: ReturnType<typeof formatCount>;
    request_failed: ReturnType<typeof formatCount>;

    request_count: ReturnType<typeof formatCount>;
    total_token: ReturnType<typeof formatCount>;
    total_cost: ReturnType<typeof formatMoney>;
}

export interface StatsChannel extends StatsMetrics {
    channel_id: number;
}

export interface StatsDaily extends StatsMetrics {
    date: string;
}
export interface StatsDailyFormatted extends StatsMetricsFormatted {
    date: string;
}

export interface StatsTotal extends StatsMetrics {
    id: number;
}
export type StatsTotalFormatted = StatsMetricsFormatted;

export interface StatsHourly extends StatsMetrics {
    hour: number;
    date: string;
}
export interface StatsHourlyFormatted extends StatsMetricsFormatted {
    hour: number;
    date: string;
}
/**
 * API Key 统计数据
 */
export interface StatsAPIKey extends StatsMetrics {
    api_key_id: number;
}

export interface StatsAPIKeyFormatted extends StatsMetricsFormatted {
    api_key_id: number;
}

/**
 * Channel Rank 数据
 */
export interface ChannelRankItem {
    channel_id: number;
    channel_name: string;
    input_token: number;
    output_token: number;
    total_cost: number;
    request_success: number;
    request_failed: number;
    wait_time: number;
}

export interface ChannelRankItemFormatted {
    channel_id: number;
    channel_name: string;
    input_token: ReturnType<typeof formatCount>;
    output_token: ReturnType<typeof formatCount>;
    total_token: ReturnType<typeof formatCount>;
    total_cost: ReturnType<typeof formatMoney>;
    request_success: ReturnType<typeof formatCount>;
    request_failed: ReturnType<typeof formatCount>;
    request_count: ReturnType<typeof formatCount>;
    wait_time: ReturnType<typeof formatTime>;
}

/**
 * Model Rank 数据
 */
export interface ModelRankItem {
    channel_id: number;
    channel_name: string;
    model_name: string;
    input_token: number;
    output_token: number;
    cache_read_token: number;
    cache_write_token: number;
    total_cost: number;
    request_success: number;
    request_failed: number;
    wait_time: number;
}

export interface ModelRankItemFormatted {
    channel_id: number;
    channel_name: string;
    model_name: string;
    input_token: ReturnType<typeof formatCount>;
    output_token: ReturnType<typeof formatCount>;
    total_token: ReturnType<typeof formatCount>;
    total_cost: ReturnType<typeof formatMoney>;
    request_success: ReturnType<typeof formatCount>;
    request_failed: ReturnType<typeof formatCount>;
    request_count: ReturnType<typeof formatCount>;
    wait_time: ReturnType<typeof formatTime>;
}

/**
 * 格式化 StatsMetrics 的通用辅助函数
 */
function formatStatsMetrics(item: StatsMetrics): StatsMetricsFormatted {
    return {
        input_token: formatCount(item.input_token),
        output_token: formatCount(item.output_token),
        cache_read_token: formatCount(item.cache_read_token),
        cache_write_token: formatCount(item.cache_write_token),
        total_token: formatCount(item.input_token + item.output_token),
        input_cost: formatMoney(item.input_cost),
        output_cost: formatMoney(item.output_cost),
        total_cost: formatMoney(item.input_cost + item.output_cost),
        wait_time: formatTime(item.wait_time),
        request_success: formatCount(item.request_success),
        request_failed: formatCount(item.request_failed),
        request_count: formatCount(item.request_success + item.request_failed),
    };
}

/**
 * 获取今日统计数据 Hook
 */
export function useStatsToday() {
    return useQuery({
        queryKey: ['stats', 'today'],
        queryFn: async () => {
            return apiClient.get<StatsDaily>('/api/v1/stats/today');
        },
        refetchInterval: 30000,
        refetchOnMount: 'always',
    });
}

/**
 * 获取每日统计数据 Hook（支持可选日期范围）
 */
export function useStatsDaily(start?: string, end?: string) {
    const { currency, exchangeRate } = useSettingStore();
    return useQuery({
        queryKey: ['stats', 'daily', start, end, currency, exchangeRate],
        queryFn: async () => {
            const params: Record<string, string> = {};
            if (start) params.start = start;
            if (end) params.end = end;
            return apiClient.get<StatsDaily[]>('/api/v1/stats/daily', Object.keys(params).length > 0 ? params : undefined);
        },
        select: (data) => data.map((item): StatsDailyFormatted => ({
            ...formatStatsMetrics(item),
            date: item.date,
        })),
        refetchInterval: 3600000, // 1 小时
        refetchOnMount: 'always',
    });
}

/**
 * 获取小时统计数据 Hook（支持可选日期参数）
 */
export function useStatsHourly(date?: string) {
    const { currency, exchangeRate } = useSettingStore();
    return useQuery({
        queryKey: ['stats', 'hourly', date, currency, exchangeRate],
        queryFn: async () => {
            const params: Record<string, string> = {};
            if (date) params.date = date;
            return apiClient.get<StatsHourly[]>('/api/v1/stats/hourly', Object.keys(params).length > 0 ? params : undefined);
        },
        select: (data) => data.map((item): StatsHourlyFormatted => ({
            ...formatStatsMetrics(item),
            hour: item.hour,
            date: item.date,
        })),
        refetchInterval: 10000, // 10 秒
        refetchOnMount: 'always',
    });
}

/**
 * 获取总统计数据 Hook
 */
export function useStatsTotal() {
    const { currency, exchangeRate } = useSettingStore();
    return useQuery({
        queryKey: ['stats', 'total', currency, exchangeRate],
        queryFn: async () => {
            return apiClient.get<StatsTotal>('/api/v1/stats/total');
        },
        select: (data) => formatStatsMetrics(data),
        refetchInterval: 10000, // 10 秒
        refetchOnMount: 'always',
    });
}

/**
 * 获取范围聚合统计数据 Hook（用于 Total 卡片）
 * 无参数时等效于 useStatsTotal
 */
export function useStatsRange(start?: string, end?: string) {
    const { currency, exchangeRate } = useSettingStore();
    return useQuery({
        queryKey: ['stats', 'range', start, end, currency, exchangeRate],
        queryFn: async () => {
            const params: Record<string, string> = {};
            if (start) params.start = start;
            if (end) params.end = end;
            return apiClient.get<StatsMetrics>('/api/v1/stats/range', Object.keys(params).length > 0 ? params : undefined);
        },
        select: (data) => formatStatsMetrics(data),
        refetchInterval: 10000,
        refetchOnMount: 'always',
    });
}

/**
 * 获取渠道排行数据 Hook
 */
export function useStatsChannelRank(start?: string, end?: string) {
    const { currency, exchangeRate } = useSettingStore();
    return useQuery({
        queryKey: ['stats', 'channel-rank', start, end, currency, exchangeRate],
        queryFn: async () => {
            const params: Record<string, string> = {};
            if (start) params.start = start;
            if (end) params.end = end;
            return apiClient.get<ChannelRankItem[]>('/api/v1/stats/channel-rank', Object.keys(params).length > 0 ? params : undefined);
        },
        select: (data) => data.map((item): ChannelRankItemFormatted => ({
            channel_id: item.channel_id,
            channel_name: item.channel_name,
            input_token: formatCount(item.input_token),
            output_token: formatCount(item.output_token),
            total_token: formatCount(item.input_token + item.output_token),
            total_cost: formatMoney(item.total_cost),
            request_success: formatCount(item.request_success),
            request_failed: formatCount(item.request_failed),
            request_count: formatCount(item.request_success + item.request_failed),
            wait_time: formatTime(item.wait_time),
        })),
        refetchInterval: 30000,
        refetchOnMount: 'always',
    });
}

/**
 * 获取模型排行数据 Hook
 */
export function useStatsModelRank(start?: string, end?: string) {
    const { currency, exchangeRate } = useSettingStore();
    return useQuery({
        queryKey: ['stats', 'model-rank', start, end, currency, exchangeRate],
        queryFn: async () => {
            const params: Record<string, string> = {};
            if (start) params.start = start;
            if (end) params.end = end;
            return apiClient.get<ModelRankItem[]>('/api/v1/stats/model-rank', Object.keys(params).length > 0 ? params : undefined);
        },
        select: (data) => data.map((item): ModelRankItemFormatted => ({
            channel_id: item.channel_id,
            channel_name: item.channel_name,
            model_name: item.model_name,
            input_token: formatCount(item.input_token),
            output_token: formatCount(item.output_token),
            total_token: formatCount(item.input_token + item.output_token),
            total_cost: formatMoney(item.total_cost),
            request_success: formatCount(item.request_success),
            request_failed: formatCount(item.request_failed),
            request_count: formatCount(item.request_success + item.request_failed),
            wait_time: formatTime(item.wait_time),
        })),
        refetchInterval: 30000,
        refetchOnMount: 'always',
    });
}

/**
 * 获取 API Key 统计数据列表 Hook
 */
export function useStatsAPIKey() {
    const { currency, exchangeRate } = useSettingStore();
    return useQuery({
        queryKey: ['stats', 'apikey', currency, exchangeRate],
        queryFn: async () => {
            return apiClient.get<StatsAPIKey[]>('/api/v1/stats/apikey');
        },
        select: (data) => data.map((item): StatsAPIKeyFormatted => ({
            ...formatStatsMetrics(item),
            api_key_id: item.api_key_id,
        })),
        refetchInterval: 30000,
        refetchOnMount: 'always',
    });
}
