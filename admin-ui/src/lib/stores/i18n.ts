import { create } from "zustand";
import enTranslations from "@/i18n/en.json";
import trTranslations from "@/i18n/tr.json";
import { CookieHelper } from "@/utilities/cookie";

const COOKIE_NAME = "locale";
const DEFAULT_LOCALE: Locale = "en";

const translations = {
  en: enTranslations,
  tr: trTranslations,
} as const;

const getInitialLocale = (): Locale => {
  if (typeof document === "undefined") {
    return DEFAULT_LOCALE;
  }

  const cookieLocale = CookieHelper.get(COOKIE_NAME);
  if (cookieLocale === "en" || cookieLocale === "tr") {
    return cookieLocale;
  }

  const browserLang = navigator.language.split("-")[0];
  if (browserLang === "tr") {
    return "tr";
  }

  return DEFAULT_LOCALE;
};

const applyLocale = (locale: Locale): void => {
  document.documentElement.lang = translations[locale].meta.lang;
};

const getByPath = (obj: unknown, path: string): string => {
  return path.split(".").reduce((current: unknown, key: string) => {
    if (current && typeof current === "object" && key in current) {
      return (current as Record<string, unknown>)[key];
    }
    return path;
  }, obj) as string;
};

const interpolate = (
  template: string,
  params?: Record<string, string>,
): string => {
  if (!params) {
    return template;
  }
  return template.replace(/\{\{(\w+)\}\}/g, (_, key) => params[key] || "");
};

const buildT =
  (locale: Locale) =>
  (key: TranslationKey, params?: Record<string, string>): string => {
    const value = getByPath(translations[locale], key as string);
    return interpolate(value, params);
  };

interface I18nState {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: TranslationKey, params?: Record<string, string>) => string;
}

export const useI18n = create<I18nState>()((set) => {
  const initialLocale = getInitialLocale();
  applyLocale(initialLocale);

  return {
    locale: initialLocale,
    t: buildT(initialLocale),

    setLocale: (locale: Locale) => {
      CookieHelper.set(COOKIE_NAME, locale);
      applyLocale(locale);
      set({ locale, t: buildT(locale) });
    },
  };
});

export const t = (
  key: TranslationKey,
  params?: Record<string, string>,
): string => useI18n.getState().t(key, params);

declare global {
  type TranslationKey = DeepKeys<typeof enTranslations>;
  type Translation = typeof enTranslations;
  type TranslateFunction = typeof t;
}
