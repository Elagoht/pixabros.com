import { IconPencil } from "@tabler/icons-react";
import type { FC } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";
import { Button, Container } from "@/components/ui";
import UserDetailForm from "@/forms/UserDetailForm";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useI18n } from "@/lib/stores/i18n";

const UserDetailPage: FC = () => {
  const { t } = useI18n();
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();

  useBreadcrumb([
    { label: t("menu.definitions"), to: "/definitions" },
    { label: t("menu.users"), to: "/definitions/users" },
    { label: t("common.detail") },
  ]);

  if (!id) {
    return <Navigate to="/definitions/users" replace />;
  }

  return (
    <Container size="lg" className="space-y-6 py-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-50">
          {t("users.detail.title")}
        </h1>
        <Button
          variant="outline"
          size="sm"
          leftIcon={IconPencil}
          onClick={() => navigate(`/definitions/users/${id}/edit`)}
        >
          {t("users.detail.edit")}
        </Button>
      </div>

      <UserDetailForm />
    </Container>
  );
};

export default UserDetailPage;
