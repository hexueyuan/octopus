'use client';

import { useMemo } from 'react';
import dayjs from 'dayjs';
import { useHomeViewStore } from './store';

export function useResolvedTimeRange(): { start?: string; end?: string } {
    const preset = useHomeViewStore((s) => s.timeRangePreset);
    const customStart = useHomeViewStore((s) => s.customStartDate);
    const customEnd = useHomeViewStore((s) => s.customEndDate);

    return useMemo(() => {
        const today = dayjs().format('YYYYMMDD');
        switch (preset) {
            case 'today':
                return { start: today, end: today };
            case '7d':
                return { start: dayjs().subtract(6, 'day').format('YYYYMMDD'), end: today };
            case '30d':
                return { start: dayjs().subtract(29, 'day').format('YYYYMMDD'), end: today };
            case 'all':
                return {};
            case 'custom':
                return {
                    start: customStart ?? today,
                    end: customEnd ?? today,
                };
        }
    }, [preset, customStart, customEnd]);
}
