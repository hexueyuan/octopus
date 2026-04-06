'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { useModelList, useUpdateModel, useCreateModel, type LLMInfo } from '@/api/endpoints/model';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import {
    Accordion,
    AccordionContent,
    AccordionItem,
    AccordionTrigger,
} from '@/components/ui/accordion';
import { toast } from '@/components/common/Toast';
import { useTranslations } from 'next-intl';
import { convertCurrency, getCurrencySymbol } from '@/lib/utils';

interface PriceRow {
    input: string;
    output: string;
    cache_read: string;
    cache_write: string;
}

export function ModelPriceEditor({ modelNames }: { modelNames: string[] }) {
    const t = useTranslations('channel.form');
    const { data: models } = useModelList();
    const updateModel = useUpdateModel();
    const createModel = useCreateModel();

    const [prices, setPrices] = useState<Map<string, PriceRow>>(new Map());
    const [saving, setSaving] = useState(false);
    const initializedRef = useRef<string>('');

    // Build a lookup from API data (keyed by lowercase for case-insensitive matching)
    const modelMap = useRef<Map<string, LLMInfo>>(new Map());
    useEffect(() => {
        if (!models) return;
        const m = new Map<string, LLMInfo>();
        for (const model of models) {
            m.set(model.name.toLowerCase(), model);
        }
        modelMap.current = m;
    }, [models]);

    // Initialize prices when modelNames change or models first load
    useEffect(() => {
        if (!models) return;
        const key = modelNames.join(',') + '|' + models.length;
        if (key === initializedRef.current) return;
        initializedRef.current = key;

        const next = new Map<string, PriceRow>();
        const symbol = getCurrencySymbol();
        const isCNY = symbol === '\u00A5';

        for (const name of modelNames) {
            const existing = modelMap.current.get(name.toLowerCase());
            if (existing) {
                next.set(name, {
                    input: formatPrice(existing.input, isCNY),
                    output: formatPrice(existing.output, isCNY),
                    cache_read: formatPrice(existing.cache_read, isCNY),
                    cache_write: formatPrice(existing.cache_write, isCNY),
                });
            } else {
                next.set(name, { input: '', output: '', cache_read: '', cache_write: '' });
            }
        }
        setPrices(next);
    }, [modelNames, models]);

    const handleChange = useCallback((name: string, field: keyof PriceRow, value: string) => {
        setPrices((prev) => {
            const next = new Map(prev);
            const row = next.get(name) ?? { input: '', output: '', cache_read: '', cache_write: '' };
            next.set(name, { ...row, [field]: value });
            return next;
        });
    }, []);

    const handleSave = async () => {
        const symbol = getCurrencySymbol();
        const isCNY = symbol === '\u00A5';
        const { exchangeRate } = await import('@/stores/setting').then(m => {
            const state = m.useSettingStore.getState();
            return { exchangeRate: state.exchangeRate };
        });

        const mutations: Promise<unknown>[] = [];
        let changedCount = 0;

        for (const [name, row] of prices) {
            const existing = modelMap.current.get(name.toLowerCase());
            const toUSD = (val: string) => {
                const num = parseFloat(val) || 0;
                return isCNY && exchangeRate > 0 ? num / exchangeRate : num;
            };

            const newData: LLMInfo = {
                name: existing ? existing.name : name,
                input: toUSD(row.input),
                output: toUSD(row.output),
                cache_read: toUSD(row.cache_read),
                cache_write: toUSD(row.cache_write),
            };

            if (existing) {
                const changed =
                    Math.abs(newData.input - existing.input) > 1e-10 ||
                    Math.abs(newData.output - existing.output) > 1e-10 ||
                    Math.abs(newData.cache_read - existing.cache_read) > 1e-10 ||
                    Math.abs(newData.cache_write - existing.cache_write) > 1e-10;
                if (changed) {
                    changedCount++;
                    mutations.push(updateModel.mutateAsync(newData));
                }
            } else {
                const hasValue = newData.input || newData.output || newData.cache_read || newData.cache_write;
                if (hasValue) {
                    changedCount++;
                    mutations.push(createModel.mutateAsync(newData));
                }
            }
        }

        if (changedCount === 0) {
            toast.info(t('modelPriceNoChanges'));
            return;
        }

        setSaving(true);
        try {
            const results = await Promise.allSettled(mutations);
            const failed = results.filter((r) => r.status === 'rejected').length;
            if (failed === 0) {
                toast.success(t('modelPriceSaved'));
            } else {
                toast.error(t('modelPriceSaveFailed'), { description: `${failed}/${changedCount}` });
            }
        } finally {
            setSaving(false);
        }
    };

    if (modelNames.length === 0) return null;

    const currencySymbol = getCurrencySymbol();

    return (
        <Accordion type="single" collapsible className="w-full border rounded-xl bg-card">
            <AccordionItem value="model-price" className="border-none">
                <AccordionTrigger className="text-sm font-medium text-card-foreground py-3 px-4 hover:no-underline hover:bg-muted/30 rounded-xl transition-colors">
                    {t('modelPrice')} ({modelNames.length})
                </AccordionTrigger>
                <AccordionContent className="pt-2 px-4 pb-4 space-y-3 border-t">
                    <div className="overflow-x-auto">
                        <table className="w-full text-sm">
                            <thead>
                                <tr className="text-xs text-muted-foreground">
                                    <th className="text-left py-1.5 pr-2 font-medium">{t('model')}</th>
                                    <th className="text-left py-1.5 px-1 font-medium">{t('modelPriceInput')} ({currencySymbol})</th>
                                    <th className="text-left py-1.5 px-1 font-medium">{t('modelPriceOutput')} ({currencySymbol})</th>
                                    <th className="text-left py-1.5 px-1 font-medium">{t('modelPriceCacheRead')} ({currencySymbol})</th>
                                    <th className="text-left py-1.5 pl-1 font-medium">{t('modelPriceCacheWrite')} ({currencySymbol})</th>
                                </tr>
                            </thead>
                            <tbody>
                                {modelNames.map((name) => {
                                    const row = prices.get(name) ?? { input: '', output: '', cache_read: '', cache_write: '' };
                                    return (
                                        <tr key={name} className="border-t border-border/40">
                                            <td className="py-1.5 pr-2 text-xs text-card-foreground truncate max-w-[200px]" title={name}>
                                                {name}
                                            </td>
                                            <td className="py-1.5 px-1">
                                                <Input
                                                    type="number"
                                                    step="any"
                                                    value={row.input}
                                                    onChange={(e) => handleChange(name, 'input', e.target.value)}
                                                    className="rounded-lg h-7 text-xs"
                                                />
                                            </td>
                                            <td className="py-1.5 px-1">
                                                <Input
                                                    type="number"
                                                    step="any"
                                                    value={row.output}
                                                    onChange={(e) => handleChange(name, 'output', e.target.value)}
                                                    className="rounded-lg h-7 text-xs"
                                                />
                                            </td>
                                            <td className="py-1.5 px-1">
                                                <Input
                                                    type="number"
                                                    step="any"
                                                    value={row.cache_read}
                                                    onChange={(e) => handleChange(name, 'cache_read', e.target.value)}
                                                    className="rounded-lg h-7 text-xs"
                                                />
                                            </td>
                                            <td className="py-1.5 pl-1">
                                                <Input
                                                    type="number"
                                                    step="any"
                                                    value={row.cache_write}
                                                    onChange={(e) => handleChange(name, 'cache_write', e.target.value)}
                                                    className="rounded-lg h-7 text-xs"
                                                />
                                            </td>
                                        </tr>
                                    );
                                })}
                            </tbody>
                        </table>
                    </div>
                    <Button
                        type="button"
                        onClick={handleSave}
                        disabled={saving}
                        className="w-full rounded-xl h-9 text-sm"
                    >
                        {saving ? t('modelPriceSaving') : t('modelPriceSaveAll')}
                    </Button>
                </AccordionContent>
            </AccordionItem>
        </Accordion>
    );
}

function formatPrice(usdValue: number, isCNY: boolean): string {
    if (!usdValue) return '';
    const val = isCNY ? convertCurrency(usdValue) : usdValue;
    return String(val);
}
