import { IconPencil } from "@tabler/icons-react";
import type { FC } from "react";
import { Container } from "@/components/ui";
import UserEditForm from "@/forms/UserEditForm";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useI18n } from "@/lib/stores/i18n";

const UserEditPage: FC = () => {
  const { t } = useI18n();
  useBreadcrumb([
    { label: t("menu.definitions"), to: "/definitions" },
    { label: t("menu.users"), to: "/definitions/users" },
    { label: t("common.edit") },
  ]);

  return (
    <Container size="lg" className="space-y-6 py-6">
      <h1 className="flex items-center gap-3 text-2xl font-bold text-gray-900 dark:text-gray-50">
        <IconPencil size={28} className="text-primary-500" />
        {t("users.edit.title")}
      </h1>

      <UserEditForm />
    </Container>
  );
};

export default UserEditPage;
