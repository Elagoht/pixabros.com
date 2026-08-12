import type { Icon } from "@tabler/icons-react";
import classNames from "classnames";
import type { FC } from "react";
import { Link } from "react-router-dom";
import { Card, Skeleton } from "@/components/ui";
import { type Accent, accents } from "./accents";

interface StatCardProps {
  icon: Icon;
  label: string;
  value?: number;
  /** Reads under the number, e.g. "2 taslak". Omitted when there is nothing to add. */
  hint?: string;
  to: string;
  accent: Accent;
  loading?: boolean;
  /**
   * Colours the number itself. Reserved for a figure that is asking for action
   * rather than reporting state -- unread messages, not "12 games".
   */
  highlight?: boolean;
}

const StatCard: FC<StatCardProps> = ({
  icon: IconComponent,
  label,
  value,
  hint,
  to,
  accent,
  loading = false,
  highlight = false,
}) => {
  const theme = accents[accent];

  return (
    // The whole card is the link: a KPI is only useful if the next click takes
    // you to the thing it counts.
    <Link
      to={to}
      className="group rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2"
    >
      <Card
        className={classNames(
          // overflow-clip is what lets the icon sit past the card's bottom edge
          // without growing the card or spilling onto its neighbour.
          "relative h-full overflow-clip transition-all duration-200 group-hover:-translate-y-0.5 group-hover:shadow-md",
          theme.border,
        )}
      >
        {/* A wash rather than a flat fill, so the colour reads as a tint on the
            card instead of repainting it. */}
        <div
          aria-hidden
          className={classNames(
            "pointer-events-none absolute inset-0 bg-gradient-to-br to-transparent",
            theme.wash,
          )}
        />

        {/* Decoration, not information: the number and its label carry the
            meaning, so this is hidden from assistive tech. */}
        <IconComponent
          size={128}
          stroke={1.25}
          aria-hidden
          className={classNames(
            "pointer-events-none absolute -bottom-5 -right-4 rotate-[24deg] transition-transform duration-300 group-hover:scale-105",
            theme.bgIcon,
          )}
        />

        <Card.Body className="relative">
          <div className="flex items-center gap-2">
            <IconComponent
              size={16}
              className={classNames("shrink-0", theme.value)}
            />
            <div className="text-sm font-medium text-gray-600 dark:text-gray-300">
              {label}
            </div>
          </div>

          {loading ? (
            <Skeleton variant="text" width={56} className="mt-1 h-8" />
          ) : (
            <div
              className={classNames(
                "text-3xl font-bold",
                highlight ? theme.value : "text-gray-900 dark:text-gray-50",
              )}
            >
              {value ?? 0}
            </div>
          )}

          {hint && !loading && (
            <div className="truncate text-xs text-gray-500 dark:text-gray-400">
              {hint}
            </div>
          )}
        </Card.Body>
      </Card>
    </Link>
  );
};

export default StatCard;
