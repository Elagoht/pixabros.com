import { IconPlus } from "@tabler/icons-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type FC, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { Button, Container, Dialog } from "@/components/ui";
import { queryKeys } from "@/lib/query/keys";
import { useI18n } from "@/lib/stores/i18n";
import { AwardService } from "@/services/award";
import { GameService } from "@/services/game";
import { handleRequest } from "@/utilities/request";
import AwardsTable from "./AwardsTable";

const AwardsListView: FC = () => {
  const { t } = useI18n();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [deleteTarget, setDeleteTarget] = useState<ResponseAward | null>(null);

  // Sorting lives in the URL so an ordering can be linked to and survives a
  // reload rather than resetting on every visit.
  const [searchParams, setSearchParams] = useSearchParams();
  const sortField =
    (searchParams.get("sort") as AwardSortField | null) ?? undefined;
  const sortDirection = searchParams.get("dir") === "desc" ? "desc" : "asc";
  const sort: AwardSort = { field: sortField, direction: sortDirection };

  const setSort = (columnId: string, direction: "asc" | "desc") => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set("sort", columnId);
      next.set("dir", direction);
      return next;
    });
  };

  const {
    data: awards = [],
    isLoading,
    isError,
  } = useQuery({
    queryKey: queryKeys.awards.list(sort),
    queryFn: () => AwardService.list(sort),
  });

  // The list shows which game an award belongs to, and the API stores only
  // the id, so the titles are resolved here.
  const { data: games = [] } = useQuery({
    queryKey: queryKeys.games.list(),
    queryFn: () => GameService.list(),
  });

  const deleteMutation = useMutation({
    mutationFn: (award: ResponseAward) =>
      handleRequest(() => AwardService.delete(award.id), {
        method: "DELETE",
        successMessage: "awards.toast.deleted",
      }),
    onSuccess: () => {
      setDeleteTarget(null);
      queryClient.invalidateQueries({ queryKey: queryKeys.awards.lists() });
    },
  });

  return (
    <Container size="xl" className="space-y-4 py-6">
      <h1 className="text-xl font-semibold text-gray-800 dark:text-gray-100">
        {t("awards.list.title")}
      </h1>

      <AwardsTable
        awards={awards}
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
            onClick={() => navigate("/awards/new")}
          >
            {t("awards.list.new")}
          </Button>
        }
      />

      <Dialog
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        title={t("awards.delete.title")}
        description={t("awards.delete.description", {
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

export default AwardsListView;
