# Component Reference

**Before writing ANY JSX|TSX, check this file.** If a component exists here, use it. Do NOT create ad-hoc `div`/`span`/`button` elements.

All imports: `import { ComponentName } from "@/components/ui";`

---

## Layout

| Component    | Props                                                                               |
| ------------ | ----------------------------------------------------------------------------------- |
| `Container`  | `size?: "sm" \| "md" \| "lg" \| "xl"`, `className?`                                 |
| `PageLayout` | `children`, `className?`                                                            |
| `Sidebar`    | `collapsed`, `onToggle`, `logo?`, `appName?`, `children`, `className?`              |
| `Navbar`     | `children`, `className?`                                                            |
| `MenuBar`    | `items: { id, label, children?: { id, label, icon?, onClick? }[] }[]`, `className?` |
| `TOC`        | `items: { id, label, level? }[]`, `className?`                                      |
| `Drawer`     | `open`, `onClose`, `position?: "left" \| "right"`, `children`, `className?`         |
| `ScrollArea` | `children`, `className?`, `maxHeight?`                                              |

## Forms & Inputs

| Component        | Props                                                                        |
| ---------------- | ---------------------------------------------------------------------------- |
| `Input`          | `name`, `type?`, `label?`, `placeholder?`, `leftIcon?`, `rightIcon?`         |
| `Textarea`       | `name`, `label?`, `rows?`, `placeholder?`                                    |
| `MarkdownEditor` | `name`, `label?`, `rows?`, `placeholder?` — markdown field with a sanitised preview tab |
| `Select`         | `name`, `label?`, `options: { label, value }[]`, `placeholder?`, `multiple?` |
| `Combobox`       | `name`, `label?`, `options: { label, value }[]`, `placeholder?`, `multiple?` |
| `Checkbox`       | `name`, `label`                                                              |
| `CheckboxGroup`  | `name`, `label`, `options: { label, value }[]`                               |
| `RadioGroup`     | `name`, `label`, `options: { label, value }[]`                               |
| `Switch`         | `name`, `label`                                                              |
| `FileInput`      | `name`, `label?`, `accept?`                                                  |
| `DatePicker`     | `name`, `label?`, `placeholder?`                                             |
| `DateTimePicker` | `name`, `label?`, `placeholder?`                                             |
| `PhoneInput`     | `name`, `label?`                                                             |
| `OTPInput`       | `value`, `onChange`, `digits?` (default 6), `disabled?`                      |
| `ColorPicker`    | `value`, `onChange`, `className?`                                            |
| `RangeSlider`    | `minValue`, `maxValue`, `onChange`, `showValue?`                             |
| `Slider`         | `value`, `onChange`, `min?`, `max?`, `step?`, `showValue?`                   |
| `Rating`         | `value`, `onChange`, `size?`, `allowHalf?`, `disabled?`                      |
| `Keywords`       | `name`, `label?`, `placeholder?`, `output: "string" \| "array"`              |
| `LinkListField`  | `name`, `labelPlaceholder`, `urlPlaceholder`, `addLabel`, `emptyLabel`, `removeLabel` — repeatable `{label, url}` rows |
| `Label`          | `children`, `htmlFor?`, `className?`                                         |
| `FieldSet`       | `legend`, `icon?`, `description?`, `error?`, `children`, `className?`        |
| `Chip`           | `children`, `onRemove`                                                       |

## Buttons

