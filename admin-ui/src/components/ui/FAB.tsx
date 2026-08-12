import { IconPlus } from "@tabler/icons-react";
import classNames from "classnames";
import type { ComponentPropsWithoutRef, FC } from "react";

interface FABProps extends Omit<ComponentPropsWithoutRef<"button">, "type"> {
  icon?: IconElement;
  position?: "bottom-right" | "bottom-left" | "top-right" | "top-left";
  variant?: "default" | "secondary";
}

const positionClasses: Record<string, string> = {
  "bottom-right": "bottom-6 right-6",
  "bottom-left": "bottom-6 left-6",
  "top-right": "top-6 right-6",
  "top-left": "top-6 left-6",
};

const FAB: FC<FABProps> = ({
  icon: Icon = IconPlus,
  position = "bottom-right",
  variant = "default",
  className,
  children,
  ...props
}) => {
  return (
    <button
      type="button"
      className={classNames(
        "fixed z-40 flex items-center gap-2 rounded-full font-medium transition-all duration-200 ease-out",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2",
        "ring-offset-white dark:ring-offset-gray-950",
        variant === "default"
          ? "bg-primary-500 text-white hover:bg-primary-400 hover:ring-2 hover:ring-primary-400/50 hover:ring-offset-2 active:bg-primary-600"
          : "bg-secondary-700 text-white hover:bg-secondary-600 hover:ring-2 hover:ring-secondary-500/50 hover:ring-offset-2 active:bg-secondary-800",
        children ? "px-4 py-3" : "h-14 w-14 justify-center",
        positionClasses[position],
        className,
      )}
      {...props}
    >
      <Icon size={22} className="shrink-0" />
      {children && <span className="text-sm">{children}</span>}
    </button>
  );
};

export default FAB;
