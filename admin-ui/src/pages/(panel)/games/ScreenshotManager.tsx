import { IconPhotoPlus, IconTrash } from "@tabler/icons-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type ChangeEvent, type FC, useRef, useState } from "react";
import { Button, Card, EmptyState, Loading, Skeleton } from "@/components/ui";
import { queryKeys } from "@/lib/query/keys";
import { useI18n } from "@/lib/stores/i18n";
import { GameService } from "@/services/game";
import { MediaService } from "@/services/media";
import { handleRequest } from "@/utilities/request";

interface ScreenshotThumbProps {
  mediaId: string;
  alt: string;
}

const ScreenshotThumb: FC<ScreenshotThumbProps> = ({ mediaId, alt }) => {
  const { data, isLoading } = useQuery({
    queryKey: queryKeys.media.detail(mediaId),
    queryFn: () => MediaService.get(mediaId),
  });

  if (isLoading) {
    return <Skeleton className="h-20 w-full" variant="rect" />;
  }
  if (!data) {
    return null;
  }
  return (
    <img
      src={data.url}
      alt={alt}
      className="h-20 w-full rounded-md object-cover"
    />
  );
};

interface ScreenshotManagerProps {
  gameId: string;
}

const ScreenshotManager: FC<ScreenshotManagerProps> = ({ gameId }) => {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const inputRef = useRef<HTMLInputElement>(null);
  const [isAdding, setIsAdding] = useState(false);

  const { data: screenshots = [], isLoading } = useQuery({
    queryKey: queryKeys.games.screenshots(gameId),
    queryFn: () => GameService.listScreenshots(gameId),
  });

  const invalidate = () =>
    queryClient.invalidateQueries({
      queryKey: queryKeys.games.screenshots(gameId),
    });

  const removeMutation = useMutation({
    mutationFn: (screenshotId: string) =>
      handleRequest(() => GameService.removeScreenshot(gameId, screenshotId), {
        method: "DELETE",
        successMessage: "games.toast.screenshotRemoved",
      }),
    onSuccess: invalidate,
  });

  // Adding is two calls: upload the bytes, then attach the returned media id
  // to this game at the end of the list.
  const handleFile = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) {
      return;
    }
    setIsAdding(true);
    const { data: uploaded } = await handleRequest(
      () => MediaService.upload(file, "screenshot"),
      {
        method: "POST",
        errorMessages: {
          400: "games.errors.invalidImage",
          413: "games.errors.fileTooLarge",
        },
        showSuccessMessage: false,
      },
    );

    if (uploaded) {
      await handleRequest(
        () =>
          GameService.addScreenshot(gameId, {
            media_id: uploaded.id,
            display_order: screenshots.length,
          }),
        { method: "POST", successMessage: "games.toast.screenshotAdded" },
      );
      invalidate();
    }

    setIsAdding(false);
    if (inputRef.current) {
      inputRef.current.value = "";
    }
  };

  return (
    <Card>
      <Card.Header className="justify-between">
        <h2 className="text-sm font-semibold text-gray-800 dark:text-gray-100">
          {t("games.screenshots.title")}
        </h2>
        <Button
          variant="outline"
          size="sm"
          leftIcon={IconPhotoPlus}
          disabled={isAdding}
          onClick={() => inputRef.current?.click()}
        >
          {isAdding ? t("common.uploading") : t("games.screenshots.add")}
        </Button>
      </Card.Header>

      <Card.Body>
        {isLoading ? (
          <Loading />
        ) : screenshots.length === 0 ? (
          <EmptyState title={t("games.screenshots.empty")} />
        ) : (
          <ul className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {screenshots.map((screenshot, index) => (
              <li
                key={screenshot.id}
                className="space-y-1.5 rounded-lg border border-gray-200 p-2 dark:border-gray-700"
              >
                <ScreenshotThumb
                  mediaId={screenshot.media_id}
                  alt={t("games.screenshots.alt", {
                    index: String(index + 1),
                  })}
                />
                <div className="flex items-center justify-between">
                  <span className="text-[10px] text-gray-400 dark:text-gray-500">
                    #{screenshot.display_order + 1}
                  </span>
                  <Button
                    variant="ghost"
                    size="sm"
                    title={t("common.delete")}
                    disabled={removeMutation.isPending}
                    onClick={() => removeMutation.mutate(screenshot.id)}
                  >
                    <IconTrash size={14} />
                  </Button>
                </div>
              </li>
            ))}
          </ul>
        )}

        <input
          ref={inputRef}
          type="file"
          accept="image/*"
          className="hidden"
          onChange={handleFile}
        />
      </Card.Body>
    </Card>
  );
};

export default ScreenshotManager;