| Component     | Props                                                                                                                                                       |
| ------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Button`      | `variant?: "default" \| "secondary" \| "ghost" \| "destructive" \| "outline"`, `size?: "sm" \| "md" \| "lg"`, `leftIcon?`, `rightIcon?`, `to?`, `disabled?` |
| `ButtonGroup` | `children`                                                                                                                                                  |
| `CopyButton`  | `value`, `timeout?`, `children?`                                                                                                                            |
| `FAB`         | `icon?`, `position?: "bottom-right" \| "bottom-left" \| "top-right" \| "top-left"`, `variant?: "default" \| "secondary"`, `onClick?`                        |

## Content Display

| Component        | Props                                                                                                                |
| ---------------- | -------------------------------------------------------------------------------------------------------------------- |
| `Card`           | `children`, `className?` — use `Card.Header` (`icon?`), `Card.Body`, `Card.Footer`                                             |
| `Badge`          | `variant?: "default" \| "secondary" \| "success" \| "warning" \| "destructive" \| "outline"`, `children`             |
| `Avatar`         | `name?`, `src?`, `size?: "xs" \| "sm" \| "md" \| "lg" \| "xl"`, `status?: "online" \| "away" \| "busy" \| "offline"` |
| `Kbd`            | `children` (for keyboard shortcuts like `⌘K`)                                                                        |
| `CodeBlock`      | `code`, `language?`, `filename?`, `maxHeight?`                                                                       |
| `Image`          | `src`, `alt`, `width`, `height`, `loading?: "lazy" \| "eager"`                                                       |
| `ImagePreview`   | `src`, `alt`, `className?` — thumbnail that opens the full image in a lightbox (max 90vh/90vw) |
| `Separator`      | `className?`                                                                                                         |
| `Skeleton`       | `className?`, `variant?: "text" \| "rect" \| "circle"`, `width?`, `height?`                                          |
| `Loading`        | `className?`                                                                                                         |
| `EmptyState`     | `title`, `description?`, `action?`, `icon?`                                                                          |
| `ProgressBar`    | `value`, `size?: "sm" \| "md" \| "lg"`, `showValue?`                                                                 |
| `ProgressCircle` | `value`, `size?`, `strokeWidth?`, `showValue?`                                                                       |
| `Timeline`       | `items: { id, title, description?, timestamp?, icon?, iconClassName? }[]`                                            |

## Navigation

| Component    | Props                                                                                             |
| ------------ | ------------------------------------------------------------------------------------------------- |
| `Breadcrumb` | `children`, `className?` — use `Breadcrumb.Item`                                                  |
| `Tabs`       | `items: { value, label, content }[]`, `defaultValue?`, `value?`, `onChange?`                      |
| `Stepper`    | `steps: string[]`, `activeStep`, `onStepClick?`                                                   |
| `Accordion`  | `items: { value, label, content }[]`, `type?: "single" \| "multiple"`, `defaultOpen?`             |
| `Pagination` | `page`, `totalPages`, `onChange`                                                                  |
| `ReorderModal` | `open`, `items: { id, label }[]`, `title`, `help`, `isSaving`, `onClose`, `onSave(orderedIds)` — drag-to-order list |
| `Carousel`   | `slides: { id, content }[]`, `autoPlay?`, `interval?`, `showDots?`, `showArrows?`, `aspectRatio?` |

## Data

| Component        | Props                                                                                                                        |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| `DataTable`      | `columns: DataTableColumn[]`, `data`, `getRowId`, `selectable?`, `isLoading?`, `isEmpty?`, `error?`, `renderRowContextMenu?` |
| `HierarchyTable` | `name` (formik field), `renderRow`, `renderActions?`, `saveButtonText?`                                                      |
| `CategoryTable`  | `categories: HierarchyNode[]`, `getCategoryLabel`, `onSave`, `onCreate`, `onDelete`, `renderFormFields`, `initialFormData`   |

## Overlay & Feedback

| Component        | Props                                                                                                                    |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------ |
| `Modal`          | `open`, `onClose`, `persistent?`, `children` — use `Modal.Header`, `Modal.Body`, `Modal.Footer`                          |
| `Dialog`         | `open`, `onClose`, `title`, `description?`, `onConfirm`, `onCancel?`, `confirmLabel?`, `cancelLabel?`, `confirmVariant?` |
| `Dropdown`       | `trigger`, `items: { id, label?, icon?, onClick?, danger?, disabled?, separator? }[]`, `align?`                          |
| `Popover`        | `children` — use `Popover.Trigger`, `Popover.Content`                                                                    |
| `HoverCard`      | `children` — use `HoverCard.Trigger`, `HoverCard.Content`                                                                |
| `ContextMenu`    | `children`, `items: { id, label?, icon?, onClick?, danger?, disabled?, separator? }[]`                                   |
| `CommandPalette` | `open`, `onClose`, `groups: { heading, items: { id, label, description?, icon?, onSelect }[] }[]`                        |
| `Tooltip`        | `content`, `children`, `position?: "top" \| "bottom" \| "left" \| "right"`                                               |
| `Alert`          | `variant?: "info" \| "success" \| "warning" \| "error"`, `title`, `description?`, `closable?`                            |

---

## Rules

1. **Check this file first.** If what you need exists here, use it.
2. **No raw `button` elements.** Use `Button`. No exceptions.
3. **No raw `input`/`select`/`textarea` elements.** Use the corresponding form component.
4. **No raw `div`/`span` for UI patterns** that have a component (badge, avatar, skeleton, separator, etc.).
5. **Use `classnames` for conditional classes**, not template literals.
6. **All components support dark mode.** Pass `dark:` variants in `className` if extending styles.
7. **Controlled forms use Formik.** Provide `FormikProvider` context. Form components use `useField(name)` internally.
8. **No inline styles.** Tailwind classes only. Exception: dynamic values like `width`, `color`, `transform`.
