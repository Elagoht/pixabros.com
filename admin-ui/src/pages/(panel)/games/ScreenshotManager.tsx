import {
  closestCenter,
  DndContext,
  type DragEndEvent,
  PointerSensor,
  useSensor,
  useSensors,
} from "@dnd-kit/core";
import { IconPhotoPlus } from "@tabler/icons-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type ChangeEvent, type FC, useEffect, useRef, useState } from "react";
import { Button, Card, EmptyState, Loading } from "@/components/ui";
import { queryKeys } from "@/lib/query/keys";
import { useI18n } from "@/lib/stores/i18n";
import { GameService } from "@/services/game";
import { MediaService } from "@/services/media";
import { handleRequest } from "@/utilities/request";
import SortableScreenshot from "./SortableScreenshot";

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

  // Dragging reorders this local copy immediately so the tiles move under the
  // pointer; the server is told afterwards. Re-syncing from the query means a
  // rejected reorder snaps back to the real order instead of lying.
  const [ordered, setOrdered] = useState<ResponseScreenshot[]>(screenshots);
  useEffect(() => {
    setOrdered(screenshots);
  }, [screenshots]);

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

  const reorderMutation = useMutation({
    mutationFn: (ids: string[]) =>
      handleRequest(() => GameService.reorderScreenshots(gameId, ids), {
        method: "PUT",
        showSuccessMessage: false,
      }),
    onSettled: invalidate,
  });

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
  );

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) {
      return;
    }
    const from = ordered.findIndex((shot) => shot.id === active.id);
    const to = ordered.findIndex((shot) => shot.id === over.id);
    if (from === -1 || to === -1) {
      return;
    }

    const next = [...ordered];
    const [moved] = next.splice(from, 1);
    next.splice(to, 0, moved);
    setOrdered(next);
    reorderMutation.mutate(next.map((shot) => shot.id));
  };

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
            display_order: ordered.length,
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

  const isBusy = removeMutation.isPending || reorderMutation.isPending;

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

      <Card.Body className="space-y-3">
        {isLoading ? (
          <Loading />
        ) : ordered.length === 0 ? (
          <EmptyState title={t("games.screenshots.empty")} />
        ) : (
          <>
            {ordered.length > 1 && (
              <p className="text-xs text-gray-500 dark:text-gray-400">
                {t("games.screenshots.reorderHint")}
              </p>
            )}
            <DndContext
              sensors={sensors}
              collisionDetection={closestCenter}
              onDragEnd={handleDragEnd}
            >
              <ul className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                {ordered.map((screenshot, index) => (
                  <SortableScreenshot
                    key={screenshot.id}
                    id={screenshot.id}
                    mediaId={screenshot.media_id}
                    position={index + 1}
                    alt={t("games.screenshots.alt", {
                      index: String(index + 1),
                    })}
                    removeLabel={t("common.delete")}
                    isBusy={isBusy}
                    onRemove={() => removeMutation.mutate(screenshot.id)}
                  />
                ))}
              </ul>
            </DndContext>
          </>
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
