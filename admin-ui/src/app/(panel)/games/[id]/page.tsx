import type { FC } from "react";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useI18n } from "@/lib/stores/i18n";
import GameEditView from "@/pages/(panel)/games/GameEditView";

const GameEditPage: FC = () => {
  const { t } = useI18n();

  useBreadcrumb([
    { label: t("menu.games"), to: "/games" },
    { label: t("games.edit.title") },
  ]);

  return <GameEditView />;
};

export default GameEditPage;
