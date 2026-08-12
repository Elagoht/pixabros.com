import { IconPlus, IconTrash, IconWorld } from "@tabler/icons-react";
import classNames from "classnames";
import { useField } from "formik";
import type { FC } from "react";
import Button from "./Button";

interface LinkListItem {
  label: string;
  url: string;
}

interface LinkListFieldProps {
  name: string;
  labelPlaceholder: string;
  urlPlaceholder: string;
  addLabel: string;
  emptyLabel: string;
  removeLabel: string;
  className?: string;
}

// A repeatable {label, url} editor. Values are kept as a real array so the
// caller can serialise them; the field never exposes raw JSON to the user.
const LinkListField: FC<LinkListFieldProps> = ({
  name,
  labelPlaceholder,
  urlPlaceholder,
  addLabel,
  emptyLabel,
  removeLabel,
  className,
}) => {
  const [field, meta, helpers] = useField<LinkListItem[]>(name);
  const links = Array.isArray(field.value) ? field.value : [];

  const update = (index: number, patch: Partial<LinkListItem>) => {
    helpers.setValue(
      links.map((link, i) => (i === index ? { ...link, ...patch } : link)),
    );
  };

  const remove = (index: number) => {
    helpers.setValue(links.filter((_, i) => i !== index));
  };

  const add = () => {
    helpers.setValue([...links, { label: "", url: "" }]);
  };

  // Yup reports per-row problems as an array of objects; anything else is a
  // message about the list as a whole.
  const rowErrors = Array.isArray(meta.error)
    ? (meta.error as unknown as (LinkListItem | undefined)[])
    : [];
  const listError =
    typeof meta.error === "string" && meta.touched ? meta.error : undefined;

  const inputClasses = (hasError: boolean) =>
    classNames(
      "w-full rounded-lg border bg-gray-50 px-3 py-2 text-sm outline-none transition-all duration-200",
      "text-gray-900 placeholder-gray-400 dark:bg-gray-800/50 dark:text-gray-50 dark:placeholder-gray-500",
      hasError
        ? "border-red-500 focus-visible:ring-2 focus-visible:ring-red-500"
        : "border-gray-200 focus-visible:ring-2 focus-visible:ring-primary-500 dark:border-gray-700",
    );

  return (
    <div className={classNames("space-y-2", className)}>
      {links.length === 0 && (
        <p className="text-xs text-gray-400 dark:text-gray-500">{emptyLabel}</p>
      )}

      {links.map((link, index) => {
        const error = rowErrors[index];
        return (
          <div
            // The row's position is its only stable identity: two rows may
            // legitimately hold identical text while being typed.
            key={index}
            className="flex items-start gap-2"
          >
            <div className="w-1/3 shrink-0">
              <input
                type="text"
                value={link.label}
                placeholder={labelPlaceholder}
                onChange={(e) => update(index, { label: e.target.value })}
                onBlur={() => helpers.setTouched(true)}
                className={inputClasses(!!error?.label)}
              />
              {error?.label && (
                <p className="mt-1 text-xs text-red-500">{error.label}</p>
              )}
            </div>

            <div className="min-w-0 flex-1">
              <div className="relative">
                <span className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-gray-400">
                  <IconWorld size={16} />
                </span>
                <input
                  type="url"
                  value={link.url}
                  placeholder={urlPlaceholder}
                  onChange={(e) => update(index, { url: e.target.value })}
                  onBlur={() => helpers.setTouched(true)}
                  className={classNames(inputClasses(!!error?.url), "pl-9")}
                />
              </div>
              {error?.url && (
                <p className="mt-1 text-xs text-red-500">{error.url}</p>
              )}
            </div>

            <Button
              variant="ghost"
              size="md"
              title={removeLabel}
              onClick={() => remove(index)}
            >
              <IconTrash size={16} />
            </Button>
          </div>
        );
      })}

      {listError && <p className="text-xs text-red-500">{listError}</p>}

      <Button variant="outline" size="sm" leftIcon={IconPlus} onClick={add}>
        {addLabel}
      </Button>
    </div>
  );
};

export default LinkListField;
