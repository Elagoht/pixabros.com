import { IconPencil, IconTrash } from "@tabler/icons-react";
import type { FC, ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { Badge, DataTable } from "@/components/ui";
import { useI18n } from "@/lib/stores/i18n";

interface AwardsTableProps {
  awards: ResponseAward[];
  games: ResponseGame[];
  isLoading: boolean;
  error?: string;
  onDelete: (award: ResponseAward) => void;
  toolbarActions?: ReactNode;
  sortBy?: string;
  sortDir?: "asc" | "desc";
  onSortChange?: (columnId: string, direction: "asc" | "desc") => void;
}

const AwardsTable: FC<AwardsTableProps> = ({
  awards,
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

  const columns: DataTableColumn<ResponseAward>[] = [
    {
      id: "title",
      header: t("awards.columns.title"),
      accessor: "title",
      sortable: true,
      onClick: (award) => navigate(`/awards/${award.id}`),
    },
    {
      id: "issuer",
      header: t("awards.columns.issuer"),
      accessor: "issuer",
      sortable: true,
    },
    {
      id: "date",
      header: t("awards.columns.date"),
      accessor: "date",
      type: "date",
      sortable: true,
    },
    {
      id: "game",
      header: t("awards.columns.game"),
      accessor: (award) =>
        award.game_id ? (gameTitles.get(award.game_id) ?? "") : "",
      cell: (_value, award) => {
        const title = award.game_id ? gameTitles.get(award.game_id) : undefined;
        if (!title) {
          return <span className="text-gray-400 dark:text-gray-600">—</span>;
        }
        return <Badge variant="outline">{title}</Badge>;
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
          onClick: (award) => navigate(`/awards/${award.id}`),
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
      data={awards}
      getRowId={(award) => award.id}
      isLoading={isLoading}
      isEmpty={!isLoading && awards.length === 0}
      error={error}
      toolbarActions={toolbarActions}
      sortBy={sortBy}
      sortDir={sortDir}
      onSortChange={onSortChange}
      hidePagination
    />
  );
};

export default AwardsTable;
