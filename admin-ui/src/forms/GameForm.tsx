import { IconInfoCircle, IconWorldWww } from "@tabler/icons-react";
import { Form, Formik } from "formik";
import type { FC, ReactNode } from "react";
import {
  Card,
  DatePicker,
  FieldSet,
  Input,
  Keywords,
  LinkListField,
  Select,
  SubmitButton,
  Switch,
  Textarea,
} from "@/components/ui";
import { useI18n } from "@/lib/stores/i18n";
import { gameValidationSchema } from "@/lib/validation/game";

export const emptyGameFormValues: GameFormValues = {
  title: "",
  short_description: "",
  full_description: "",
  tags: "",
  genre: "",
  release_date: "",
  kind: "production",
  is_for_sale: false,
  price_display: "",
  external_links: [],
  is_published: false,
};

// The API stores external links as raw JSON text, so anything unparsable (or
// a shape that is not a list of links) degrades to an empty list rather than
// crashing the edit screen.
export const parseExternalLinks = (raw: string): GameExternalLink[] => {
  if (!raw.trim()) {
    return [];
  }
  try {
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      return [];
    }
    return parsed
      .filter((item): item is Record<string, unknown> =>
        Boolean(item && typeof item === "object"),
      )
      .map((item) => ({
        label: typeof item.label === "string" ? item.label : "",
        url: typeof item.url === "string" ? item.url : "",
      }));
  } catch {
    return [];
  }
};

export const toGameFormValues = (game: ResponseGame): GameFormValues => ({
  title: game.title,
  short_description: game.short_description,
  full_description: game.full_description,
  tags: game.tags,
  genre: game.genre,
  release_date: game.release_date,
  kind: game.kind,
  is_for_sale: game.is_for_sale,
  price_display: game.price_display,
  external_links: parseExternalLinks(game.external_links_json),
  is_published: game.is_published,
});

interface GameFormProps {
  initialValues: GameFormValues;
  submitLabel: string;
  onSubmit: (values: GameFormValues) => Promise<void>;
  slugHint?: ReactNode;
  /** Rendered after the form's own cards (the edit page's screenshots). */
  extraSections?: ReactNode;
}

const GameForm: FC<GameFormProps> = ({
  initialValues,
  submitLabel,
  onSubmit,
  slugHint,
  extraSections,
}) => {
  const { t } = useI18n();

  return (
    <Formik
      initialValues={initialValues}
      validationSchema={gameValidationSchema(t)}
      onSubmit={onSubmit}
    >
      {({ values }) => (
        <Form noValidate className="space-y-4">
          <Card>
            <Card.Header icon={IconInfoCircle}>
              <h2 className="text-sm font-semibold text-gray-800 dark:text-gray-100">
                {t("games.form.basics")}
              </h2>
            </Card.Header>
            <Card.Body className="space-y-4">
              <div className="space-y-1.5">
                <Input
                  name="title"
                  label={t("games.form.title")}
                  placeholder={t("games.form.titlePlaceholder")}
                />
                {slugHint}
              </div>

              <Textarea
                name="short_description"
                label={t("games.form.shortDescription")}
                rows={2}
              />

              <Textarea
                name="full_description"
                label={t("games.form.fullDescription")}
                rows={6}
              />

              <Keywords
                name="tags"
                label={t("games.form.tags")}
                placeholder={t("games.form.tagsPlaceholder")}
                output="string"
              />

              <div className="grid gap-3 sm:grid-cols-2">
                <Input
                  name="genre"
                  label={t("games.form.genre")}
                  placeholder={t("games.form.genrePlaceholder")}
                />
                <DatePicker
                  name="release_date"
                  label={t("games.form.releaseDate")}
                />
              </div>

              <div className="space-y-1.5">
                <Select
                  name="kind"
                  label={t("games.form.kind")}
                  options={[
                    { label: t("games.kind.production"), value: "production" },
                    { label: t("games.kind.gamejam"), value: "gamejam" },
                  ]}
                />
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  {t("games.form.kindHelp")}
                </p>
              </div>
            </Card.Body>
          </Card>

          <Card>
            <Card.Header icon={IconWorldWww}>
              <h2 className="text-sm font-semibold text-gray-800 dark:text-gray-100">
                {t("games.form.availability")}
              </h2>
            </Card.Header>
            <Card.Body className="space-y-4">
              <div className="grid gap-3 sm:grid-cols-2">
                <Switch name="is_published" label={t("games.form.published")} />
                <Switch name="is_for_sale" label={t("games.form.forSale")} />
              </div>

              {/* A price label is meaningless unless the game is actually
                  for sale, so the field only exists while that is on. */}
              {values.is_for_sale && (
                <Input
                  name="price_display"
                  label={t("games.form.priceDisplay")}
                  placeholder={t("games.form.priceDisplayPlaceholder")}
                />
              )}

              <FieldSet
                legend={t("games.form.externalLinks")}
                description={t("games.form.externalLinksHelp")}
              >
                <LinkListField
                  name="external_links"
                  labelPlaceholder={t("games.form.linkLabelPlaceholder")}
                  urlPlaceholder={t("games.form.linkUrlPlaceholder")}
                  addLabel={t("games.form.addLink")}
                  emptyLabel={t("games.form.noLinks")}
                  removeLabel={t("common.delete")}
                />
              </FieldSet>
            </Card.Body>
          </Card>

          {extraSections}

          <SubmitButton variant="default" loadingText={t("common.loading")}>
            {submitLabel}
          </SubmitButton>
        </Form>
      )}
    </Formik>
  );
};

export default GameForm;
