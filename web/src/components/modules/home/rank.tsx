'use client';

import { useStatsModelRank, type ModelRankItemFormatted } from '@/api/endpoints/stats';
import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { TrendingUp } from 'lucide-react';
import { Tabs, TabsList, TabsTrigger, TabsContents, TabsContent } from '@/components/animate-ui/components/animate/tabs';
import { useHomeViewStore, type RankSortMode } from '@/components/modules/home/store';
import { useResolvedTimeRange } from './hooks';

interface NormalizedModel {
    name: string;
    channel_name: string;
    total_cost: { raw: number; formatted: { value: string; unit: string } };
    request_count: { raw: number; formatted: { value: string; unit: string } };
    total_token: { raw: number; formatted: { value: string; unit: string } };
    request_success: { raw: number; formatted: { value: string; unit: string } };
    request_failed: { raw: number; formatted: { value: string; unit: string } };
}

export function Rank() {
    const { start, end } = useResolvedTimeRange();
    const { data: rankedData } = useStatsModelRank(start, end);

    const t = useTranslations('home.rank');
    const rankSortMode = useHomeViewStore((state) => state.rankSortMode);
    const setRankSortMode = useHomeViewStore((state) => state.setRankSortMode);

    const models = useMemo<NormalizedModel[]>(() => {
        if (!rankedData) return [];
        return rankedData.map((item: ModelRankItemFormatted) => ({
            name: item.model_name,
            channel_name: item.channel_name,
            total_cost: item.total_cost,
            request_count: item.request_count,
            total_token: item.total_token,
            request_success: item.request_success,
            request_failed: item.request_failed,
        }));
    }, [rankedData]);

    const rankedByCost = useMemo(() => {
        return [...models].sort((a, b) => b.total_cost.raw - a.total_cost.raw);
    }, [models]);

    const rankedByCount = useMemo(() => {
        return [...models].sort((a, b) => b.request_count.raw - a.request_count.raw);
    }, [models]);

    const rankedByTokens = useMemo(() => {
        return [...models].sort((a, b) => b.total_token.raw - a.total_token.raw);
    }, [models]);

    const getMedalEmoji = (rank: number): string => {
        switch (rank) {
            case 1: return '\u{1F947}';
            case 2: return '\u{1F948}';
            case 3: return '\u{1F949}';
            default: return '';
        }
    };

    const renderList = (items: NormalizedModel[], mode: RankSortMode) => {
        if (items.length === 0) {
            return (
                <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                    <TrendingUp className="w-12 h-12 mb-3 opacity-30" />
                    <p className="text-sm">{t('noData')}</p>
                </div>
            );
        }
        return (
            <div className="space-y-3 max-h-[300px] overflow-y-auto">
                {items.map((model, index) => {
                    const rank = index + 1;
                    const medal = getMedalEmoji(rank);

                    return (
                        <div
                            key={`${model.channel_name}:${model.name}`}
                            className="flex items-center gap-3 p-3 rounded-2xl hover:bg-accent/5 transition-colors"
                        >
                            <div className="w-8 h-8 rounded-lg flex items-center justify-center font-bold text-lg shrink-0">
                                {medal || rank}
                            </div>

                            <div className="flex-1 min-w-0">
                                <p className="font-medium text-sm truncate">{model.name}</p>
                                <p className="text-xs text-muted-foreground truncate">{model.channel_name}</p>
                                {mode === 'count' && (() => {
                                    const successCount = model.request_success.raw;
                                    const failedCount = model.request_failed.raw;
                                    const totalCount = successCount + failedCount;
                                    const successRate = totalCount > 0 ? (successCount / totalCount) * 100 : 0;

                                    return (
                                        <div className="flex items-center gap-1 text-xs text-muted-foreground mt-1">
                                            <span>{t('successRate')}:</span>
                                            <span>{successRate.toFixed(1)}%</span>
                                        </div>
                                    );
                                })()}
                            </div>

                            <div className="flex items-center gap-1 text-right shrink-0">
                                {mode === 'count' ? (
                                    <div className="flex items-center gap-1 text-sm font-medium tabular-nums">
                                        <span className="text-accent">
                                            {model.request_success.formatted.value}
                                            <span className="text-xs text-muted-foreground">
                                                {model.request_success.formatted.unit}
                                            </span>
                                        </span>
                                        <span className="text-muted-foreground/40 font-light">/</span>
                                        <span className="text-destructive">
                                            {model.request_failed.formatted.value}
                                            <span className="text-xs text-muted-foreground">
                                                {model.request_failed.formatted.unit}
                                            </span>
                                        </span>
                                    </div>
                                ) : mode === 'tokens' ? (
                                    <span className="font-semibold text-base">
                                        {model.total_token.formatted.value}
                                        <span className="text-xs text-muted-foreground">
                                            {model.total_token.formatted.unit}
                                        </span>
                                    </span>
                                ) : (
                                    <span className="font-semibold text-base">
                                        {model.total_cost.formatted.value}
                                        <span className="text-xs text-muted-foreground">
                                            {model.total_cost.formatted.unit}
                                        </span>
                                    </span>
                                )}
                            </div>
                        </div>
                    );
                })}
            </div>
        );
    };

    return (
        <div className="rounded-3xl bg-card text-card-foreground border-card-border border p-4">
            <Tabs value={rankSortMode} onValueChange={(value) => setRankSortMode(value as RankSortMode)}>
                <div className="flex items-center justify-between">
                    <h3 className="font-semibold text-base">{t('title')}</h3>
                    <TabsList>
                        <TabsTrigger value="cost">{t('sortByCost')}</TabsTrigger>
                        <TabsTrigger value="count">{t('sortByCount')}</TabsTrigger>
                        <TabsTrigger value="tokens">{t('sortByTokens')}</TabsTrigger>
                    </TabsList>
                </div>
                <TabsContents>
                    <TabsContent value="cost">
                        {renderList(rankedByCost, 'cost')}
                    </TabsContent>
                    <TabsContent value="count">
                        {renderList(rankedByCount, 'count')}
                    </TabsContent>
                    <TabsContent value="tokens">
                        {renderList(rankedByTokens, 'tokens')}
                    </TabsContent>
                </TabsContents>
            </Tabs>
        </div>
    );
}
