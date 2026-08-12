import { IconSettings } from "@tabler/icons-react";
import { Form, Formik } from "formik";
import type { FC } from "react";
import ImageUploadField from "@/components/media/ImageUploadField";
import { Card, Input, SubmitButton, Textarea } from "@/components/ui";
import { useI18n } from "@/lib/stores/i18n";
import { settingsValidationSchema } from "@/lib/validation/settings";

interface SettingsFormProps {
  definitions: SettingDefinition[];
  values: Record<string, string>;
  title: string;
  hint: string;
  onSubmit: (values: Record<string, string>) => Promise<void>;
}

// One form for both settings groups. The fields come from the server's
// registry, so adding a setting in Go makes it appear here with no change on
// this side; only its label is looked up locally.
const SettingsForm: FC<SettingsFormProps> = ({
  definitions,
  values,
  title,
  hint,
  onSubmit,
}) => {
  const { t } = useI18n();

  // A key with no translation falls back to the key itself, so a setting
  // added in Go is usable before anyone writes a label for it.
  const labelFor = (key: string) => {
    const translated = t(`settings.labels.${key}` as TranslationKey);
    return translated === `settings.labels.${key}` ? key : translated;
  };

  return (
    <Formik
      initialValues={values}
      validationSchema={settingsValidationSchema(t, definitions)}
      onSubmit={onSubmit}
    >
      {({ values: current, setFieldValue }) => (
        <Form noValidate className="space-y-4">
          <Card>
            <Card.Header icon={IconSettings}>
              <h2 className="text-sm font-semibold text-gray-800 dark:text-gray-100">
                {title}
              </h2>
            </Card.Header>
            <Card.Body className="space-y-4">
              <p className="text-xs text-gray-500 dark:text-gray-400">
                {hint} {t("settings.blankMeansUnset")}
              </p>

              {definitions.map((definition) => {
                if (definition.kind === "media") {
                  return (
                    <ImageUploadField
                      key={definition.key}
                      label={labelFor(definition.key)}
                      hint={definition.target ?? ""}
                      target={definition.target ?? "og_image"}
                      mediaId={current[definition.key] || null}
                      // Unlike the other modules the image is a form value
                      // here, so it is saved with the rest of the form rather
                      // than being page state attached on save.
                      onChange={(mediaId) =>
                        setFieldValue(definition.key, mediaId ?? "")
                      }
                    />
                  );
                }

                if (definition.multiline) {
                  return (
                    <Textarea
                      key={definition.key}
                      name={definition.key}
                      label={labelFor(definition.key)}
                      rows={4}
                    />
                  );
                }

                return (
                  <Input
                    key={definition.key}
                    name={definition.key}
                    type={definition.kind === "uri" ? "url" : "text"}
                    label={labelFor(definition.key)}
                  />
                );
              })}
            </Card.Body>
          </Card>

          <SubmitButton variant="default" loadingText={t("common.loading")}>
            {t("settings.submit")}
          </SubmitButton>
        </Form>
      )}
    </Formik>
  );
};

export default SettingsForm;
