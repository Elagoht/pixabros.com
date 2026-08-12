import { IconPencil, IconTrash } from "@tabler/icons-react";
import type { FC, ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { Badge, DataTable } from "@/components/ui";
import { useI18n } from "@/lib/stores/i18n";

interface MembersTableProps {
  members: ResponseMember[];
  isLoading: boolean;
  error?: string;
  onDelete: (member: ResponseMember) => void;
  toolbarActions?: ReactNode;
  sortBy?: string;
  sortDir?: "asc" | "desc";
  onSortChange?: (columnId: string, direction: "asc" | "desc") => void;
}

const MembersTable: FC<MembersTableProps> = ({
  members,
  isLoading,
  error,
  onDelete,
  toolbarActions,
  sortBy,
  sortDir,
  onSortChange,
}) => {
  const { t } = useI18n();
  const navigate = useNavigate();

  const columns: DataTableColumn<ResponseMember>[] = [
    {
      id: "name",
      header: t("members.columns.name"),
      accessor: "name",
      sortable: true,
      onClick: (member) => navigate(`/members/${member.id}`),
    },
    {
      id: "tags",
      header: t("members.columns.tags"),
      accessor: "tags",
      cell: (value) => {
        const tags = String(value ?? "")
          .split(",")
          .map((tag) => tag.trim())
          .filter(Boolean);
        if (tags.length === 0) {
          return <span className="text-gray-400 dark:text-gray-600">—</span>;
        }
        return (
          <span className="flex flex-wrap gap-1">
            {tags.map((tag) => (
              <Badge key={tag} variant="outline">
                {tag}
              </Badge>
            ))}
          </span>
        );
      },
    },
    {
      id: "is_published",
      header: t("members.columns.status"),
      accessor: "is_published",
      sortable: true,
      cell: (value) => (
        <Badge variant={value ? "success" : "secondary"}>
          {value ? t("members.status.published") : t("members.status.draft")}
        </Badge>
      ),
    },
    {
      id: "display_order",
      header: t("members.columns.order"),
      accessor: "display_order",
      type: "number",
      sortable: true,
      align: "right",
    },
    {
      id: "actions",
      header: "",
      accessor: () => "",
      type: "actions",
      align: "right",
      actions: [
        {
          icon: IconPencil,
          label: t("common.edit"),
          onClick: (member) => navigate(`/members/${member.id}`),
        },
        {
          icon: IconTrash,
          label: t("common.delete"),
          danger: true,
          variant: "destructive",
          onClick: onDelete,
        },
      ],
    },
  ];

  return (
    <DataTable
      columns={columns}
      data={members}
      getRowId={(member) => member.id}
      isLoading={isLoading}
      isEmpty={!isLoading && members.length === 0}
      error={error}
      toolbarActions={toolbarActions}
      sortBy={sortBy}
      sortDir={sortDir}
      onSortChange={onSortChange}
      hidePagination
    />
  );
};

export default MembersTable;
