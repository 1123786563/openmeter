import en from '@/i18n/locales/en'
import zhCN from '@/i18n/locales/zh-CN'
import i18next from 'i18next'
import { initReactI18next } from 'react-i18next'

const LANGUAGE_STORAGE_KEY = 'openmeter-admin.language'

export const languages = [
  { code: 'zh-CN', label: '中文' },
  { code: 'en', label: 'English' },
] as const

export type LanguageCode = (typeof languages)[number]['code']

export function setLanguage(code: LanguageCode) {
  localStorage.setItem(LANGUAGE_STORAGE_KEY, code)
  document.documentElement.lang = code
  void i18next.changeLanguage(code)
}

const initialLanguage: LanguageCode =
  (localStorage.getItem(LANGUAGE_STORAGE_KEY) as LanguageCode | null) ?? 'zh-CN'

void i18next.use(initReactI18next).init({
  resources: {
    'zh-CN': { translation: zhCN },
    en: { translation: en },
  },
  lng: initialLanguage,
  fallbackLng: 'en',
  interpolation: { escapeValue: false },
})

document.documentElement.lang = initialLanguage
