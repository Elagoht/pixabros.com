import { IconSettings } from "@tabler/icons-react";
import { Form, Formik } from "formik";
import type { FC } from "react";
import ImageUploadField from "@/components/media/ImageUploadField";
import {
  Card,
  Input,
  SubmitButton,
  Textarea,
  UrlListField,
} from "@/components/ui";
import { useI18n } from "@/lib/stores/i18n";
import { settingsValidationSchema } from "@/lib/validation/settings";

interface SettingsFormProps {
  definitions: SettingDefinition[];
  values: Record<string, string>;
  title: string;
  hint: string;
  onSubmit: (values: Record<string, string>) => Promise<void>;
}

// The API stores a uri_list as JSON text, so anything unparsable (or a shape
// that is not a list of strings) degrades to an empty list rather than
// crashing the settings screen on a value someone hand-edited earlier.
export const parseUrlList = (raw: string): string[] => {
  if (!raw.trim()) {
    return [];
  }
  try {
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      return [];
    }
    return parsed.filter((entry): entry is string => typeof entry === "string");
  } catch {
    return [];
  }
};

// Turns the API's flat string map into the shape the form edits, and back
// again on submit.
const toFormValues = (
  definitions: SettingDefinition[],
  values: Record<string, string>,
): SettingsFormValues => {
  const formValues: SettingsFormValues = { ...values };
  for (const definition of definitions) {
    if (definition.kind === "uri_list") {
      formValues[definition.key] = parseUrlList(values[definition.key] ?? "");
    }
  }
  return formValues;
};

const toApiValues = (
  definitions: SettingDefinition[],
  formValues: SettingsFormValues,
): Record<string, string> => {
  const values: Record<string, string> = {};
  for (const definition of definitions) {
    const value = formValues[definition.key];
    if (definition.kind === "uri_list") {
      const entries = Array.isArray(value) ? value : [];
      // Blank rows are dropped rather than stored as empty strings, which
      // would be invalid entries in the site's structured data.
      const filled = entries.map((entry) => entry.trim()).filter(Boolean);
      values[definition.key] = filled.length > 0 ? JSON.stringify(filled) : "";
      continue;
    }
    values[definition.key] = typeof value === "string" ? value : "";
  }
  return values;
};

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

  // Settings are not self-explanatory -- "Studio description" says what shape
  // the value has, not where it ends up -- so each one may carry a line saying
  // what it is for. Missing hints simply do not render.
  const optional = (prefix: string) => (key: string) => {
    const lookup = `settings.${prefix}.${key}` as TranslationKey;
    const translated = t(lookup);
    return translated === lookup ? "" : translated;
  };
  const hintFor = optional("hints");
  const placeholderFor = optional("placeholders");

  return (
    <Formik
      initialValues={toFormValues(definitions, values)}
      validationSchema={settingsValidationSchema(t, definitions)}
      onSubmit={(formValues) => onSubmit(toApiValues(definitions, formValues))}
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
                      mediaId={
                        typeof current[definition.key] === "string"
                          ? (current[definition.key] as string) || null
                          : null
                      }
                      // Unlike the other modules the image is a form value
                      // here, so it is saved with the rest of the form rather
                      // than being page state attached on save.
                      onChange={(mediaId) =>
                        setFieldValue(definition.key, mediaId ?? "")
                      }
                    />
                  );
                }

                if (definition.kind === "uri_list") {
                  return (
                    <UrlListField
                      key={definition.key}
                      name={definition.key}
                      label={labelFor(definition.key)}
                      placeholder={t("settings.urlList.placeholder")}
                      addLabel={t("settings.urlList.add")}
                      emptyLabel={t("settings.urlList.empty")}
                      removeLabel={t("common.delete")}
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
                  <div key={definition.key} className="space-y-1.5">
                    <Input
                      name={definition.key}
                      // A link may be a path, and type="url" would have the
                      // browser refuse one before the schema is consulted.
                      type={definition.kind === "uri" ? "url" : "text"}
                      label={labelFor(definition.key)}
                      placeholder={placeholderFor(definition.key)}
                    />
                    {hintFor(definition.key) && (
                      <p className="text-xs text-gray-500 dark:text-gray-400">
                        {hintFor(definition.key)}
                      </p>
                    )}
                  </div>
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
