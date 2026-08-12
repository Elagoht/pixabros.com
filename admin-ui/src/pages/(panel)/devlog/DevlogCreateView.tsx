import { useQueryClient } from "@tanstack/react-query";
import type { FC } from "react";
import { useNavigate } from "react-router-dom";
import { Alert, Container } from "@/components/ui";
import DevlogForm, { emptyDevlogFormValues } from "@/forms/DevlogForm";
import { queryKeys } from "@/lib/query/keys";
import { useI18n } from "@/lib/stores/i18n";
import { DevlogService } from "@/services/devlog";
import { handleRequest } from "@/utilities/request";

const DevlogCreateView: FC = () => {
  const { t } = useI18n();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  return (
    <Container size="md" className="space-y-4 py-6">
      <h1 className="text-xl font-semibold text-gray-800 dark:text-gray-100">
        {t("devlog.create.title")}
      </h1>

      <Alert variant="info" description={t("devlog.create.hint")} />

      <DevlogForm
        initialValues={emptyDevlogFormValues}
        submitLabel={t("devlog.create.submit")}
        onSubmit={async (values) => {
          const { data } = await handleRequest(
            () =>
              DevlogService.create({
                title: values.title,
                content_markdown: values.content_markdown,
                is_published: values.is_published,
                published_at: values.published_at,
              }),
            {
              method: "POST",
              errorMessages: { 400: "devlog.errors.invalidFields" },
              successMessage: "devlog.toast.created",
            },
          );
          if (data) {
            queryClient.invalidateQueries({
              queryKey: queryKeys.devlog.lists(),
            });
            navigate(`/devlog/${data.id}`, { replace: true });
          }
        }}
      />
    </Container>
  );
};

export default DevlogCreateView;
