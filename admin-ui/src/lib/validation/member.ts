import * as Yup from "yup";

// Links are validated per row: the public site renders both halves, so a row
// with only one filled in would show up there as a broken link.
const linkSchema = (t: TranslateFunction) =>
  Yup.object({
    label: Yup.string().trim().required(t("members.validation.linkLabel")),
    url: Yup.string()
      .trim()
      .url(t("members.validation.linkUrl"))
      .required(t("members.validation.linkUrl")),
  });

export const memberValidationSchema = (t: TranslateFunction) =>
  Yup.object({
    name: Yup.string().trim().required(t("members.validation.nameRequired")),
    tags: Yup.string(),
    description: Yup.string(),
    links: Yup.array().of(linkSchema(t)),
    is_published: Yup.boolean(),
  });
