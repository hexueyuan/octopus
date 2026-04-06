'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { CalendarDays } from 'lucide-react';
import dayjs from 'dayjs';
import { Calendar } from '@/components/ui/calendar';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Tabs, TabsList, TabsTrigger } from '@/components/animate-ui/components/animate/tabs';
import { useHomeViewStore, type TimeRangePreset } from './store';
import type { DateRange } from 'react-day-picker';

const PRESETS: TimeRangePreset[] = ['today', '7d', '30d', 'all'];

export function TimeRangePicker() {
    const t = useTranslations('home.timeRange');
    const preset = useHomeViewStore((s) => s.timeRangePreset);
    const customStartDate = useHomeViewStore((s) => s.customStartDate);
    const customEndDate = useHomeViewStore((s) => s.customEndDate);
    const setTimeRangePreset = useHomeViewStore((s) => s.setTimeRangePreset);
    const setCustomDateRange = useHomeViewStore((s) => s.setCustomDateRange);
    const [calendarOpen, setCalendarOpen] = useState(false);

    const presetLabels: Record<TimeRangePreset, string> = {
        today: t('today'),
        '7d': t('last7Days'),
        '30d': t('last30Days'),
        all: t('all'),
        custom: t('custom'),
    };

    const handlePresetChange = (value: string) => {
        setTimeRangePreset(value as TimeRangePreset);
    };

    const customRange: DateRange | undefined =
        customStartDate && customEndDate
            ? {
                from: dayjs(customStartDate, 'YYYYMMDD').toDate(),
                to: dayjs(customEndDate, 'YYYYMMDD').toDate(),
            }
            : undefined;

    const handleRangeSelect = (range: DateRange | undefined) => {
        if (range?.from && range?.to) {
            setCustomDateRange(
                dayjs(range.from).format('YYYYMMDD'),
                dayjs(range.to).format('YYYYMMDD')
            );
            setCalendarOpen(false);
        }
    };

    const customLabel = customStartDate && customEndDate && preset === 'custom'
        ? `${dayjs(customStartDate, 'YYYYMMDD').format('MM/DD')} - ${dayjs(customEndDate, 'YYYYMMDD').format('MM/DD')}`
        : t('custom');

    return (
        <div className="flex items-center justify-between gap-3">
            <Tabs value={preset} onValueChange={handlePresetChange}>
                <TabsList>
                    {PRESETS.map((p) => (
                        <TabsTrigger key={p} value={p}>
                            {presetLabels[p]}
                        </TabsTrigger>
                    ))}
                    <Popover open={calendarOpen} onOpenChange={setCalendarOpen}>
                        <PopoverTrigger asChild>
                            <TabsTrigger
                                value="custom"
                                onClick={(e) => {
                                    if (preset === 'custom') {
                                        e.preventDefault();
                                        setCalendarOpen(true);
                                    }
                                }}
                                className="gap-1"
                            >
                                <CalendarDays className="size-3.5" />
                                {customLabel}
                            </TabsTrigger>
                        </PopoverTrigger>
                        <PopoverContent
                            align="end"
                            side="bottom"
                            sideOffset={8}
                            className="w-fit rounded-2xl border border-border/60 shadow-xl overflow-hidden bg-card p-0"
                        >
                            <Calendar
                                mode="range"
                                selected={customRange}
                                onSelect={handleRangeSelect}
                                numberOfMonths={2}
                                disabled={{ after: new Date() }}
                                defaultMonth={dayjs().subtract(1, 'month').toDate()}
                            />
                        </PopoverContent>
                    </Popover>
                </TabsList>
            </Tabs>
        </div>
    );
}
