import { IconCheck } from "@tabler/icons-react";
import type { FC } from "react";
import { Button, Dropdown } from "@/components/ui";
import { useI18n } from "@/lib/stores/i18n";
import { getFlag, getLanguageOptions } from "@/utilities/localization";

// The panel had no way to change language at all -- only the login screen did,
// so the choice was unreachable once you were signed in. The store already
// persisted the locale to a cookie and updated <html lang>; this only exposes
// it.
const LanguageSwitcher: FC = () => {
  const { locale, setLocale } = useI18n();

  const items = getLanguageOptions().map((option) => ({
    id: option.value,
    label: option.label,
    // A tick on the active language, so the menu says which one is on without
    // needing a second visual language.
    icon: option.value === locale ? IconCheck : undefined,
    onClick: () => setLocale(option.value),
  }));

  return (
    <Dropdown
      align="right"
      trigger={
        <Button
          variant="ghost"
          aria-label={getLanguageLabelForCurrent(locale)}
          className="!rounded-xl !px-2 !py-1.5 text-lg leading-none hover:!bg-gray-100 dark:hover:!bg-white/10"
        >
          {getFlag(locale)}
        </Button>
      }
      items={items}
    />
  );
};

// Kept separate so the trigger's accessible name is the language name rather
// than the flag emoji, which screen readers announce as a country.
const getLanguageLabelForCurrent = (locale: Locale): string =>
  getLanguageOptions().find((option) => option.value === locale)?.label ??
  locale;

export default LanguageSwitcher;
