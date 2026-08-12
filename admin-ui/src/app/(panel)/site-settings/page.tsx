import type { FC } from "react";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useI18n } from "@/lib/stores/i18n";
import SettingsGroupView from "@/pages/(panel)/settings/SettingsGroupView";

const SiteSettingsPage: FC = () => {
  const { t } = useI18n();

  useBreadcrumb([{ label: t("menu.siteSettings") }]);

  return <SettingsGroupView group="site" />;
};

export default SiteSettingsPage;
