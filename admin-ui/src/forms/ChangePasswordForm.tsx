import { IconKey, IconLock, IconLockPlus } from "@tabler/icons-react";
import { Form, Formik } from "formik";
import type { FC } from "react";
import { useNavigate } from "react-router-dom";
import { Alert, Card, Input, SubmitButton } from "@/components/ui";
import { useAuthStore } from "@/lib/stores/auth";
import { useI18n } from "@/lib/stores/i18n";
import { changePasswordValidationSchema } from "@/lib/validation/auth";
import { SessionService } from "@/services/session";
import { handleRequest } from "@/utilities/request";

const ChangePasswordForm: FC = () => {
  const { t } = useI18n();
  const navigate = useNavigate();
  const { setUser, setAuthenticated } = useAuthStore();

  return (
    <Card>
      <Card.Header icon={IconKey}>
        <h1 className="text-lg font-semibold text-gray-800 dark:text-gray-100">
          {t("pages.changePassword.title")}
        </h1>
      </Card.Header>

      <Card.Body className="space-y-4">
        <Alert
          variant="warning"
          description={t("pages.changePassword.signsOutWarning")}
        />

        <Formik
          initialValues={{
            current_password: "",
            new_password: "",
            confirm_password: "",
          }}
          validationSchema={changePasswordValidationSchema(t)}
          onSubmit={async (values) => {
            const { success } = await handleRequest(
              () =>
                SessionService.changePassword({
                  current_password: values.current_password,
                  new_password: values.new_password,
                }),
              {
                errorMessages: {
                  400: "errors.weakPassword",
                  401: "errors.currentPasswordIncorrect",
                },
                method: "POST",
                successMessage: "pages.changePassword.success",
              },
            );

            if (success) {
              // The API drops every session for this admin, so the cookie in
              // this tab is already dead. Navigate client-side rather than
              // reloading, so the success toast survives the transition.
              setUser(null);
              setAuthenticated(false);
              navigate("/login", { replace: true });
            }
          }}
        >
          <Form noValidate className="space-y-4">
            <Input
              name="current_password"
              type="password"
              leftIcon={IconLock}
              label={t("pages.changePassword.currentPassword")}
              autoComplete="current-password"
            />

            <Input
              name="new_password"
              type="password"
              leftIcon={IconLockPlus}
              label={t("pages.changePassword.newPassword")}
              autoComplete="new-password"
            />

            <Input
              name="confirm_password"
              type="password"
              leftIcon={IconLockPlus}
              label={t("pages.changePassword.confirmPassword")}
              autoComplete="new-password"
            />

            <SubmitButton variant="default" loadingText={t("common.loading")}>
              {t("pages.changePassword.submit")}
            </SubmitButton>
          </Form>
        </Formik>
      </Card.Body>
    </Card>
  );
};

export default ChangePasswordForm;
