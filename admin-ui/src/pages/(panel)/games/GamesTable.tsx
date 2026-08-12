import { IconExternalLink, IconPencil, IconTrash } from "@tabler/icons-react";
import type { FC } from "react";
import { useNavigate } from "react-router-dom";
import { Badge, DataTable } from "@/components/ui";
import { useI18n } from "@/lib/stores/i18n";

interface GamesTableProps {
  games: ResponseGame[];
  isLoading: boolean;
  error?: string;
  onDelete: (game: ResponseGame) => void;
  toolbarActions?: React.ReactNode;
  sortBy?: string;
  sortDir?: "asc" | "desc";
  onSortChange?: (columnId: string, direction: "asc" | "desc") => void;
}

const GamesTable: FC<GamesTableProps> = ({
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

  const distributionLabels = (game: ResponseGame): string[] => {
    const labels: string[] = [];
    if (game.is_browser_playable) {
      labels.push(t("games.distribution.browser"));
    }
    if (game.is_for_sale) {
      labels.push(t("games.distribution.forSale"));
    }
    return labels;
  };

  const columns: DataTableColumn<ResponseGame>[] = [
    {
      id: "title",
      header: t("games.columns.title"),
      accessor: "title",
      sortable: true,
      filterable: true,
      onClick: (game) => navigate(`/games/${game.id}`),
    },
    {
      id: "is_published",
      header: t("games.columns.status"),
      accessor: "is_published",
      sortable: true,
      cell: (value) => (
        <Badge variant={value ? "success" : "secondary"}>
          {value ? t("games.status.published") : t("games.status.draft")}
        </Badge>
      ),
    },
    {
      id: "distribution",
      header: t("games.columns.distribution"),
      accessor: (game) => distributionLabels(game).join(", "),
      cell: (_value, game) => {
        const labels = distributionLabels(game);
        if (labels.length === 0) {
          return <span className="text-gray-400 dark:text-gray-600">—</span>;
        }
        return (
          <span className="flex flex-wrap gap-1">
            {labels.map((label) => (
              <Badge key={label} variant="outline">
                {label}
              </Badge>
            ))}
          </span>
        );
      },
    },
    {
      id: "display_order",
      header: t("games.columns.order"),
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
          icon: IconExternalLink,
          label: t("games.actions.viewOnSite"),
          // /play/ is served by Go outside the SPA, so this leaves the app
          // entirely rather than going through react-router.
          disabled: (game: ResponseGame) => !game.is_browser_playable,
          onClick: (game: ResponseGame) =>
            window.open(`/play/${game.slug}/`, "_blank", "noreferrer"),
        },
        {
          icon: IconPencil,
          label: t("common.edit"),
          onClick: (game) => navigate(`/games/${game.id}`),
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
      data={games}
      getRowId={(game) => game.id}
      isLoading={isLoading}
      isEmpty={!isLoading && games.length === 0}
      error={error}
      toolbarActions={toolbarActions}
      sortBy={sortBy}
      sortDir={sortDir}
      onSortChange={onSortChange}
      hidePagination
    />
  );
};

export default GamesTable;
