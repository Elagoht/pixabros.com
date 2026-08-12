import { useQuery } from "@tanstack/react-query";
import type { FC } from "react";
import { Select } from "@/components/ui";
import { queryKeys } from "@/lib/query/keys";
import { useI18n } from "@/lib/stores/i18n";
import { GameService } from "@/services/game";

interface GameSelectProps {
  name: string;
  label: string;
  /** Label for the "not linked to any game" option. */
  noneLabel: string;
}

// A picker over the game list, used by anything that can reference a game.
// The empty option means "no game"; the column behind it is nullable.
const GameSelect: FC<GameSelectProps> = ({ name, label, noneLabel }) => {
  useI18n();

  // Usually already cached by the games screens.
  const { data: games = [] } = useQuery({
    queryKey: queryKeys.games.list(),
    queryFn: () => GameService.list(),
  });

  const options = [
    { label: noneLabel, value: "" },
    ...games.map((game) => ({ label: game.title, value: game.id })),
  ];

  return <Select name={name} label={label} options={options} />;
};

export default GameSelect;
