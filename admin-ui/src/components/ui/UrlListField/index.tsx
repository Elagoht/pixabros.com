import {
  closestCenter,
  DndContext,
  type DragEndEvent,
  PointerSensor,
  useSensor,
  useSensors,
} from "@dnd-kit/core";
import { IconPlus } from "@tabler/icons-react";
import classNames from "classnames";
import { useField } from "formik";
import type { FC } from "react";
import Button from "../Button";
import UrlRow from "./UrlRow";

interface UrlListFieldProps {
  name: string;
  label?: string;
  placeholder: string;
  addLabel: string;
  emptyLabel: string;
  removeLabel: string;
  className?: string;
}

// A repeatable, drag-orderable list of plain addresses. Unlike LinkListField
// the entries carry no label, because the thing this edits -- JSON-LD's
// sameAs -- is a bare list of URLs.
const UrlListField: FC<UrlListFieldProps> = ({
  name,
  label,
  placeholder,
  addLabel,
  emptyLabel,
  removeLabel,
  className,
}) => {
  const [field, meta, helpers] = useField<string[]>(name);
  const urls = Array.isArray(field.value) ? field.value : [];

  const update = (index: number, value: string) => {
    helpers.setValue(urls.map((url, i) => (i === index ? value : url)));
  };

  const remove = (index: number) => {
    helpers.setValue(urls.filter((_, i) => i !== index));
  };

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
  );

  // Rows have no id of their own, so position doubles as the drag id. That is
  // safe because the array is only rewritten on drop, once the drag is over.
  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) {
      return;
    }
    const next = [...urls];
    const [moved] = next.splice(Number(active.id), 1);
    next.splice(Number(over.id), 0, moved);
    helpers.setValue(next);
  };

  // Yup reports per-entry problems as an array; anything else is about the
  // list as a whole.
  const rowErrors = Array.isArray(meta.error)
    ? (meta.error as unknown as (string | undefined)[])
    : [];
  const listError =
    typeof meta.error === "string" && meta.touched ? meta.error : undefined;

  return (
    <div className={classNames("w-full space-y-2", className)}>
      {label && (
        <span className="block text-sm font-medium text-gray-700 dark:text-gray-300">
          {label}
        </span>
      )}

      {urls.length === 0 && (
        <p className="text-xs text-gray-400 dark:text-gray-500">{emptyLabel}</p>
      )}

      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragEnd={handleDragEnd}
      >
        <div className="space-y-2">
          {urls.map((url, index) => (
            <UrlRow
              // Position is the row's only identity: two rows may hold
              // identical text while being typed.
              key={index}
              id={String(index)}
              url={url}
              placeholder={placeholder}
              removeLabel={removeLabel}
              error={rowErrors[index]}
              draggable={urls.length > 1}
              onChange={(value) => update(index, value)}
              onBlur={() => helpers.setTouched(true)}
              onRemove={() => remove(index)}
            />
          ))}
        </div>
      </DndContext>

      {listError && <p className="text-xs text-red-500">{listError}</p>}

      <Button
        variant="outline"
        size="sm"
        leftIcon={IconPlus}
        onClick={() => helpers.setValue([...urls, ""])}
      >
        {addLabel}
      </Button>
    </div>
  );
};

export default UrlListField;
