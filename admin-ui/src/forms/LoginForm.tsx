import { IconLock, IconUser } from "@tabler/icons-react";
import { Form, Formik } from "formik";
import type { FC } from "react";
import { Link } from "react-router-dom";
import { Card, Input, SubmitButton } from "@/components/ui";
import { useLoginRedirect } from "@/hooks/useLoginRedirect";
import { useAuthStore } from "@/lib/stores/auth";
import { useI18n } from "@/lib/stores/i18n";
import { loginValidationSchema } from "@/lib/validation/auth";
import { SessionService } from "@/services/session";
import { handleRequest } from "@/utilities/request";

const LoginForm: FC = () => {
  const { t } = useI18n();
  const { setSession } = useAuthStore();
  const redirectAfterLogin = useLoginRedirect();

  return (
    <section className="w-full max-w-md">
      <Card>
        <Card.Header className="flex-col items-center gap-1 border-b-0 pb-2 pt-6">
          <h1 className="text-xl font-semibold text-primary-500 dark:text-primary-300">
            {t("pages.auth.login.title")}
          </h1>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            {t("pages.auth.login.subtitle")}
          </p>
        </Card.Header>

        <Card.Body className="space-y-4">
          <Formik
            initialValues={{ username: "", password: "" }}
            validationSchema={loginValidationSchema(t)}
            onSubmit={async (values) => {
              const { data } = await handleRequest(
                () => SessionService.create(values),
                {
                  errorMessages: {
                    400: "errors.invalidCredentials",
                    401: "errors.invalidCredentials",
                  },
                  method: "POST",
                  showSuccessMessage: false,
                },
              );

              if (data) {
                setSession(data);
                redirectAfterLogin();
              }
            }}
          >
            <Form noValidate className="space-y-4">
              <Input
                name="username"
                leftIcon={IconUser}
                placeholder={`${t("auth.username")} *`}
                autoComplete="username"
                autoFocus
              />

              <Input
                name="password"
                type="password"
                leftIcon={IconLock}
                placeholder={`${t("auth.password")} *`}
                autoComplete="current-password"
              />

              <p className="flex justify-end">
                <Link
                  to="/forgot-password"
                  className="text-xs font-medium text-primary-500 hover:underline dark:text-primary-300"
                >
                  {t("auth.forgotPassword")}
                </Link>
              </p>

              <SubmitButton
                className="w-full"
                variant="default"
                loadingText={t("common.loading")}
              >
                {t("auth.login")}
              </SubmitButton>
            </Form>
          </Formik>
        </Card.Body>
      </Card>
    </section>
  );
};

export default LoginForm;
