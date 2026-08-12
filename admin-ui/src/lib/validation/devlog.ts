import * as Yup from "yup";

// published_at is stored as TEXT and ordered as a string, so a differently
// shaped date would sort into the wrong place. Mirrors the server's check.
const ISO_DATE = /^\d{4}-\d{2}-\d{2}$/;

export const devlogValidationSchema = (t: TranslateFunction) =>
  Yup.object({
    title: Yup.string().trim().required(t("devlog.validation.titleRequired")),
    content_markdown: Yup.string(),
    game_id: Yup.string(),
    is_published: Yup.boolean(),
    // Optional: the server stamps a date the first time a post is published.
    published_at: Yup.string().matches(ISO_DATE, {
      message: t("devlog.validation.dateFormat"),
      excludeEmptyString: true,
    }),
  });
