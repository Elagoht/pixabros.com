import type { FC } from "react";

const Loading: FC = () => {
  return (
    <div className="flex h-screen w-full items-center justify-center">
      <div className="h-10 w-10 animate-spin rounded-full border-[3px] border-gray-200 border-t-primary-500 dark:border-gray-700 dark:border-t-primary-400" />
    </div>
  );
};

export default Loading;
