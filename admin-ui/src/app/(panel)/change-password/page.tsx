import type { FC } from "react";
import { Container } from "@/components/ui";
import ChangePasswordForm from "@/forms/ChangePasswordForm";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useI18n } from "@/lib/stores/i18n";

const ChangePasswordPage: FC = () => {
  const { t } = useI18n();

  useBreadcrumb([{ label: t("pages.changePassword.title") }]);

  return (
    <Container size="sm" className="py-6">
      <ChangePasswordForm />
    </Container>
  );
};

export default ChangePasswordPage;
