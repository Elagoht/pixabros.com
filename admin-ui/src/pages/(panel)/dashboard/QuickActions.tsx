import {
  IconArticle,
  IconAward,
  IconBolt,
  IconDeviceGamepad2,
  IconPhoto,
  IconUsers,
} from "@tabler/icons-react";
import classNames from "classnames";
import type { FC } from "react";
import { Link } from "react-router-dom";
import { Card } from "@/components/ui";
import { useI18n } from "@/lib/stores/i18n";
import { type Accent, accents } from "./accents";

// Only routes that actually exist belong here: a quick action that lands on a
// 404 is worse than no shortcut at all. Media has no create route -- uploading
// happens inside the library -- so it links to the library itself.
//
// Each action carries the same accent its dashboard card does, so the colour
// means "this is the games one" in both places rather than being decoration.
const ACTIONS: {
  id: string;
  to: string;
  icon: typeof IconBolt;
  labelKey: TranslationKey;
  accent: Accent;
}[] = [
  {
    id: "game",
    to: "/games/new",
    icon: IconDeviceGamepad2,
    labelKey: "pages.dashboard.actions.newGame",
    accent: "primary",
  },
  {
    id: "devlog",
    to: "/devlog/new",
    icon: IconArticle,
    labelKey: "pages.dashboard.actions.newDevlog",
    accent: "sky",
  },
  {
    id: "award",
    to: "/awards/new",
    icon: IconAward,
    labelKey: "pages.dashboard.actions.newAward",
    accent: "amber",
  },
  {
    id: "member",
    to: "/members/new",
    icon: IconUsers,
    labelKey: "pages.dashboard.actions.newMember",
    accent: "emerald",
  },
  {
    id: "media",
    to: "/media",
    icon: IconPhoto,
    labelKey: "pages.dashboard.actions.mediaLibrary",
    accent: "violet",
  },
];

const QuickActions: FC = () => {
  const { t } = useI18n();

  return (
    <Card>
      <Card.Header icon={IconBolt}>
        {t("pages.dashboard.actions.title")}
      </Card.Header>
      <Card.Body>
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-5">
          {ACTIONS.map((action) => {
            const theme = accents[action.accent];
            return (
              // A plain Link, not a Button: the shared Button renders a
              // <button> and has no `to`, so it cannot navigate.
              <Link
                key={action.id}
                to={action.to}
                className={classNames(
                  "flex items-center gap-2.5 rounded-lg border p-2.5 transition-all duration-150",
                  "hover:-translate-y-0.5 hover:shadow-sm",
                  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2",
                  theme.border,
                )}
              >
                <span
                  className={classNames(
                    "flex h-8 w-8 shrink-0 items-center justify-center rounded-lg",
                    theme.chip,
                  )}
                >
                  <action.icon size={17} />
                </span>
                <span className="truncate text-sm font-medium text-gray-700 dark:text-gray-200">
                  {t(action.labelKey)}
                </span>
              </Link>
            );
          })}
        </div>
      </Card.Body>
    </Card>
  );
};

export default QuickActions;
