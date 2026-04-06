'use client';

import { create } from 'zustand';
import { createJSONStorage, persist } from 'zustand/middleware';

export type RankSortMode = 'cost' | 'count' | 'tokens';
export type ChartMetricType = 'cost' | 'count' | 'tokens';
export type TimeRangePreset = 'today' | '7d' | '30d' | 'all' | 'custom';

interface HomeViewState {
    rankSortMode: RankSortMode;
    chartMetricType: ChartMetricType;
    timeRangePreset: TimeRangePreset;
    customStartDate: string | null; // YYYYMMDD
    customEndDate: string | null;   // YYYYMMDD
    setRankSortMode: (value: RankSortMode) => void;
    setChartMetricType: (value: ChartMetricType) => void;
    setTimeRangePreset: (preset: TimeRangePreset) => void;
    setCustomDateRange: (start: string, end: string) => void;
}

export const useHomeViewStore = create<HomeViewState>()(
    persist(
        (set) => ({
            rankSortMode: 'cost',
            chartMetricType: 'cost',
            timeRangePreset: 'today',
            customStartDate: null,
            customEndDate: null,
            setRankSortMode: (value) => set({ rankSortMode: value }),
            setChartMetricType: (value) => set({ chartMetricType: value }),
            setTimeRangePreset: (preset) => set({ timeRangePreset: preset }),
            setCustomDateRange: (start, end) => set({
                timeRangePreset: 'custom',
                customStartDate: start,
                customEndDate: end,
            }),
        }),
        {
            name: 'home-view-options-storage',
            storage: createJSONStorage(() => localStorage),
            partialize: (state) => ({
                rankSortMode: state.rankSortMode,
                chartMetricType: state.chartMetricType,
                timeRangePreset: state.timeRangePreset,
                customStartDate: state.customStartDate,
                customEndDate: state.customEndDate,
            }),
        }
    )
);
