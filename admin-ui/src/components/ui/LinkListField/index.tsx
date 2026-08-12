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
import LinkRow from "./LinkRow";

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

// A repeatable, drag-orderable {label, url} editor. Values stay a real array
// so the caller can serialise them; the field never exposes raw JSON.
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

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
  );

  // Rows have no id of their own, so position doubles as the drag id. That is
  // safe because the array cannot change mid-gesture: it is only rewritten on
  // drop, once the drag is already over.
  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) {
      return;
    }
    const from = Number(active.id);
    const to = Number(over.id);
    const next = [...links];
    const [moved] = next.splice(from, 1);
    next.splice(to, 0, moved);
    helpers.setValue(next);
  };

  // Yup reports per-row problems as an array of objects; anything else is a
  // message about the list as a whole.
  const rowErrors = Array.isArray(meta.error)
    ? (meta.error as unknown as (LinkListItem | undefined)[])
    : [];
  const listError =
    typeof meta.error === "string" && meta.touched ? meta.error : undefined;

  return (
    <div className={classNames("space-y-2", className)}>
      {links.length === 0 && (
        <p className="text-xs text-gray-400 dark:text-gray-500">{emptyLabel}</p>
      )}

      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragEnd={handleDragEnd}
      >
        <div className="space-y-2">
          {links.map((link, index) => (
            <LinkRow
              // Position is the row's only identity: two rows may hold
              // identical text while being typed.
              key={index}
              id={String(index)}
              label={link.label}
              url={link.url}
              labelPlaceholder={labelPlaceholder}
              urlPlaceholder={urlPlaceholder}
              removeLabel={removeLabel}
              labelError={rowErrors[index]?.label}
              urlError={rowErrors[index]?.url}
              draggable={links.length > 1}
              onChange={(patch) => update(index, patch)}
              onBlur={() => helpers.setTouched(true)}
              onRemove={() => remove(index)}
            />
          ))}
        </div>
      </DndContext>

      {listError && <p className="text-xs text-red-500">{listError}</p>}

      <Button variant="outline" size="sm" leftIcon={IconPlus} onClick={add}>
        {addLabel}
      </Button>
    </div>
  );
};

export default LinkListField;
