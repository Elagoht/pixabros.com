import type { FC } from "react";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useI18n } from "@/lib/stores/i18n";
import DevlogEditView from "@/pages/(panel)/devlog/DevlogEditView";

const DevlogEditPage: FC = () => {
  const { t } = useI18n();

  useBreadcrumb([
    { label: t("menu.devlog"), to: "/devlog" },
    { label: t("devlog.edit.title") },
  ]);

  return <DevlogEditView />;
};

export default DevlogEditPage;
