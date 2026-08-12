import type { FC } from "react";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useI18n } from "@/lib/stores/i18n";
import GamesListView from "@/pages/(panel)/games/GamesListView";

const GamesPage: FC = () => {
  const { t } = useI18n();

  useBreadcrumb([{ label: t("menu.games") }]);

  return <GamesListView />;
};

export default GamesPage;
