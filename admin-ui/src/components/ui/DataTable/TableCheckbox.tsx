import { IconCheck, IconMinus } from "@tabler/icons-react";
import classNames from "classnames";
import { checkboxBase } from "@/utilities/constants";

interface TableCheckboxProps {
  checked: boolean;
  indeterminate?: boolean;
  onChange: (event: React.ChangeEvent<HTMLInputElement>) => void;
}

const TableCheckbox = ({
  checked,
  indeterminate,
  onChange,
}: TableCheckboxProps) => {
  return (
    <label className="relative inline-flex cursor-pointer items-center justify-center">
      <input
        type="checkbox"
        checked={checked}
        onChange={onChange}
        ref={(el) => {
          if (el) {
            el.indeterminate = indeterminate ?? false;
          }
        }}
        className={classNames(
          checkboxBase,
          "transition-all duration-200",
          checked || indeterminate
            ? "border-primary-500 bg-primary-500"
            : "hover:ring-2 hover:ring-primary-400/30 hover:ring-offset-1",
        )}
      />
      <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
        {checked && (
          <IconCheck size={14} strokeWidth={3} className="text-white" />
        )}
        {!checked && indeterminate && (
          <IconMinus size={12} strokeWidth={3} className="text-white" />
        )}
      </div>
    </label>
  );
};

export default TableCheckbox;
