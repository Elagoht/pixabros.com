// Tailwind generates classes by scanning source for complete strings, so an
// accent cannot be built as `text-${accent}-500` -- that class would simply
// never exist in the stylesheet. Every variant is written out here in full.
export type Accent =
  | "primary"
  | "rose"
  | "sky"
  | "amber"
  | "emerald"
  | "violet";

interface AccentClasses {
  /** The oversized decorative icon bleeding out of the card corner. */
  bgIcon: string;
  /** A wash behind the card content, strongest at the top-left. */
  wash: string;
  /** Small solid icon chip, used by the quick actions. */
  chip: string;
  /** The card's border once it is carrying an accent. */
  border: string;
  /** The number itself, when the card is highlighted. */
  value: string;
}

export const accents: Record<Accent, AccentClasses> = {
  primary: {
    bgIcon: "text-primary-500/20 dark:text-primary-400/20",
    wash: "from-primary-50 dark:from-primary-950/40",
    chip: "bg-primary-500/10 text-primary-600 dark:bg-primary-400/10 dark:text-primary-400",
    border: "border-primary-200 dark:border-primary-900/60",
    value: "text-primary-600 dark:text-primary-400",
  },
  rose: {
    bgIcon: "text-rose-500/20 dark:text-rose-400/20",
    wash: "from-rose-50 dark:from-rose-950/40",
    chip: "bg-rose-500/10 text-rose-600 dark:bg-rose-400/10 dark:text-rose-400",
    border: "border-rose-200 dark:border-rose-900/60",
    value: "text-rose-600 dark:text-rose-400",
  },
  sky: {
    bgIcon: "text-sky-500/20 dark:text-sky-400/20",
    wash: "from-sky-50 dark:from-sky-950/40",
    chip: "bg-sky-500/10 text-sky-600 dark:bg-sky-400/10 dark:text-sky-400",
    border: "border-sky-200 dark:border-sky-900/60",
    value: "text-sky-600 dark:text-sky-400",
  },
  amber: {
    bgIcon: "text-amber-500/20 dark:text-amber-400/20",
    wash: "from-amber-50 dark:from-amber-950/40",
    chip: "bg-amber-500/10 text-amber-600 dark:bg-amber-400/10 dark:text-amber-400",
    border: "border-amber-200 dark:border-amber-900/60",
    value: "text-amber-600 dark:text-amber-400",
  },
  emerald: {
    bgIcon: "text-emerald-500/20 dark:text-emerald-400/20",
    wash: "from-emerald-50 dark:from-emerald-950/40",
    chip: "bg-emerald-500/10 text-emerald-600 dark:bg-emerald-400/10 dark:text-emerald-400",
    border: "border-emerald-200 dark:border-emerald-900/60",
    value: "text-emerald-600 dark:text-emerald-400",
  },
  violet: {
    bgIcon: "text-violet-500/20 dark:text-violet-400/20",
    wash: "from-violet-50 dark:from-violet-950/40",
    chip: "bg-violet-500/10 text-violet-600 dark:bg-violet-400/10 dark:text-violet-400",
    border: "border-violet-200 dark:border-violet-900/60",
    value: "text-violet-600 dark:text-violet-400",
  },
};
