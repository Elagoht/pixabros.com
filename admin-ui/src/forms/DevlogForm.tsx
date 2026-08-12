import { IconArticle, IconRocket } from "@tabler/icons-react";
import { Form, Formik } from "formik";
import type { FC, ReactNode } from "react";
import GameSelect from "@/components/games/GameSelect";
import {
  Card,
  DatePicker,
  Input,
  MarkdownEditor,
  SubmitButton,
  Switch,
} from "@/components/ui";
import { useI18n } from "@/lib/stores/i18n";
import { devlogValidationSchema } from "@/lib/validation/devlog";

export const emptyDevlogFormValues: DevlogFormValues = {
  title: "",
  content_markdown: "",
  game_id: "",
  is_published: false,
  published_at: "",
};

export const toDevlogFormValues = (
  post: ResponseDevlogPost,
): DevlogFormValues => ({
  title: post.title,
  content_markdown: post.content_markdown,
  // The picker works in strings; "no game" is the empty option.
  game_id: post.game_id ?? "",
  is_published: post.is_published,
  published_at: post.published_at,
});

interface DevlogFormProps {
  initialValues: DevlogFormValues;
  submitLabel: string;
  onSubmit: (values: DevlogFormValues) => Promise<void>;
  slugHint?: ReactNode;
}

const DevlogForm: FC<DevlogFormProps> = ({
  initialValues,
  submitLabel,
  onSubmit,
  slugHint,
}) => {
  const { t } = useI18n();

  return (
    <Formik
      initialValues={initialValues}
      validationSchema={devlogValidationSchema(t)}
      onSubmit={onSubmit}
    >
      <Form noValidate className="space-y-4">
        <Card>
          <Card.Header icon={IconArticle}>
            <h2 className="text-sm font-semibold text-gray-800 dark:text-gray-100">
              {t("devlog.form.basics")}
            </h2>
          </Card.Header>
          <Card.Body className="space-y-4">
            <div className="space-y-1.5">
              <Input
                name="title"
                label={t("devlog.form.title")}
                placeholder={t("devlog.form.titlePlaceholder")}
              />
              {slugHint}
            </div>

            <MarkdownEditor
              name="content_markdown"
              label={t("devlog.form.content")}
              rows={16}
              placeholder={t("devlog.form.contentPlaceholder")}
            />

            <div className="space-y-1.5">
              <GameSelect
                name="game_id"
                label={t("devlog.form.game")}
                noneLabel={t("devlog.form.gameNone")}
              />
              <p className="text-xs text-gray-500 dark:text-gray-400">
                {t("devlog.form.gameHelp")}
              </p>
            </div>
          </Card.Body>
        </Card>

        <Card>
          <Card.Header icon={IconRocket}>
            <h2 className="text-sm font-semibold text-gray-800 dark:text-gray-100">
              {t("devlog.form.publishing")}
            </h2>
          </Card.Header>
          <Card.Body className="space-y-4">
            <Switch name="is_published" label={t("devlog.form.published")} />

            <div className="space-y-1.5">
              <DatePicker
                name="published_at"
                label={t("devlog.form.publishedAt")}
              />
              <p className="text-xs text-gray-500 dark:text-gray-400">
                {t("devlog.form.publishedAtHelp")}
              </p>
            </div>
          </Card.Body>
        </Card>

        <SubmitButton variant="default" loadingText={t("common.loading")}>
          {submitLabel}
        </SubmitButton>
      </Form>
    </Formik>
  );
};

export default DevlogForm;
