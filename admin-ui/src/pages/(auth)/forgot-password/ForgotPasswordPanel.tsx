import { IconArrowLeft } from "@tabler/icons-react";
import type { FC } from "react";
import { Alert, Button, Card, CodeBlock } from "@/components/ui";
import { useI18n } from "@/lib/stores/i18n";

const RESET_COMMAND =
  "admincli reset-password -username <username> -password '<new-password>'";

const ForgotPasswordPanel: FC = () => {
  const { t } = useI18n();

  return (
    <section className="w-full max-w-md">
      <Card>
        <Card.Header className="flex-col items-center gap-1 border-b-0 pb-2 pt-6">
          <h1 className="text-xl font-semibold text-primary-500 dark:text-primary-300">
            {t("pages.auth.forgotPassword.title")}
          </h1>
          <p className="text-center text-sm text-gray-500 dark:text-gray-400">
            {t("pages.auth.forgotPassword.subtitle")}
          </p>
        </Card.Header>

        <Card.Body className="space-y-4">
          <Alert
            variant="info"
            description={t("pages.auth.forgotPassword.noSelfService")}
          />

          <div className="space-y-2">
            <p className="text-sm text-gray-600 dark:text-gray-300">
              {t("pages.auth.forgotPassword.instructions")}
            </p>
            <CodeBlock code={RESET_COMMAND} language="bash" />
            <p className="text-xs text-gray-500 dark:text-gray-400">
              {t("pages.auth.forgotPassword.sessionsNote")}
            </p>
          </div>

          <Button
            to="/login"
            variant="outline"
            leftIcon={IconArrowLeft}
            className="w-full"
          >
            {t("pages.auth.forgotPassword.backToLogin")}
          </Button>
        </Card.Body>
      </Card>
    </section>
  );
};

export default ForgotPasswordPanel;
