import {
  IconDownload,
  IconEye,
  IconPencil,
  IconPlus,
} from "@tabler/icons-react";
import { useQuery } from "@tanstack/react-query";
import { type FC, useMemo } from "react";
import { Link, useNavigate } from "react-router-dom";
import { toast } from "sonner";
import {
  Badge,
  Button,
  Container,
  DataTable,
  FilterPanel,
} from "@/components/ui";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useFilters } from "@/hooks/useFilters";
import { usePageParams } from "@/hooks/usePageParams";
import { useSortParams } from "@/hooks/useSortParams";
import { queryKeys } from "@/lib/query/keys";
import { useI18n } from "@/lib/stores/i18n";
import { type UserListItem, userService } from "@/services/user";

const roleVariant = (
  role: string,
):
  | "default"
  | "secondary"
  | "success"
  | "warning"
  | "destructive"
  | "outline" => {
  const lower = role.toLowerCase();
  if (lower.includes("admin") || lower.includes("super")) {
    return "destructive";
  }
  if (lower.includes("supplier")) {
    return "warning";
  }
  if (lower.includes("center")) {
    return "default";
  }
  if (lower.includes("satın alma") || lower.includes("procurement")) {
    return "secondary";
  }
  if (lower.includes("approver") || lower.includes("onay")) {
    return "success";
  }
  if (lower.includes("category")) {
    return "outline";
  }
  return "default";
};

const sortFieldMap: Record<string, string> = {
  name: "first_name",
  email: "email",
  created_at: "created_at",
};

const UsersPage: FC = () => {
  const { t } = useI18n();
  const navigate = useNavigate();

  useBreadcrumb([
    { label: t("menu.definitions"), to: "/definitions" },
    { label: t("menu.users") },
  ]);

  const { page, pageSize, setPage, setPageSize } = usePageParams();
  const { sortBy, sortDir, ordering, setSort } = useSortParams(sortFieldMap);
  const filters = useFilters();

  const { data: roleOptions = [] } = useQuery({
    queryKey: queryKeys.users.roles(),
    queryFn: userService.roles,
  });

  const { data, isLoading, isError, error } = useQuery({
    queryKey: queryKeys.users.list({ page, pageSize, filters, ordering }),
    queryFn: () =>
      userService.list({ page, take: pageSize, ordering, ...filters }),
  });

  const totalCount = data?.total ?? 0;
  const results = data?.data ?? [];

  const handleExport = async () => {
    try {
      const blob = await userService.exportList(
        filters as Record<string, string>,
      );
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `${t("menu.users")}.xlsx`;
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      toast.error(t("suppliers.unknownError"));
    }
  };

  const filterDefs = useMemo(
    () => [
      {
        type: "text" as const,
        key: "email",
        label: t("suppliers.columns.email"),
        placeholder: `${t("suppliers.columns.email")}...`,
      },
      {
        type: "text" as const,
        key: "fullname",
        label: t("users.create.firstName"),
        placeholder: `${t("users.create.firstName")}...`,
      },
      {
        type: "text" as const,
        key: "project_name",
        label: t("menu.projects"),
        placeholder: `${t("menu.projects")}...`,
      },
      {
        type: "multiselect" as const,
        key: "role",
        label: t("profile.role"),
        options: roleOptions.map((r) => ({ label: r, value: r })),
      },
      {
        type: "select" as const,
        key: "sortBy",
        label: t("filters.sort"),
        placeholder: t("common.previous"),
        options: [
          { label: t("users.detail.title"), value: "created_at" },
          { label: t("users.create.firstName"), value: "name" },
          { label: t("suppliers.columns.email"), value: "email" },
        ],
      },
      {
        type: "select" as const,
        key: "sortDir",
        label: t("filters.direction"),
        placeholder: t("common.previous"),
        disabled: !sortBy,
        options: [
          { label: t("filters.ascending"), value: "asc" },
          { label: t("filters.descending"), value: "desc" },
        ],
      },
    ],
    [roleOptions, sortBy, t],
  );

  const columns = useMemo(
    () =>
      [
        {
          id: "name",
          header: t("users.create.firstName"),
          accessor: (row: UserListItem) => `${row.first_name} ${row.last_name}`,
          type: "string" as const,
        },
        {
          id: "email",
          header: t("suppliers.columns.email"),
          accessor: "email" as const,
          type: "string" as const,
        },
        {
          id: "project_name",
          header: t("menu.projects"),
          accessor: (row: UserListItem) => row.project_name ?? "-",
          type: "string" as const,
          sortable: false,
        },
        {
          id: "role",
          header: t("profile.role"),
          accessor: (row: UserListItem) => row.role.join(", "),
          type: "string" as const,
          sortable: false,
          cell: (_value: unknown, row: UserListItem) => (
            <div className="flex flex-wrap gap-1">
              {row.role.map((r) => (
                <Badge key={r} variant={roleVariant(r)}>
                  {t(`roles.${r}` as TranslationKey)}
                </Badge>
              ))}
            </div>
          ),
        },
        {
          id: "created_at",
          header: t("users.list.registrationDate"),
          accessor: "created_at" as const,
          type: "date" as const,
        },
        {
          id: "actions",
          header: "",
          accessor: "id" as const,
          type: "actions" as const,
          actions: [
            {
              icon: IconEye,
              label: t("suppliers.actions.view"),
              variant: "primary",
              onClick: (row: UserListItem) =>
                navigate(`/definitions/users/${row.id}`),
            },
            {
              icon: IconPencil,
              label: t("suppliers.actions.edit"),
              variant: "warning",
              onClick: (row: UserListItem) =>
                navigate(`/definitions/users/${row.id}/edit`),
            },
          ],
        },
      ] as DataTableColumn<UserListItem>[],
    [t, navigate],
  );

  return (
    <Container size="xl" className="space-y-6 py-6">
      <div className="flex flex-col gap-6 lg:flex-row">
        <div className="min-w-0 flex-1">
          <h1 className="mb-6 text-2xl font-bold text-gray-900 dark:text-gray-50">
            {t("users.list.title")}
          </h1>
          <DataTable<UserListItem>
            columns={columns}
            data={results}
            getRowId={(row) => row.id}
            page={page}
            pageSize={pageSize}
            totalCount={totalCount}
            onPageChange={setPage}
            onPageSizeChange={setPageSize}
            sortBy={sortBy}
            sortDir={sortDir}
            onSortChange={setSort}
            toolbarActions={
              <div className="flex items-center gap-2">
                <Link to="/definitions/users/new">
                  <Button variant="success" size="sm" leftIcon={IconPlus}>
                    {t("users.create.button")}
                  </Button>
                </Link>
                <Button
                  variant="warning"
                  size="sm"
                  leftIcon={IconDownload}
                  onClick={handleExport}
                >
                  {t("suppliers.actions.exportExcel")}
                </Button>
              </div>
            }
            isLoading={isLoading}
            isEmpty={!(isLoading || isError) && results.length === 0}
            error={
              isError
                ? error instanceof Error
                  ? error.message
                  : t("users.list.unknownError")
                : undefined
            }
          />
        </div>
        <FilterPanel
          filters={filterDefs}
          className="w-full shrink-0 lg:w-64"
          title={t("filters.title")}
          clearLabel={t("filters.clear")}
        />
      </div>
    </Container>
  );
};

export default UsersPage;
