type Locale = "en" | "tr";

type TranslationValue = string | TranslationObject;

interface TranslationObject {
  [key: string]: TranslationValue;
}

type DeepKeys<T> = T extends string
  ? never
  : {
      [K in keyof T]: K extends string
        ? T[K] extends string
          ? K
          : T[K] extends object
            ? K | `${K}.${DeepKeys<T[K]>}`
            : K
        : never;
    }[keyof T];
