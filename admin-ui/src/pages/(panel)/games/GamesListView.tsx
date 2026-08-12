import { IconArrowsSort, IconPlus } from "@tabler/icons-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type FC, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Button, Container, Dialog } from "@/components/ui";
import { queryKeys } from "@/lib/query/keys";
import { useI18n } from "@/lib/stores/i18n";
import { GameService } from "@/services/game";
import { handleRequest } from "@/utilities/request";
import GamesTable from "./GamesTable";
import ReorderGamesModal from "./ReorderGamesModal";

const GamesListView: FC = () => {
  const { t } = useI18n();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [deleteTarget, setDeleteTarget] = useState<ResponseGame | null>(null);
  const [reorderOpen, setReorderOpen] = useState(false);

  const {
    data: games = [],
    isLoading,
    isError,
  } = useQuery({
    queryKey: queryKeys.games.list(),
    queryFn: GameService.list,
  });

  const invalidateList = () =>
    queryClient.invalidateQueries({ queryKey: queryKeys.games.list() });

  const deleteMutation = useMutation({
    mutationFn: (game: ResponseGame) =>
      handleRequest(() => GameService.delete(game.id), {
        method: "DELETE",
        successMessage: "games.toast.deleted",
      }),
    onSuccess: () => {
      setDeleteTarget(null);
      invalidateList();
    },
  });

  const reorderMutation = useMutation({
    mutationFn: (ids: string[]) =>
      handleRequest(() => GameService.reorder(ids), {
        method: "PUT",
        successMessage: "games.toast.reordered",
      }),
    onSuccess: () => {
      setReorderOpen(false);
      invalidateList();
    },
  });

  return (
    <Container size="xl" className="space-y-4 py-6">
      <div className="flex items-center justify-between gap-3">
        <h1 className="text-xl font-semibold text-gray-800 dark:text-gray-100">
          {t("games.list.title")}
        </h1>
      </div>

      <GamesTable
        games={games}
        isLoading={isLoading}
        error={isError ? t("common.error") : undefined}
        onDelete={setDeleteTarget}
        toolbarActions={
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              leftIcon={IconArrowsSort}
              disabled={games.length < 2}
              onClick={() => setReorderOpen(true)}
            >
              {t("games.list.reorder")}
            </Button>
            <Button
              variant="default"
              size="sm"
              leftIcon={IconPlus}
              onClick={() => navigate("/games/new")}
            >
              {t("games.list.new")}
            </Button>
          </div>
        }
      />

      <ReorderGamesModal
        open={reorderOpen}
        games={games}
        isSaving={reorderMutation.isPending}
        onClose={() => setReorderOpen(false)}
        onSave={(ids) => reorderMutation.mutate(ids)}
      />

      <Dialog
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        title={t("games.delete.title")}
        description={t("games.delete.description", {
          title: deleteTarget?.title ?? "",
        })}
        confirmLabel={t("common.delete")}
        confirmVariant="destructive"
        onConfirm={() => {
          if (deleteTarget) {
            deleteMutation.mutate(deleteTarget);
          }
        }}
        onCancel={() => setDeleteTarget(null)}
      />
    </Container>
  );
};

export default GamesListView;
