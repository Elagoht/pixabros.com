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
import DevlogForm, { toDevlogFormValues } from "@/forms/DevlogForm";
import { queryKeys } from "@/lib/query/keys";
import { useI18n } from "@/lib/stores/i18n";
import { DevlogService } from "@/services/devlog";
import { handleRequest } from "@/utilities/request";

const DevlogEditView: FC = () => {
  const { t } = useI18n();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { id: postId = "" } = useParams();

  const [confirmingDelete, setConfirmingDelete] = useState(false);
  // The image id is page state, not a form field: uploading must not save a
  // half-edited form, but PUT is a full replace so it travels with a save.
  const [ogImageId, setOgImageId] = useState<string | null>(null);

  const {
    data: post,
    isLoading,
    isError,
  } = useQuery({
    queryKey: queryKeys.devlog.detail(postId),
    queryFn: () => DevlogService.get(postId),
    enabled: !!postId,
  });

  useEffect(() => {
    if (post) {
      setOgImageId(post.og_image_id);
    }
  }, [post]);

  const deleteMutation = useMutation({
    mutationFn: () =>
      handleRequest(() => DevlogService.delete(postId), {
        method: "DELETE",
        successMessage: "devlog.toast.deleted",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.devlog.lists() });
      navigate("/devlog", { replace: true });
    },
  });

  if (isLoading) {
    return <Loading />;
  }

  if (isError || !post) {
    return (
      <Container size="md" className="py-6">
        <EmptyState title={t("devlog.edit.notFound")} />
      </Container>
    );
  }

  return (
    <Container size="xl" className="space-y-4 py-6">
      <h1 className="text-xl font-semibold text-gray-800 dark:text-gray-100">
        {post.title}
      </h1>

      <div className="grid gap-4 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <DevlogForm
            initialValues={toDevlogFormValues(post)}
            submitLabel={t("devlog.edit.submit")}
            slugHint={
              <p className="text-xs text-gray-500 dark:text-gray-400">
                {t("devlog.form.slug")}{" "}
                <span className="font-mono text-gray-700 dark:text-gray-300">
                  {post.slug}
                </span>{" "}
                {t("devlog.form.slugFollowsTitle")}
              </p>
            }
            onSubmit={async (values) => {
              const { data } = await handleRequest(
                () =>
                  DevlogService.update(postId, {
                    title: values.title,
                    content_markdown: values.content_markdown,
                    is_published: values.is_published,
                    published_at: values.published_at,
                    // The picker uses "" for "no game"; the column is nullable.
                    game_id: values.game_id || null,
                    og_image_id: ogImageId,
                  }),
                {
                  method: "PUT",
                  errorMessages: { 400: "devlog.errors.invalidFields" },
                  successMessage: "devlog.toast.saved",
                },
              );
              if (data) {
                queryClient.invalidateQueries({
                  queryKey: queryKeys.devlog.detail(postId),
                });
                queryClient.invalidateQueries({
                  queryKey: queryKeys.devlog.lists(),
                });
              }
            }}
          />
        </div>

        <div className="space-y-4">
          <Card>
            <Card.Header icon={IconPhoto}>
              <h2 className="text-sm font-semibold text-gray-800 dark:text-gray-100">
                {t("devlog.image.title")}
              </h2>
            </Card.Header>
            <Card.Body className="space-y-3">
              <Alert variant="info" description={t("devlog.image.hint")} />
              <ImageUploadField
                label={t("devlog.image.label")}
                hint="1200 × 630"
                target="og_image"
                mediaId={ogImageId}
                onChange={setOgImageId}
              />
            </Card.Body>
          </Card>

          <Card className="border-red-300/60 dark:border-red-900/60">
            <Card.Header
              icon={IconAlertTriangle}
              className="border-red-200 dark:border-red-900/60"
            >
              <h2 className="text-sm font-semibold text-red-700 dark:text-red-400">
                {t("devlog.danger.title")}
              </h2>
            </Card.Header>
            <Card.Body className="space-y-3">
              <p className="text-xs text-gray-600 dark:text-gray-400">
                {t("devlog.danger.description")}
              </p>
              {confirmingDelete ? (
                <div className="space-y-2">
                  <Alert
                    variant="error"
                    description={t("devlog.delete.description", {
                      title: post.title,
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
                  {t("devlog.danger.delete")}
                </Button>
              )}
            </Card.Body>
          </Card>
        </div>
      </div>
    </Container>
  );
};

export default DevlogEditView;
