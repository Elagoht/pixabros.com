import * as Yup from "yup";

// An absolute http(s) address with a host, which is exactly what the server
// checks with url.Parse and IsAbs.
//
// Yup's own .url() is not used: its pattern insists on a dotted domain, so it
// rejects http://localhost:8080 -- the address the site actually runs on while
// it is being built. Refusing to save a real address is a worse failure than
// letting an odd one through, and the server validates it again either way.
const ABSOLUTE_URL = /^https?:\/\/[^\s/?#]+[^\s]*$/i;

// Somewhere to send a visitor: a path on this site, or a whole address
// elsewhere. Mirrors settings.KindLink in Go, including the refusal of a
// protocol-relative "//host/path", which is another site wearing a path's
// clothes.
const PATH_OR_URL = /^(\/(?!\/)[^\s]*|https?:\/\/[^\s/?#]+[^\s]*)$/i;

const absoluteUrl = (t: TranslateFunction) =>
  Yup.string()
    .trim()
    .matches(ABSOLUTE_URL, {
      message: t("settings.validation.uri"),
      excludeEmptyString: true,
    });

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
      shape[definition.key] = absoluteUrl(t);
      continue;
    }
    if (definition.kind === "link") {
      shape[definition.key] = Yup.string()
        .trim()
        .matches(PATH_OR_URL, {
          message: t("settings.validation.link"),
          excludeEmptyString: true,
        });
      continue;
    }
    if (definition.kind === "uri_list") {
      // Each entry is validated on its own so the row that is wrong is the
      // row that shows an error.
      shape[definition.key] = Yup.array().of(absoluteUrl(t));
      continue;
    }
    shape[definition.key] = Yup.string();
  }

  return Yup.object(shape);
};
