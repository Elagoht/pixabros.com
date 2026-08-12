import { IconPencil, IconTrash } from "@tabler/icons-react";
import type { FC, ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { Badge, DataTable } from "@/components/ui";
import { useI18n } from "@/lib/stores/i18n";

interface DevlogTableProps {
  posts: ResponseDevlogPost[];
  games: ResponseGame[];
  isLoading: boolean;
  error?: string;
  onDelete: (post: ResponseDevlogPost) => void;
  toolbarActions?: ReactNode;
  sortBy?: string;
  sortDir?: "asc" | "desc";
  onSortChange?: (columnId: string, direction: "asc" | "desc") => void;
}

const DevlogTable: FC<DevlogTableProps> = ({
  posts,
  games,
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

  const gameTitles = new Map(games.map((game) => [game.id, game.title]));

  const columns: DataTableColumn<ResponseDevlogPost>[] = [
    {
      id: "title",
      header: t("devlog.columns.title"),
      accessor: "title",
      sortable: true,
      onClick: (post) => navigate(`/devlog/${post.id}`),
    },
    {
      id: "game",
      header: t("devlog.columns.game"),
      accessor: (post) =>
        post.game_id ? (gameTitles.get(post.game_id) ?? "") : "",
      cell: (_value, post) => {
        const title = post.game_id ? gameTitles.get(post.game_id) : undefined;
        if (!title) {
          return <span className="text-gray-400 dark:text-gray-600">—</span>;
        }
        return <Badge variant="outline">{title}</Badge>;
      },
    },
    {
      id: "is_published",
      header: t("devlog.columns.status"),
      accessor: "is_published",
      sortable: true,
      cell: (value) => (
        <Badge variant={value ? "success" : "secondary"}>
          {value ? t("devlog.status.published") : t("devlog.status.draft")}
        </Badge>
      ),
    },
    {
      id: "published_at",
      header: t("devlog.columns.publishedAt"),
      accessor: "published_at",
      sortable: true,
      cell: (value) => {
        const date = String(value ?? "");
        if (!date) {
          return <span className="text-gray-400 dark:text-gray-600">—</span>;
        }
        return <span className="tabular-nums">{date}</span>;
      },
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
          onClick: (post) => navigate(`/devlog/${post.id}`),
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
      data={posts}
      getRowId={(post) => post.id}
      isLoading={isLoading}
      isEmpty={!isLoading && posts.length === 0}
      error={error}
      toolbarActions={toolbarActions}
      sortBy={sortBy}
      sortDir={sortDir}
      onSortChange={onSortChange}
      hidePagination
    />
  );
};

export default DevlogTable;
