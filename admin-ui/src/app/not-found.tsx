import type { FC } from "react";
import { Link } from "react-router-dom";
import { Container } from "@/components/ui";
import { useI18n } from "@/lib/stores/i18n";

const NotFoundPage: FC = () => {
  const { t } = useI18n();

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 dark:bg-gray-950">
      <Container size="sm" className="text-center">
        <h1 className="text-6xl font-bold text-primary-500">404</h1>
        <p className="mt-4 text-lg font-semibold text-gray-900 dark:text-gray-50">
          {t("pages.notFound.subtitle")}
        </p>
        <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
          {t("pages.notFound.description")}
        </p>
        <div className="mt-6 flex items-center justify-center gap-3">
          <Link
            to="/"
            className="inline-flex items-center rounded-md bg-primary-500 px-4 py-2 text-sm font-medium text-white transition hover:bg-primary-600"
          >
            {t("pages.notFound.backHome")}
          </Link>
        </div>
      </Container>
    </div>
  );
};

export default NotFoundPage;
