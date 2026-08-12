import type { FC } from "react";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useI18n } from "@/lib/stores/i18n";
import MembersListView from "@/pages/(panel)/members/MembersListView";

const MembersPage: FC = () => {
  const { t } = useI18n();

  useBreadcrumb([{ label: t("menu.members") }]);

  return <MembersListView />;
};

export default MembersPage;
