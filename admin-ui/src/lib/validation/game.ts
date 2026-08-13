import * as Yup from "yup";

// Links are edited as rows, so each row is validated on its own -- the
// public site renders both halves, and a row with only one filled in would
// show up there as a broken link.
const externalLinkSchema = (t: TranslateFunction) =>
  Yup.object({
    label: Yup.string().trim().required(t("games.validation.linkLabel")),
    url: Yup.string()
      .trim()
      .url(t("games.validation.linkUrl"))
      .required(t("games.validation.linkUrl")),
  });

export const gameValidationSchema = (t: TranslateFunction) =>
  Yup.object({
    title: Yup.string().trim().required(t("games.validation.titleRequired")),
    short_description: Yup.string(),
    full_description: Yup.string(),
    tags: Yup.string(),
    genre: Yup.string(),
    // The API stores and sorts the date as text, so the shape matters. The
    // picker produces it; this catches a value typed in by hand.
    release_date: Yup.string().matches(/^(\d{4}-\d{2}-\d{2})?$/, {
      message: t("games.validation.releaseDate"),
    }),
    kind: Yup.string().oneOf(["production", "gamejam"]),
    is_for_sale: Yup.boolean(),
    price_display: Yup.string(),
    external_links: Yup.array().of(externalLinkSchema(t)),
    is_published: Yup.boolean(),
  });
