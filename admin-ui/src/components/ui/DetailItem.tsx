import type { FC, ReactNode } from "react";

interface DetailItemProps {
  label: string;
  value: ReactNode;
}

const DetailItem: FC<DetailItemProps> = ({ label, value }) => (
  <div className="rounded-md border bg-gray-50/50 px-3 py-2 border-gray-100 dark:border-gray-700 dark:bg-gray-800">
    <dt className="text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-400">
      {label}
    </dt>
    <dd className="mt-1 text-sm font-medium text-gray-900 dark:text-white">
      {value ?? (
        <span className="text-gray-300 dark:text-gray-600">&mdash;</span>
      )}
    </dd>
  </div>
);

export default DetailItem;
