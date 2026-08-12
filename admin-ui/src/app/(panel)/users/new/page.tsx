import { IconUserPlus } from "@tabler/icons-react";
import type { FC } from "react";
import { Card, Container } from "@/components/ui";
import UserCreateForm from "@/forms/UserCreateForm";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useI18n } from "@/lib/stores/i18n";

const UserCreatePage: FC = () => {
  const { t } = useI18n();
  useBreadcrumb([
    { label: t("menu.definitions"), to: "/definitions" },
    { label: t("menu.users"), to: "/definitions/users" },
    { label: t("common.create") },
  ]);

  return (
    <Container size="lg" className="space-y-6 py-6">
      <h1 className="flex items-center gap-3 text-2xl font-bold text-gray-900 dark:text-gray-50">
        <IconUserPlus size={28} className="text-primary-500" />
        {t("users.create.title")}
      </h1>

      <Card>
        <Card.Body>
          <UserCreateForm />
        </Card.Body>
      </Card>
    </Container>
  );
};

export default UserCreatePage;
