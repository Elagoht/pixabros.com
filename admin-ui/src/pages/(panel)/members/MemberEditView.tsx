import { IconAlertTriangle, IconPhoto, IconTrash } from "@tabler/icons-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type FC, useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import ImageUploadField from "@/components/media/ImageUploadField";
import {
  Alert,
  Button,
  Card,
  Container,
  EmptyState,
  Loading,
} from "@/components/ui";
import MemberForm, { toMemberFormValues } from "@/forms/MemberForm";
import { queryKeys } from "@/lib/query/keys";
import { useI18n } from "@/lib/stores/i18n";
import { MemberService } from "@/services/member";
import { handleRequest } from "@/utilities/request";

const MemberEditView: FC = () => {
  const { t } = useI18n();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { id: memberId = "" } = useParams();

  const [confirmingDelete, setConfirmingDelete] = useState(false);
  // The avatar id is page state, not a form field: uploading an image must
  // not save the rest of a half-edited form, but PUT is a full replace so the
  // id still has to travel with every save.
  const [avatarId, setAvatarId] = useState<string | null>(null);

  const {
    data: member,
    isLoading,
    isError,
  } = useQuery({
    queryKey: queryKeys.members.detail(memberId),
    queryFn: () => MemberService.get(memberId),
    enabled: !!memberId,
  });

  useEffect(() => {
    if (member) {
      setAvatarId(member.avatar_id);
    }
  }, [member]);

  const deleteMutation = useMutation({
    mutationFn: () =>
      handleRequest(() => MemberService.delete(memberId), {
        method: "DELETE",
        successMessage: "members.toast.deleted",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.members.lists() });
      navigate("/members", { replace: true });
    },
  });

  if (isLoading) {
    return <Loading />;
  }

  if (isError || !member) {
    return (
      <Container size="md" className="py-6">
        <EmptyState title={t("members.edit.notFound")} />
      </Container>
    );
  }

  return (
    <Container size="xl" className="space-y-4 py-6">
      <h1 className="text-xl font-semibold text-gray-800 dark:text-gray-100">
        {member.name}
      </h1>

      <div className="grid gap-4 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <MemberForm
            initialValues={toMemberFormValues(member)}
            submitLabel={t("members.edit.submit")}
            onSubmit={async (values) => {
              const { data } = await handleRequest(
                () =>
                  MemberService.update(memberId, {
                    name: values.name,
                    tags: values.tags,
                    description: values.description,
                    links_json: JSON.stringify(values.links),
                    // Owned by the list page's drag control; PUT replaces
                    // the whole row, so it has to be carried through.
                    display_order: member.display_order,
                    is_published: values.is_published,
                    avatar_id: avatarId,
                  }),
                {
                  method: "PUT",
                  errorMessages: { 400: "members.errors.nameRequired" },
                  successMessage: "members.toast.saved",
                },
              );
              if (data) {
                queryClient.invalidateQueries({
                  queryKey: queryKeys.members.detail(memberId),
                });
                queryClient.invalidateQueries({
                  queryKey: queryKeys.members.lists(),
                });
              }
            }}
          />
        </div>

        <div className="space-y-4">
          <Card>
            <Card.Header icon={IconPhoto}>
              <h2 className="text-sm font-semibold text-gray-800 dark:text-gray-100">
                {t("members.avatar.title")}
              </h2>
            </Card.Header>
            <Card.Body className="space-y-3">
              <Alert variant="info" description={t("members.avatar.hint")} />
              <ImageUploadField
                label={t("members.avatar.label")}
                hint="400 × 400"
                target="avatar"
                mediaId={avatarId}
                onChange={setAvatarId}
              />
            </Card.Body>
          </Card>

          <Card className="border-red-300/60 dark:border-red-900/60">
            <Card.Header
              icon={IconAlertTriangle}
              className="border-red-200 dark:border-red-900/60"
            >
              <h2 className="text-sm font-semibold text-red-700 dark:text-red-400">
                {t("members.danger.title")}
              </h2>
            </Card.Header>
            <Card.Body className="space-y-3">
              <p className="text-xs text-gray-600 dark:text-gray-400">
                {t("members.danger.description")}
              </p>
              {confirmingDelete ? (
                <div className="space-y-2">
                  <Alert
                    variant="error"
                    description={t("members.delete.description", {
                      name: member.name,
                    })}
                  />
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      className="flex-1"
                      onClick={() => setConfirmingDelete(false)}
                    >
                      {t("common.cancel")}
                    </Button>
                    <Button
                      variant="destructive"
                      size="sm"
                      className="flex-1"
                      disabled={deleteMutation.isPending}
                      onClick={() => deleteMutation.mutate()}
                    >
                      {t("common.confirm")}
                    </Button>
                  </div>
                </div>
              ) : (
                <Button
                  variant="destructive"
                  size="sm"
                  leftIcon={IconTrash}
                  className="w-full"
                  onClick={() => setConfirmingDelete(true)}
                >
                  {t("members.danger.delete")}
                </Button>
              )}
            </Card.Body>
          </Card>
        </div>
      </div>
    </Container>
  );
};

export default MemberEditView;
