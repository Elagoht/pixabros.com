import * as Yup from "yup";

// The date column is TEXT and the list is ordered by it as a string, so a
// differently shaped date would sort into the wrong place. This mirrors the
// server's own check.
const ISO_DATE = /^\d{4}-\d{2}-\d{2}$/;

export const awardValidationSchema = (t: TranslateFunction) =>
  Yup.object({
    title: Yup.string().trim().required(t("awards.validation.titleRequired")),
    issuer: Yup.string().trim().required(t("awards.validation.issuerRequired")),
    date: Yup.string()
      .required(t("awards.validation.dateRequired"))
      .matches(ISO_DATE, t("awards.validation.dateFormat")),
    // Optional, but a half-typed address would render as a broken link.
    link: Yup.string().trim().url(t("awards.validation.linkUrl")),
    game_id: Yup.string(),
  });
