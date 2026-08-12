import {
  IconMail,
  IconMailOpened,
  IconPhoneCall,
  IconTrash,
} from "@tabler/icons-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import classNames from "classnames";
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
      silent = false,
    }: {
      submission: ResponseContactSubmission;
      isRead: boolean;
      silent?: boolean;
    }) =>
      handleRequest(() => ContactService.setRead(submission.id, isRead), {
        method: "PUT",
        showSuccessMessage: !silent,
        successMessage: isRead
          ? "contact.toast.markedRead"
          : "contact.toast.markedUnread",
      }),
    onSuccess: invalidate,
  });

  // Opening a message is reading it. These used to be two separate actions,
  // which meant the unread badge stayed on until you remembered to clear it by
  // hand.
  const openSubmission = (submission: ResponseContactSubmission) => {
    setViewing(submission);
    if (!submission.is_read) {
      readMutation.mutate({ submission, isRead: true, silent: true });
    }
  };

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
      onClick: openSubmission,
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
      // Any of the three may be missing, so the column shows whichever the
      // sender actually left, name first.
      accessor: (submission) =>
        submission.name || submission.email || submission.phone,
      cell: (_value, submission) => {
        const contact = submission.email || submission.phone;
        if (!(submission.name || contact)) {
          return <span className="text-gray-400 dark:text-gray-600">—</span>;
        }
        return (
          <div className="min-w-0">
            {submission.name && (
              <div className="truncate text-sm font-medium text-gray-900 dark:text-gray-100">
                {submission.name}
              </div>
            )}
            {contact && (
              <div
                className={classNames(
                  "break-all",
                  submission.name
                    ? "text-xs text-gray-500 dark:text-gray-400"
                    : "text-sm",
                )}
              >
                {contact}
              </div>
            )}
          </div>
        );
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
          label: t("contact.actions.read"),
          onClick: openSubmission,
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
