import type { FC } from "react";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useI18n } from "@/lib/stores/i18n";
import SettingsGroupView from "@/pages/(panel)/settings/SettingsGroupView";

const HomepagePage: FC = () => {
  const { t } = useI18n();

  useBreadcrumb([{ label: t("menu.homepage") }]);

  return <SettingsGroupView group="homepage" />;
};

export default HomepagePage;
