import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { FC } from "react";
import { useNavigate } from "react-router-dom";
import { Alert, Container } from "@/components/ui";
import GameForm, { emptyGameFormValues } from "@/forms/GameForm";
import { queryKeys } from "@/lib/query/keys";
import { useI18n } from "@/lib/stores/i18n";
import { GameService } from "@/services/game";
import { handleRequest } from "@/utilities/request";

const GameCreateView: FC = () => {
  const { t } = useI18n();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  // A new game goes to the end of the list rather than tying with everything
  // else on display_order 0; the order itself is changed by dragging on the
  // list page. This read is normally already cached by that page.
  const { data: games = [] } = useQuery({
    queryKey: queryKeys.games.list(),
    queryFn: GameService.list,
  });

  return (
    <Container size="md" className="space-y-4 py-6">
      <h1 className="text-xl font-semibold text-gray-800 dark:text-gray-100">
        {t("games.create.title")}
      </h1>

      <Alert variant="info" description={t("games.create.hint")} />

      <GameForm
        initialValues={emptyGameFormValues}
        submitLabel={t("games.create.submit")}
        onSubmit={async (values) => {
          const { data } = await handleRequest(
            () =>
              GameService.create({
                ...values,
                external_links_json: JSON.stringify(values.external_links),
                display_order: games.length,
              }),
            {
              method: "POST",
              errorMessages: { 400: "games.errors.titleRequired" },
              successMessage: "games.toast.created",
            },
          );
          if (data) {
            queryClient.invalidateQueries({
              queryKey: queryKeys.games.list(),
            });
            navigate(`/games/${data.id}`, { replace: true });
          }
        }}
      />
    </Container>
  );
};

export default GameCreateView;
