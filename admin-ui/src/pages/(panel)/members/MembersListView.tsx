import { IconArrowsSort, IconPlus } from "@tabler/icons-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type FC, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { Button, Container, Dialog, ReorderModal } from "@/components/ui";
import { queryKeys } from "@/lib/query/keys";
import { useI18n } from "@/lib/stores/i18n";
import { MemberService } from "@/services/member";
import { handleRequest } from "@/utilities/request";
import MembersTable from "./MembersTable";

const MembersListView: FC = () => {
  const { t } = useI18n();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [deleteTarget, setDeleteTarget] = useState<ResponseMember | null>(null);
  const [reorderOpen, setReorderOpen] = useState(false);

  // Sorting lives in the URL so an ordering can be linked to and survives a
  // reload rather than resetting on every visit.
  const [searchParams, setSearchParams] = useSearchParams();
  const sortField =
    (searchParams.get("sort") as MemberSortField | null) ?? undefined;
  const sortDirection = searchParams.get("dir") === "desc" ? "desc" : "asc";
  const sort: MemberSort = { field: sortField, direction: sortDirection };

  const setSort = (columnId: string, direction: "asc" | "desc") => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set("sort", columnId);
      next.set("dir", direction);
      return next;
    });
  };

  const {
    data: members = [],
    isLoading,
    isError,
  } = useQuery({
    queryKey: queryKeys.members.list(sort),
    queryFn: () => MemberService.list(sort),
  });

  // The reorder modal edits display_order, so it must always show the manual
  // order rather than whatever column the table is sorted by. Mirrors the
  // server's "display_order ASC, id ASC".
  const manualOrder = [...members].sort(
    (a, b) =>
      a.display_order - b.display_order ||
      (a.id < b.id ? -1 : a.id > b.id ? 1 : 0),
  );

  const invalidateList = () =>
    queryClient.invalidateQueries({ queryKey: queryKeys.members.lists() });

  const deleteMutation = useMutation({
    mutationFn: (member: ResponseMember) =>
      handleRequest(() => MemberService.delete(member.id), {
        method: "DELETE",
        successMessage: "members.toast.deleted",
      }),
    onSuccess: () => {
      setDeleteTarget(null);
      invalidateList();
    },
  });

  const reorderMutation = useMutation({
    mutationFn: (ids: string[]) =>
      handleRequest(() => MemberService.reorder(ids), {
        method: "PUT",
        successMessage: "members.toast.reordered",
      }),
    onSuccess: () => {
      setReorderOpen(false);
      invalidateList();
    },
  });

  return (
    <Container size="xl" className="space-y-4 py-6">
      <h1 className="text-xl font-semibold text-gray-800 dark:text-gray-100">
        {t("members.list.title")}
      </h1>

      <MembersTable
        members={members}
        isLoading={isLoading}
        error={isError ? t("common.error") : undefined}
        onDelete={setDeleteTarget}
        sortBy={sortField}
        sortDir={sortDirection}
        onSortChange={setSort}
        toolbarActions={
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              leftIcon={IconArrowsSort}
              disabled={members.length < 2}
              onClick={() => setReorderOpen(true)}
            >
              {t("members.list.reorder")}
            </Button>
            <Button
              variant="default"
              size="sm"
              leftIcon={IconPlus}
              onClick={() => navigate("/members/new")}
            >
              {t("members.list.new")}
            </Button>
          </div>
        }
      />

      <ReorderModal
        open={reorderOpen}
        items={manualOrder.map((member) => ({
          id: member.id,
          label: member.name,
        }))}
        title={t("members.reorder.title")}
        help={t("members.reorder.help")}
        isSaving={reorderMutation.isPending}
        onClose={() => setReorderOpen(false)}
        onSave={(ids) => reorderMutation.mutate(ids)}
      />

      <Dialog
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        title={t("members.delete.title")}
        description={t("members.delete.description", {
          name: deleteTarget?.name ?? "",
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

export default MembersListView;
