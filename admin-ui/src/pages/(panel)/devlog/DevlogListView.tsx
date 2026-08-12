import { IconPlus } from "@tabler/icons-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type FC, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { Button, Container, Dialog } from "@/components/ui";
import { queryKeys } from "@/lib/query/keys";
import { useI18n } from "@/lib/stores/i18n";
import { DevlogService } from "@/services/devlog";
import { GameService } from "@/services/game";
import { handleRequest } from "@/utilities/request";
import DevlogTable from "./DevlogTable";

const DevlogListView: FC = () => {
  const { t } = useI18n();
  const queryClient = useQueryClient();

  const [deleteTarget, setDeleteTarget] = useState<ResponseDevlogPost | null>(
    null,
  );

  // Sorting lives in the URL so an ordering can be linked to and survives a
  // reload rather than resetting on every visit.
  const [searchParams, setSearchParams] = useSearchParams();
  const sortField =
    (searchParams.get("sort") as DevlogSortField | null) ?? undefined;
  const sortDirection = searchParams.get("dir") === "desc" ? "desc" : "asc";
  const sort: DevlogSort = { field: sortField, direction: sortDirection };

  const setSort = (columnId: string, direction: "asc" | "desc") => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set("sort", columnId);
      next.set("dir", direction);
      return next;
    });
  };

  const {
    data: posts = [],
    isLoading,
    isError,
  } = useQuery({
    queryKey: queryKeys.devlog.list(sort),
    queryFn: () => DevlogService.list(sort),
  });

  // The list shows which game a post belongs to, and the API stores only the
  // id, so the titles are resolved here.
  const { data: games = [] } = useQuery({
    queryKey: queryKeys.games.list(),
    queryFn: () => GameService.list(),
  });

  const deleteMutation = useMutation({
    mutationFn: (post: ResponseDevlogPost) =>
      handleRequest(() => DevlogService.delete(post.id), {
        method: "DELETE",
        successMessage: "devlog.toast.deleted",
      }),
    onSuccess: () => {
      setDeleteTarget(null);
      queryClient.invalidateQueries({ queryKey: queryKeys.devlog.lists() });
    },
  });

  return (
    <Container size="xl" className="space-y-4 py-6">
      <h1 className="text-xl font-semibold text-gray-800 dark:text-gray-100">
        {t("devlog.list.title")}
      </h1>

      <DevlogTable
        posts={posts}
        games={games}
        isLoading={isLoading}
        error={isError ? t("common.error") : undefined}
        onDelete={setDeleteTarget}
        sortBy={sortField}
        sortDir={sortDirection}
        onSortChange={setSort}
        toolbarActions={
          <Button
            variant="default"
            size="sm"
            leftIcon={IconPlus}
            to="/devlog/new"
          >
            {t("devlog.list.new")}
          </Button>
        }
      />

      <Dialog
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        title={t("devlog.delete.title")}
        description={t("devlog.delete.description", {
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

export default DevlogListView;
