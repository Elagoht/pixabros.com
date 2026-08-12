import * as Yup from "yup";

// The schema is built from the server's registry rather than hard-coded, so a
// setting added in Go is validated here without a second list to maintain.
// Every setting may be blank: blank means "not set".
export const settingsValidationSchema = (
  t: TranslateFunction,
  definitions: SettingDefinition[],
) => {
  const shape: Record<string, Yup.StringSchema> = {};

  for (const definition of definitions) {
    shape[definition.key] =
      definition.kind === "uri"
        ? Yup.string().trim().url(t("settings.validation.uri"))
        : Yup.string();
  }

  return Yup.object(shape);
};
