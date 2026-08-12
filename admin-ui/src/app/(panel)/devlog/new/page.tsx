import type { FC } from "react";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useI18n } from "@/lib/stores/i18n";
import DevlogCreateView from "@/pages/(panel)/devlog/DevlogCreateView";

const DevlogCreatePage: FC = () => {
  const { t } = useI18n();

  useBreadcrumb([
    { label: t("menu.devlog"), to: "/devlog" },
    { label: t("devlog.create.title") },
  ]);

  return <DevlogCreateView />;
};

export default DevlogCreatePage;
