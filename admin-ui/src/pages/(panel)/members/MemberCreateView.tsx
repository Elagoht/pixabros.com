import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { FC } from "react";
import { useNavigate } from "react-router-dom";
import { Alert, Container } from "@/components/ui";
import MemberForm, { emptyMemberFormValues } from "@/forms/MemberForm";
import { queryKeys } from "@/lib/query/keys";
import { useI18n } from "@/lib/stores/i18n";
import { MemberService } from "@/services/member";
import { handleRequest } from "@/utilities/request";

const MemberCreateView: FC = () => {
  const { t } = useI18n();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  // A new member goes to the end of the list rather than tying with everyone
  // else on display_order 0. Usually already cached by the list page.
  const { data: members = [] } = useQuery({
    queryKey: queryKeys.members.list(),
    queryFn: () => MemberService.list(),
  });

  return (
    <Container size="md" className="space-y-4 py-6">
      <h1 className="text-xl font-semibold text-gray-800 dark:text-gray-100">
        {t("members.create.title")}
      </h1>

      <Alert variant="info" description={t("members.create.hint")} />

      <MemberForm
        initialValues={emptyMemberFormValues}
        submitLabel={t("members.create.submit")}
        onSubmit={async (values) => {
          const { data } = await handleRequest(
            () =>
              MemberService.create({
                name: values.name,
                tags: values.tags,
                description: values.description,
                links_json: JSON.stringify(values.links),
                display_order: members.length,
                is_published: values.is_published,
              }),
            {
              method: "POST",
              errorMessages: { 400: "members.errors.nameRequired" },
              successMessage: "members.toast.created",
            },
          );
          if (data) {
            queryClient.invalidateQueries({
              queryKey: queryKeys.members.lists(),
            });
            navigate(`/members/${data.id}`, { replace: true });
          }
        }}
      />
    </Container>
  );
};

export default MemberCreateView;
