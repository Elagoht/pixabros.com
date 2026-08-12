import type { FC } from "react";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useI18n } from "@/lib/stores/i18n";
import AwardEditView from "@/pages/(panel)/awards/AwardEditView";

const AwardEditPage: FC = () => {
  const { t } = useI18n();

  useBreadcrumb([
    { label: t("menu.awards"), to: "/awards" },
    { label: t("awards.edit.title") },
  ]);

  return <AwardEditView />;
};

export default AwardEditPage;
