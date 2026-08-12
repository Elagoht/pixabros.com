import type { FC } from "react";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useI18n } from "@/lib/stores/i18n";
import AwardCreateView from "@/pages/(panel)/awards/AwardCreateView";

const AwardCreatePage: FC = () => {
  const { t } = useI18n();

  useBreadcrumb([
    { label: t("menu.awards"), to: "/awards" },
    { label: t("awards.create.title") },
  ]);

  return <AwardCreateView />;
};

export default AwardCreatePage;
