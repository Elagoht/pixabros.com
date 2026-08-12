import type { FC } from "react";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useI18n } from "@/lib/stores/i18n";
import GameCreateView from "@/pages/(panel)/games/GameCreateView";

const GameCreatePage: FC = () => {
  const { t } = useI18n();

  useBreadcrumb([
    { label: t("menu.games"), to: "/games" },
    { label: t("games.create.title") },
  ]);

  return <GameCreateView />;
};

export default GameCreatePage;
