import type { FC } from "react";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useI18n } from "@/lib/stores/i18n";
import MemberCreateView from "@/pages/(panel)/members/MemberCreateView";

const MemberCreatePage: FC = () => {
  const { t } = useI18n();

  useBreadcrumb([
    { label: t("menu.members"), to: "/members" },
    { label: t("members.create.title") },
  ]);

  return <MemberCreateView />;
};

export default MemberCreatePage;
