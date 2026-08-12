import {
  IconAlertCircle,
  IconAlertTriangle,
  IconCheck,
  IconInfoCircle,
  IconX,
} from "@tabler/icons-react";
import classNames from "classnames";
import { type FC, useState } from "react";

type AlertVariant = "info" | "success" | "warning" | "error";

interface AlertProps {
  variant?: AlertVariant;
  title?: string;
  description?: string;
  icon?: IconElement;
  closable?: boolean;
  className?: string;
}

const variantConfig: Record<
  AlertVariant,
  {
    icon: IconElement;
    border: string;
    bg: string;
    text: string;
    shadow: string;
  }
> = {
  info: {
    icon: IconInfoCircle,
    border: "border-blue-400",
    bg: "bg-blue-50 dark:bg-blue-900/20",
    text: "text-blue-700 dark:text-blue-300",
    shadow: "",
  },
  success: {
    icon: IconCheck,
    border: "border-green-400",
    bg: "bg-green-50 dark:bg-green-900/20",
    text: "text-green-700 dark:text-green-300",
    shadow: "",
  },
  warning: {
    icon: IconAlertTriangle,
    border: "border-yellow-400",
    bg: "bg-yellow-50 dark:bg-yellow-900/20",
    text: "text-yellow-700 dark:text-yellow-400",
    shadow: "",
  },
  error: {
    icon: IconAlertCircle,
    border: "border-red-400",
    bg: "bg-red-50 dark:bg-red-900/20",
    text: "text-red-700 dark:text-red-300",
    shadow: "",
  },
};

const Alert: FC<AlertProps> = ({
  variant = "info",
  title,
  description,
  icon,
  closable = false,
  className,
}) => {
  const [visible, setVisible] = useState(true);
  if (!visible) {
    return null;
  }

  const config = variantConfig[variant];
  const Icon = icon ?? config.icon;

  return (
    <div
      className={classNames(
        "flex gap-3 rounded-lg border-l-4 px-4 py-3 transition-all duration-200",
        config.border,
        config.bg,
        config.shadow,
        className,
      )}
    >
      <Icon size={20} className={classNames("mt-0.5 shrink-0", config.text)} />
      <div className="min-w-0 flex-1">
        {title && (
          <p className={classNames("text-sm font-medium", config.text)}>
            {title}
          </p>
        )}
        {description && (
          <p
            className={classNames(
              "text-sm",
              title && "mt-1",
              config.text,
              "opacity-80",
            )}
          >
            {description}
          </p>
        )}
      </div>
      {closable && (
        <button
          type="button"
          onClick={() => setVisible(false)}
          className={classNames(
            "shrink-0 rounded p-0.5 hover:opacity-70",
            config.text,
          )}
        >
          <IconX size={16} />
        </button>
      )}
    </div>
  );
};

export default Alert;
