import * as Yup from "yup";

// The schema is built from the server's registry rather than hard-coded, so a
// setting added in Go is validated here without a second list to maintain.
// Every setting may be blank: blank means "not set".
export const settingsValidationSchema = (
  t: TranslateFunction,
  definitions: SettingDefinition[],
) => {
  const shape: Record<string, Yup.Schema> = {};

  for (const definition of definitions) {
    if (definition.kind === "uri") {
      shape[definition.key] = Yup.string()
        .trim()
        .url(t("settings.validation.uri"));
      continue;
    }
    if (definition.kind === "uri_list") {
      // Each entry is validated on its own so the row that is wrong is the
      // row that shows an error.
      shape[definition.key] = Yup.array().of(
        Yup.string().trim().url(t("settings.validation.uri")),
      );
      continue;
    }
    shape[definition.key] = Yup.string();
  }

  return Yup.object(shape);
};
