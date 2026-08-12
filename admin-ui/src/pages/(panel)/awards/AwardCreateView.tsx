import { useQueryClient } from "@tanstack/react-query";
import type { FC } from "react";
import { useNavigate } from "react-router-dom";
import { Alert, Container } from "@/components/ui";
import AwardForm, { emptyAwardFormValues } from "@/forms/AwardForm";
import { queryKeys } from "@/lib/query/keys";
import { useI18n } from "@/lib/stores/i18n";
import { AwardService } from "@/services/award";
import { handleRequest } from "@/utilities/request";

const AwardCreateView: FC = () => {
  const { t } = useI18n();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  return (
    <Container size="md" className="space-y-4 py-6">
      <h1 className="text-xl font-semibold text-gray-800 dark:text-gray-100">
        {t("awards.create.title")}
      </h1>

      <Alert variant="info" description={t("awards.create.hint")} />

      <AwardForm
        initialValues={emptyAwardFormValues}
        submitLabel={t("awards.create.submit")}
        onSubmit={async (values) => {
          const { data } = await handleRequest(
            () =>
              AwardService.create({
                title: values.title,
                issuer: values.issuer,
                date: values.date,
                link: values.link,
              }),
            {
              method: "POST",
              errorMessages: { 400: "awards.errors.invalidFields" },
              successMessage: "awards.toast.created",
            },
          );
          if (data) {
            queryClient.invalidateQueries({
              queryKey: queryKeys.awards.lists(),
            });
            navigate(`/awards/${data.id}`, { replace: true });
          }
        }}
      />
    </Container>
  );
};

export default AwardCreateView;
