import { useEffect } from "react";

export const useGlobalShortcut = (
  keys: { key: string; metaKey?: boolean; ctrlKey?: boolean },
  callback: () => void,
) => {
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const metaMatch =
        keys.metaKey === undefined ? true : e.metaKey === keys.metaKey;
      const ctrlMatch =
        keys.ctrlKey === undefined ? true : e.ctrlKey === keys.ctrlKey;

      if (
        e.key.toLowerCase() === keys.key.toLowerCase() &&
        metaMatch &&
        ctrlMatch
      ) {
        e.preventDefault();
        callback();
      }
    };

    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [keys.key, keys.metaKey, keys.ctrlKey, callback]);
};
