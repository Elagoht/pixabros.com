import { IconArrowLeft, IconTrash } from "@tabler/icons-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type FC, useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import {
  Alert,
  Button,
  Card,
  Container,
  EmptyState,
  Loading,
} from "@/components/ui";
import GameForm, { toGameFormValues } from "@/forms/GameForm";
import { queryKeys } from "@/lib/query/keys";
import { useI18n } from "@/lib/stores/i18n";
import { GameService } from "@/services/game";
import { handleRequest } from "@/utilities/request";
import BuildUploadCard from "./BuildUploadCard";
import ImageUploadField from "./ImageUploadField";
import ScreenshotManager from "./ScreenshotManager";

const GameEditView: FC = () => {
  const { t } = useI18n();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { id: gameId = "" } = useParams();

  const [confirmingDelete, setConfirmingDelete] = useState(false);

  // Artwork ids are page state, not form fields: uploading an image must not
  // save the rest of a half-edited form, but PUT is a full replace so the
  // ids still have to travel with every save.
  const [cartridgeArtId, setCartridgeArtId] = useState<string | null>(null);
  const [cdCoverArtId, setCdCoverArtId] = useState<string | null>(null);
  const [ogImageId, setOgImageId] = useState<string | null>(null);

  const {
    data: game,
    isLoading,
    isError,
  } = useQuery({
    queryKey: queryKeys.games.detail(gameId),
    queryFn: () => GameService.get(gameId),
    enabled: !!gameId,
  });

  useEffect(() => {
    if (game) {
      setCartridgeArtId(game.cartridge_art_id);
      setCdCoverArtId(game.cd_cover_art_id);
      setOgImageId(game.og_image_id);
    }
  }, [game]);

  const deleteMutation = useMutation({
    mutationFn: () =>
      handleRequest(() => GameService.delete(gameId), {
        method: "DELETE",
        successMessage: "games.toast.deleted",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.games.list() });
      navigate("/games", { replace: true });
    },
  });

  if (isLoading) {
    return <Loading />;
  }

  if (isError || !game) {
    return (
      <Container size="md" className="py-6">
        <EmptyState
          title={t("games.edit.notFound")}
          action={
            <Button variant="outline" onClick={() => navigate("/games")}>
              {t("games.backToList")}
            </Button>
          }
        />
      </Container>
    );
  }

  return (
    <Container size="xl" className="space-y-4 py-6">
      <div className="space-y-2">
        <Button
          variant="ghost"
          size="sm"
          leftIcon={IconArrowLeft}
          onClick={() => navigate("/games")}
        >
          {t("games.backToList")}
        </Button>
        <h1 className="text-xl font-semibold text-gray-800 dark:text-gray-100">
          {game.title}
        </h1>
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <div className="space-y-4 lg:col-span-2">
          <GameForm
            initialValues={toGameFormValues(game)}
            submitLabel={t("games.edit.submit")}
            beforePublishing={<ScreenshotManager gameId={gameId} />}
            slugHint={
              <p className="text-xs text-gray-500 dark:text-gray-400">
                {t("games.edit.slugHint")}{" "}
                <span className="font-mono text-gray-700 dark:text-gray-300">
                  {game.slug}
                </span>
              </p>
            }
            onSubmit={async (values) => {
              const { data } = await handleRequest(
                () =>
                  GameService.update(gameId, {
                    ...values,
                    external_links_json: JSON.stringify(values.external_links),
                    // Carried through untouched: the list page's drag
                    // control owns this, and PUT replaces the whole row.
                    display_order: game.display_order,
                    cartridge_art_id: cartridgeArtId,
                    cd_cover_art_id: cdCoverArtId,
                    og_image_id: ogImageId,
                  }),
                {
                  method: "PUT",
                  errorMessages: { 400: "games.errors.titleRequired" },
                  successMessage: "games.toast.saved",
                },
              );
              if (data) {
                queryClient.invalidateQueries({
                  queryKey: queryKeys.games.detail(gameId),
                });
                queryClient.invalidateQueries({
                  queryKey: queryKeys.games.list(),
                });
              }
            }}
          />
        </div>

        <div className="space-y-4">
          <Card>
            <Card.Header>
              <h2 className="text-sm font-semibold text-gray-800 dark:text-gray-100">
                {t("games.artwork.title")}
              </h2>
            </Card.Header>
            <Card.Body className="space-y-3">
              <Alert variant="info" description={t("games.artwork.hint")} />
              <ImageUploadField
                label={t("games.artwork.cartridge")}
                hint="400 × 560"
                target="cartridge_art"
                mediaId={cartridgeArtId}
                onChange={setCartridgeArtId}
              />
              <ImageUploadField
                label={t("games.artwork.cdCover")}
                hint="600 × 600"
                target="cd_cover_art"
                mediaId={cdCoverArtId}
                onChange={setCdCoverArtId}
              />
              <ImageUploadField
                label={t("games.artwork.ogImage")}
                hint="1200 × 630"
                target="og_image"
                mediaId={ogImageId}
                onChange={setOgImageId}
              />
            </Card.Body>
          </Card>

          <BuildUploadCard
            slug={game.slug}
            webExportPath={game.web_export_path}
            onUploaded={() =>
              queryClient.invalidateQueries({
                queryKey: queryKeys.games.detail(gameId),
              })
            }
          />

          <Card className="border-red-300/60 dark:border-red-900/60">
            <Card.Header className="border-red-200 dark:border-red-900/60">
              <h2 className="text-sm font-semibold text-red-700 dark:text-red-400">
                {t("games.danger.title")}
              </h2>
            </Card.Header>
            <Card.Body className="space-y-3">
              <p className="text-xs text-gray-600 dark:text-gray-400">
                {t("games.danger.description")}
              </p>
              {confirmingDelete ? (
                <div className="space-y-2">
                  <Alert
                    variant="error"
                    description={t("games.delete.description", {
                      title: game.title,
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
                  {t("games.danger.delete")}
                </Button>
              )}
            </Card.Body>
          </Card>
        </div>
      </div>
    </Container>
  );
};

export default GameEditView;
