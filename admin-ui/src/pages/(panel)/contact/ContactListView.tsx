import {
  IconMail,
  IconMailOpened,
  IconPhoneCall,
  IconTrash,
} from "@tabler/icons-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type FC, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { Badge, Container, DataTable, Dialog } from "@/components/ui";
import { queryKeys } from "@/lib/query/keys";
import { useI18n } from "@/lib/stores/i18n";
import { ContactService } from "@/services/contact";
import { handleRequest } from "@/utilities/request";
import SubmissionDetail from "./SubmissionDetail";

const ContactListView: FC = () => {
  const { t } = useI18n();
  const queryClient = useQueryClient();

  const [viewing, setViewing] = useState<ResponseContactSubmission | null>(
    null,
  );
  const [deleteTarget, setDeleteTarget] =
    useState<ResponseContactSubmission | null>(null);

  // Sorting lives in the URL so an ordering can be linked to and survives a
  // reload rather than resetting on every visit.
  const [searchParams, setSearchParams] = useSearchParams();
  const sortField =
    (searchParams.get("sort") as ContactSortField | null) ?? undefined;
  const sortDirection = searchParams.get("dir") === "desc" ? "desc" : "asc";
  const sort: ContactSort = { field: sortField, direction: sortDirection };

  const setSort = (columnId: string, direction: "asc" | "desc") => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set("sort", columnId);
      next.set("dir", direction);
      return next;
    });
  };

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.contact.list(sort),
    queryFn: () => ContactService.list(sort),
  });

  const submissions = data?.submissions ?? [];
  const unread = data?.unread ?? 0;

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: queryKeys.contact.lists() });

  const readMutation = useMutation({
    mutationFn: ({
      submission,
      isRead,
    }: {
      submission: ResponseContactSubmission;
      isRead: boolean;
    }) =>
      handleRequest(() => ContactService.setRead(submission.id, isRead), {
        method: "PUT",
        successMessage: isRead
          ? "contact.toast.markedRead"
          : "contact.toast.markedUnread",
      }),
    onSuccess: invalidate,
  });

  const deleteMutation = useMutation({
    mutationFn: (submission: ResponseContactSubmission) =>
      handleRequest(() => ContactService.delete(submission.id), {
        method: "DELETE",
        successMessage: "contact.toast.deleted",
      }),
    onSuccess: () => {
      setDeleteTarget(null);
      invalidate();
    },
  });

  const columns: DataTableColumn<ResponseContactSubmission>[] = [
    {
      id: "subject",
      header: t("contact.columns.subject"),
      accessor: "subject",
      sortable: true,
      onClick: (submission) => setViewing(submission),
      // An unread message is what the inbox is for, so it carries weight.
      cell: (value, submission) => (
        <span
          className={
            submission.is_read
              ? "text-gray-700 dark:text-gray-300"
              : "font-semibold text-gray-900 dark:text-gray-50"
          }
        >
          {String(value)}
          {submission.wants_callback && (
            <IconPhoneCall
              size={14}
              className="ml-1.5 inline-block align-text-bottom text-amber-500"
            />
          )}
        </span>
      ),
    },
    {
      id: "from",
      header: t("contact.columns.from"),
      // Either contact field may be missing, so the column shows whichever
      // the sender actually left.
      accessor: (submission) => submission.email || submission.phone,
      cell: (value) => {
        const from = String(value ?? "");
        if (!from) {
          return <span className="text-gray-400 dark:text-gray-600">—</span>;
        }
        return <span className="break-all text-sm">{from}</span>;
      },
    },
    {
      id: "is_read",
      header: t("contact.columns.status"),
      accessor: "is_read",
      sortable: true,
      cell: (value) => (
        <Badge variant={value ? "secondary" : "success"}>
          {value ? t("contact.status.read") : t("contact.status.unread")}
        </Badge>
      ),
    },
    {
      id: "created_at",
      header: t("contact.columns.received"),
      accessor: "created_at",
      type: "date",
      dateOptions: { format: "datetime" },
      sortable: true,
    },
    {
      id: "actions",
      header: "",
      accessor: () => "",
      type: "actions",
      align: "right",
      actions: [
        {
          icon: IconMailOpened,
          label: t("contact.actions.view"),
          onClick: (submission) => setViewing(submission),
        },
        {
          icon: IconMail,
          label: t("contact.actions.markUnread"),
          disabled: (submission: ResponseContactSubmission) =>
            !submission.is_read,
          onClick: (submission) =>
            readMutation.mutate({ submission, isRead: false }),
        },
        {
          icon: IconMailOpened,
          label: t("contact.actions.markRead"),
          disabled: (submission: ResponseContactSubmission) =>
            submission.is_read,
          onClick: (submission) =>
            readMutation.mutate({ submission, isRead: true }),
        },
        {
          icon: IconTrash,
          label: t("common.delete"),
          danger: true,
          variant: "destructive",
          onClick: setDeleteTarget,
        },
      ],
    },
  ];

  return (
    <Container size="xl" className="space-y-4 py-6">
      <div className="flex items-center gap-3">
        <h1 className="text-xl font-semibold text-gray-800 dark:text-gray-100">
          {t("contact.list.title")}
        </h1>
        {unread > 0 && (
          <Badge variant="success">
            {t("contact.list.unread", { count: String(unread) })}
          </Badge>
        )}
      </div>

      <DataTable
        columns={columns}
        data={submissions}
        getRowId={(submission) => submission.id}
        isLoading={isLoading}
        isEmpty={!isLoading && submissions.length === 0}
        error={isError ? t("common.error") : undefined}
        sortBy={sortField}
        sortDir={sortDirection}
        onSortChange={setSort}
        hidePagination
      />

      <SubmissionDetail submission={viewing} onClose={() => setViewing(null)} />

      <Dialog
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        title={t("contact.delete.title")}
        description={t("contact.delete.description", {
          subject: deleteTarget?.subject ?? "",
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

export default ContactListView;
