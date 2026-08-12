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
import AwardForm, { toAwardFormValues } from "@/forms/AwardForm";
import { queryKeys } from "@/lib/query/keys";
import { useI18n } from "@/lib/stores/i18n";
import { AwardService } from "@/services/award";
import { handleRequest } from "@/utilities/request";

const AwardEditView: FC = () => {
  const { t } = useI18n();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { id: awardId = "" } = useParams();

  const [confirmingDelete, setConfirmingDelete] = useState(false);
  // The picture id is page state, not a form field: uploading must not save
  // a half-edited form, but PUT is a full replace so it travels with a save.
  const [pictureId, setPictureId] = useState<string | null>(null);

  const {
    data: award,
    isLoading,
    isError,
  } = useQuery({
    queryKey: queryKeys.awards.detail(awardId),
    queryFn: () => AwardService.get(awardId),
    enabled: !!awardId,
  });

  useEffect(() => {
    if (award) {
      setPictureId(award.picture_id);
    }
  }, [award]);

  const deleteMutation = useMutation({
    mutationFn: () =>
      handleRequest(() => AwardService.delete(awardId), {
        method: "DELETE",
        successMessage: "awards.toast.deleted",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.awards.lists() });
      navigate("/awards", { replace: true });
    },
  });

  if (isLoading) {
    return <Loading />;
  }

  if (isError || !award) {
    return (
      <Container size="md" className="py-6">
        <EmptyState title={t("awards.edit.notFound")} />
      </Container>
    );
  }

  return (
    <Container size="xl" className="space-y-4 py-6">
      <h1 className="text-xl font-semibold text-gray-800 dark:text-gray-100">
        {award.title}
      </h1>

      <div className="grid gap-4 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <AwardForm
            initialValues={toAwardFormValues(award)}
            submitLabel={t("awards.edit.submit")}
            onSubmit={async (values) => {
              const { data } = await handleRequest(
                () =>
                  AwardService.update(awardId, {
                    title: values.title,
                    issuer: values.issuer,
                    date: values.date,
                    link: values.link,
                    // The picker uses "" for "no game"; the column is nullable.
                    game_id: values.game_id || null,
                    picture_id: pictureId,
                  }),
                {
                  method: "PUT",
                  errorMessages: { 400: "awards.errors.invalidFields" },
                  successMessage: "awards.toast.saved",
                },
              );
              if (data) {
                queryClient.invalidateQueries({
                  queryKey: queryKeys.awards.detail(awardId),
                });
                queryClient.invalidateQueries({
                  queryKey: queryKeys.awards.lists(),
                });
              }
            }}
          />
        </div>

        <div className="space-y-4">
          <Card>
            <Card.Header icon={IconPhoto}>
              <h2 className="text-sm font-semibold text-gray-800 dark:text-gray-100">
                {t("awards.picture.title")}
              </h2>
            </Card.Header>
            <Card.Body className="space-y-3">
              <Alert variant="info" description={t("awards.picture.hint")} />
              <ImageUploadField
                label={t("awards.picture.label")}
                hint="320 × 320"
                target="award_picture"
                mediaId={pictureId}
                onChange={setPictureId}
              />
            </Card.Body>
          </Card>

          <Card className="border-red-300/60 dark:border-red-900/60">
            <Card.Header
              icon={IconAlertTriangle}
              className="border-red-200 dark:border-red-900/60"
            >
              <h2 className="text-sm font-semibold text-red-700 dark:text-red-400">
                {t("awards.danger.title")}
              </h2>
            </Card.Header>
            <Card.Body className="space-y-3">
              <p className="text-xs text-gray-600 dark:text-gray-400">
                {t("awards.danger.description")}
              </p>
              {confirmingDelete ? (
                <div className="space-y-2">
                  <Alert
                    variant="error"
                    description={t("awards.delete.description", {
                      title: award.title,
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
                  {t("awards.danger.delete")}
                </Button>
              )}
            </Card.Body>
          </Card>
        </div>
      </div>
    </Container>
  );
};

export default AwardEditView;
