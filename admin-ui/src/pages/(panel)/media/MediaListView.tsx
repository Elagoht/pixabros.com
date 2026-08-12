import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type FC, useState } from "react";
import { Alert, Badge, Container, EmptyState, Loading } from "@/components/ui";
import { queryKeys } from "@/lib/query/keys";
import { useI18n } from "@/lib/stores/i18n";
import { MediaLibraryService } from "@/services/mediaLibrary";
import { handleRequest } from "@/utilities/request";
import MediaDetail from "./MediaDetail";

const MediaListView: FC = () => {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.media.library(),
    queryFn: MediaLibraryService.list,
  });

  const items = data?.items ?? [];
  const orphaned = data?.orphaned ?? 0;
  // Read from the freshly fetched list rather than held in state, so the panel
  // reflects a save without needing its own copy.
  const selected = items.find((item) => item.id === selectedId) ?? null;

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: queryKeys.media.all });

  const altTextMutation = useMutation({
    mutationFn: ({ id, altText }: { id: string; altText: string }) =>
      handleRequest(() => MediaLibraryService.setAltText(id, altText), {
        method: "PUT",
        successMessage: "media.toast.altSaved",
      }),
    onSuccess: invalidate,
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) =>
      handleRequest(() => MediaLibraryService.delete(id), {
        method: "DELETE",
        errorMessages: { 409: "media.errors.stillInUse" },
        successMessage: "media.toast.deleted",
      }),
    onSuccess: () => {
      setSelectedId(null);
      invalidate();
    },
  });

  if (isLoading) {
    return <Loading />;
  }

  if (isError) {
    return (
      <Container size="xl" className="py-6">
        <EmptyState title={t("common.error")} />
      </Container>
    );
  }

  return (
    <Container size="xl" className="space-y-4 py-6">
      <div className="flex flex-wrap items-center gap-3">
        <h1 className="text-xl font-semibold text-gray-800 dark:text-gray-100">
          {t("media.list.title")}
        </h1>
        <Badge variant="secondary">
          {t("media.list.count", { count: String(items.length) })}
        </Badge>
        {orphaned > 0 && (
          <Badge variant="warning">
            {t("media.list.orphaned", { count: String(orphaned) })}
          </Badge>
        )}
      </div>

      <Alert variant="info" description={t("media.sweepNote")} />

      {items.length === 0 ? (
        <EmptyState title={t("common.noResults")} />
      ) : (
        <ul className="grid gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
          {items.map((item) => {
            const inUse = item.usages.length > 0;
            return (
              <li key={item.id}>
                <button
                  type="button"
                  onClick={() => setSelectedId(item.id)}
                  className="group w-full space-y-1.5 rounded-lg border border-gray-200 bg-white p-2 text-left transition-colors hover:border-primary-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:border-gray-700 dark:bg-gray-950 dark:hover:border-primary-500"
                >
                  <img
                    src={item.url}
                    alt={item.alt_text || t("media.altText.missing")}
                    className="h-24 w-full rounded-md bg-gray-100 object-contain dark:bg-gray-900"
                  />
                  <p className="text-[10px] tabular-nums text-gray-500 dark:text-gray-400">
                    {item.width} × {item.height}
                  </p>
                  <div className="flex flex-wrap gap-1">
                    {inUse ? (
                      <Badge variant="outline">
                        {t(
                          `media.modules.${item.usages[0].module}` as TranslationKey,
                        )}
                        {item.usages.length > 1 &&
                          ` +${item.usages.length - 1}`}
                      </Badge>
                    ) : (
                      <Badge variant="warning">{t("media.unused")}</Badge>
                    )}
                    {!item.alt_text && (
                      <Badge variant="secondary">
                        {t("media.altText.missing")}
                      </Badge>
                    )}
                  </div>
                </button>
              </li>
            );
          })}
        </ul>
      )}

      <MediaDetail
        // Remounting on selection change resets the alt-text field to the
        // selected image's own value.
        key={selected?.id ?? "none"}
        item={selected}
        isSaving={altTextMutation.isPending}
        isDeleting={deleteMutation.isPending}
        onClose={() => setSelectedId(null)}
        onSaveAltText={(altText) => {
          if (selected) {
            altTextMutation.mutate({ id: selected.id, altText });
          }
        }}
        onDelete={() => {
          if (selected) {
            deleteMutation.mutate(selected.id);
          }
        }}
      />
    </Container>
  );
};

export default MediaListView;
