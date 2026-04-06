import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export type Locale = 'zh_hans' | 'zh_hant' | 'en';
export type Currency = 'USD' | 'CNY';

export const DEFAULT_USD_TO_CNY_RATE = 7.3;

interface SettingState {
    locale: Locale;
    setLocale: (locale: Locale) => void;
    currency: Currency;
    setCurrency: (currency: Currency) => void;
    exchangeRate: number;
    setExchangeRate: (rate: number) => void;
}

export const useSettingStore = create<SettingState>()(
    persist(
        (set) => ({
            locale: 'zh_hans',
            setLocale: (locale) => set({ locale }),
            currency: 'USD',
            setCurrency: (currency) => set({ currency }),
            exchangeRate: DEFAULT_USD_TO_CNY_RATE,
            setExchangeRate: (rate) => set({ exchangeRate: rate }),
        }),
        {
            name: 'octopus-settings',
        }
    )
);

