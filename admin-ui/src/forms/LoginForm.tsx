import { Card, Input, SubmitButton } from "@/components/ui";
import { useI18n } from "@/lib/stores/i18n";
import { loginValidationSchema } from "@/lib/validation/auth";
import { sessionService } from "@/services/session";
import { handleRequest } from "@/utilities/request";
import { Form, Formik } from "formik";
import type { FC } from "react";
import { Link, useNavigate } from "react-router-dom";

const LoginForm: FC = () => {
  const { t } = useI18n();
  const navigate = useNavigate();

  return (
    <section className="w-full max-w-md">
      <Card>
        <Card.Header className="flex-col items-center gap-1 border-b-0 pb-2 pt-6">
          <h1 className="text-xl font-semibold text-primary-500 dark:text-primary-300">
            {t("pages.auth.login.title")}
          </h1>
        </Card.Header>

        <Card.Body className="space-y-4">
          <Formik
            initialValues={{ email: "", password: "" }}
            validationSchema={loginValidationSchema(t)}
            onSubmit={async (values) => {
              const { data } = await handleRequest(
                () =>
                  sessionService.create({
                    email: values.email,
                    password: values.password,
                  }),
                {
                  errorMessages: { 400: "errors.invalidCredentials" },
                  method: "POST",
                },
              );

              if (data) {
                navigate(`/login-otp/${data.otp_challenge_id}`, {
                  state: { email: values.email },
                });
              }
            }}
          >
            <Form noValidate className="space-y-4">
              <Input
                name="email"
                placeholder={`${t("auth.email")} *`}
                autoComplete="email"
              />

              <Input
                name="password"
                type="password"
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
