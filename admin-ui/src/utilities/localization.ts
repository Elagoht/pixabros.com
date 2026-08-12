import { useI18n } from "@/lib/stores/i18n";

const LOCALE_MAP: Record<string, string> = {
  tr: "tr-TR",
  en: "en-US",
};

const FLAG_MAP: Record<string, string> = {
  tr: "🇹🇷",
  en: "🇺🇸",
};

const LABEL_MAP: Record<string, string> = {
  tr: "Türkçe",
  en: "English",
};

const DEFAULT_LOCALE = "tr-TR";

export const getLocale = (): string => {
  const { locale } = useI18n.getState();
  return LOCALE_MAP[locale] ?? DEFAULT_LOCALE;
};

export const getFlag = (locale?: string): string => {
  const current = locale ?? useI18n.getState().locale;
  return FLAG_MAP[current] ?? "";
};

export const getLanguageLabel = (locale?: string): string => {
  const current = locale ?? useI18n.getState().locale;
  return LABEL_MAP[current] ?? current;
};

export const getLanguageOptions = () => {
  const locales: Locale[] = ["en", "tr"];
  return locales.map((l) => ({
    label: `${FLAG_MAP[l]} ${LABEL_MAP[l]}`,
    value: l,
  }));
};

export const formatDate = (
  value: unknown,
  options?: DataTableDateOptions,
): string => {
  if (!value) {
    return "";
  }
  const date = new Date(value as string);
  if (Number.isNaN(date.getTime())) {
    return String(value);
  }
  const format = options?.format ?? "date";
  const locale = getLocale();
  switch (format) {
    case "datetime":
      return date.toLocaleDateString(locale, {
        day: "2-digit",
        month: "long",
        year: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      });
    case "time":
      return date.toLocaleTimeString(locale, {
        hour: "2-digit",
        minute: "2-digit",
      });
    default:
      return date.toLocaleDateString(locale, {
        day: "2-digit",
        month: "long",
        year: "numeric",
      });
  }
};

export const formatMoney = (
  value: unknown,
  options?: DataTableMoneyOptions,
): string => {
  const num = Number(value);
  if (Number.isNaN(num)) {
    return String(value ?? "");
  }
  return new Intl.NumberFormat(getLocale(), {
    style: "currency",
    currency: options?.currency ?? "TRY",
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(num);
};

export const formatNumber = (
  value: unknown,
  options?: DataTableNumberOptions,
): string => {
  const num = Number(value);
  if (Number.isNaN(num)) {
    return String(value ?? "");
  }
  if (options?.format === "percent") {
    return `%${num.toFixed(options?.precision ?? 0)}`;
  }
  return new Intl.NumberFormat(getLocale(), {
    minimumFractionDigits: options?.precision ?? 0,
    maximumFractionDigits: options?.precision ?? 0,
  }).format(num);
};
