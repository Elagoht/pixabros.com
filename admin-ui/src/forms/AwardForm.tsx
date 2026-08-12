import { IconTrophy } from "@tabler/icons-react";
import { useQuery } from "@tanstack/react-query";
import { Form, Formik } from "formik";
import type { FC } from "react";
import { Card, DatePicker, Input, Select, SubmitButton } from "@/components/ui";
import { queryKeys } from "@/lib/query/keys";
import { useI18n } from "@/lib/stores/i18n";
import { awardValidationSchema } from "@/lib/validation/award";
import { GameService } from "@/services/game";

export const emptyAwardFormValues: AwardFormValues = {
  title: "",
  issuer: "",
  date: "",
  link: "",
  game_id: "",
};

export const toAwardFormValues = (award: ResponseAward): AwardFormValues => ({
  title: award.title,
  issuer: award.issuer,
  date: award.date,
  link: award.link,
  // The picker works in strings; "no game" is the empty option.
  game_id: award.game_id ?? "",
});

interface AwardFormProps {
  initialValues: AwardFormValues;
  submitLabel: string;
  onSubmit: (values: AwardFormValues) => Promise<void>;
}

const AwardForm: FC<AwardFormProps> = ({
  initialValues,
  submitLabel,
  onSubmit,
}) => {
  const { t } = useI18n();

  // Awards can name a related game, so the picker needs the game list. It is
  // usually already cached by the games screens.
  const { data: games = [] } = useQuery({
    queryKey: queryKeys.games.list(),
    queryFn: () => GameService.list(),
  });

  const gameOptions = [
    { label: t("awards.form.gameNone"), value: "" },
    ...games.map((game) => ({ label: game.title, value: game.id })),
  ];

  return (
    <Formik
      initialValues={initialValues}
      validationSchema={awardValidationSchema(t)}
      onSubmit={onSubmit}
    >
      <Form noValidate className="space-y-4">
        <Card>
          <Card.Header icon={IconTrophy}>
            <h2 className="text-sm font-semibold text-gray-800 dark:text-gray-100">
              {t("awards.form.basics")}
            </h2>
          </Card.Header>
          <Card.Body className="space-y-4">
            <Input
              name="title"
              label={t("awards.form.title")}
              placeholder={t("awards.form.titlePlaceholder")}
            />

            <Input
              name="issuer"
              label={t("awards.form.issuer")}
              placeholder={t("awards.form.issuerPlaceholder")}
            />

            <DatePicker name="date" label={t("awards.form.date")} />

            <div className="space-y-1.5">
              <Select
                name="game_id"
                label={t("awards.form.game")}
                options={gameOptions}
              />
              <p className="text-xs text-gray-500 dark:text-gray-400">
                {t("awards.form.gameHelp")}
              </p>
            </div>

            <div className="space-y-1.5">
              <Input
                name="link"
                label={t("awards.form.link")}
                placeholder={t("awards.form.linkPlaceholder")}
              />
              <p className="text-xs text-gray-500 dark:text-gray-400">
                {t("awards.form.linkHelp")}
              </p>
            </div>
          </Card.Body>
        </Card>

        <SubmitButton variant="default" loadingText={t("common.loading")}>
          {submitLabel}
        </SubmitButton>
      </Form>
    </Formik>
  );
};

export default AwardForm;
