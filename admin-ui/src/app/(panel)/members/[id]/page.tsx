import type { FC } from "react";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useI18n } from "@/lib/stores/i18n";
import MemberEditView from "@/pages/(panel)/members/MemberEditView";

const MemberEditPage: FC = () => {
  const { t } = useI18n();

  useBreadcrumb([
    { label: t("menu.members"), to: "/members" },
    { label: t("members.edit.title") },
  ]);

  return <MemberEditView />;
};

export default MemberEditPage;
