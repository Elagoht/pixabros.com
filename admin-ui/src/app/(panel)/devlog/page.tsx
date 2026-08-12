import type { FC } from "react";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useI18n } from "@/lib/stores/i18n";
import DevlogListView from "@/pages/(panel)/devlog/DevlogListView";

const DevlogPage: FC = () => {
  const { t } = useI18n();

  useBreadcrumb([{ label: t("menu.devlog") }]);

  return <DevlogListView />;
};

export default DevlogPage;
