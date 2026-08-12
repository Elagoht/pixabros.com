import type { FC } from "react";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useI18n } from "@/lib/stores/i18n";
import AwardsListView from "@/pages/(panel)/awards/AwardsListView";

const AwardsPage: FC = () => {
  const { t } = useI18n();

  useBreadcrumb([{ label: t("menu.awards") }]);

  return <AwardsListView />;
};

export default AwardsPage;
