import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { FC } from "react";
import { Container, EmptyState, Loading } from "@/components/ui";
import SettingsForm from "@/forms/SettingsForm";
import { queryKeys } from "@/lib/query/keys";
import { useI18n } from "@/lib/stores/i18n";
import { SettingsService } from "@/services/settings";
import { handleRequest } from "@/utilities/request";

interface SettingsGroupViewProps {
  group: SettingsGroupName;
}

// Both settings screens are this one view: the group decides which registry
// the server hands back, and the form is built from that.
const SettingsGroupView: FC<SettingsGroupViewProps> = ({ group }) => {
  const { t } = useI18n();
  const queryClient = useQueryClient();

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.settings.group(group),
    queryFn: () => SettingsService.get(group),
  });

  if (isLoading) {
    return <Loading />;
  }

  if (isError || !data) {
    return (
      <Container size="md" className="py-6">
        <EmptyState title={t("common.error")} />
      </Container>
    );
  }

  return (
    <Container size="md" className="space-y-4 py-6">
      <h1 className="text-xl font-semibold text-gray-800 dark:text-gray-100">
        {t(`settings.${group}.title` as TranslationKey)}
      </h1>

      <SettingsForm
        definitions={data.definitions}
        values={data.values}
        title={t(`settings.${group}.title` as TranslationKey)}
        hint={t(`settings.${group}.hint` as TranslationKey)}
        onSubmit={async (values) => {
          const { data: saved } = await handleRequest(
            () => SettingsService.update(group, { values }),
            {
              method: "PUT",
              errorMessages: {
                400: "settings.errors.invalidFields",
              },
              successMessage: "settings.toast.saved",
            },
          );
          if (saved) {
            queryClient.invalidateQueries({
              queryKey: queryKeys.settings.group(group),
            });
          }
        }}
      />
    </Container>
  );
};

export default SettingsGroupView;
