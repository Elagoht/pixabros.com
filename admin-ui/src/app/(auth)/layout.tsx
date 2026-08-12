import GuestGuard from "@/components/guards/GuestGuard";
import { Button } from "@/components/ui";
import { useI18n } from "@/lib/stores/i18n";
import { getFlag, getLanguageLabel } from "@/utilities/localization";
import classNames from "classnames";
import type { FC } from "react";
import { Outlet } from "react-router-dom";

const LANGUAGES: Locale[] = ["en", "tr"];

const AuthLayout: FC = () => {
  const { locale, setLocale } = useI18n();

  return (
    <GuestGuard>
      <main className="flex h-screen">
        <section className="flex w-full flex-col overflow-hidden">
          <div className="flex items-center justify-end px-4 pt-4">
            <nav className="inline-flex rounded-lg border border-gray-200 bg-gray-50 p-0.5 dark:border-gray-700 dark:bg-gray-800">
              {LANGUAGES.map((lang) => (
                <Button
                  key={lang}
                  variant="ghost"
                  size="sm"
                  onClick={() => setLocale(lang)}
                  className={classNames(
                    "!rounded-md",
                    locale === lang
                      ? "!bg-white !text-gray-900 !shadow-sm dark:!bg-gray-700 dark:!text-gray-50"
                      : "!text-gray-500 hover:!text-gray-700 dark:!text-gray-400 dark:hover:!text-gray-200",
                  )}
                >
                  {getFlag(lang)} {getLanguageLabel(lang)}
                </Button>
              ))}
            </nav>
          </div>

          <div className="flex-1 overflow-y-auto px-4 py-12">
            <div className="flex min-h-full items-center justify-center">
              <Outlet />
            </div>
          </div>
        </section>
      </main>
    </GuestGuard>
  );
};

export default AuthLayout;
