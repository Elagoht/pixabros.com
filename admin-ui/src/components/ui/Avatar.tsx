import { IconUser } from "@tabler/icons-react";
import classNames from "classnames";
import { type FC, useState } from "react";
import Tooltip from "./Tooltip";

type AvatarSize = "xs" | "sm" | "md" | "lg" | "xl";
type AvatarStatus = "online" | "offline" | "away" | "busy";

interface AvatarProps {
  src?: string;
  name?: string;
  size?: AvatarSize;
  status?: AvatarStatus;
  className?: string;
}

const sizeClasses: Record<AvatarSize, string> = {
  xs: "h-6 w-6 text-xs",
  sm: "h-8 w-8 text-xs",
  md: "h-10 w-10 text-sm",
  lg: "h-12 w-12 text-base",
  xl: "h-16 w-16 text-lg",
};

const statusSizeClasses: Record<AvatarSize, string> = {
  xs: "h-1.5 w-1.5 ring-1",
  sm: "h-2 w-2 ring-1",
  md: "h-2.5 w-2.5 ring-2",
  lg: "h-3 w-3 ring-2",
  xl: "h-3.5 w-3.5 ring-2",
};

const statusColorClasses: Record<AvatarStatus, string> = {
  online: "bg-green-500",
  offline: "bg-gray-400 dark:bg-gray-500",
  away: "bg-yellow-400",
  busy: "bg-red-500",
};

const getInitials = (name: string): string => {
  const parts = name.trim().split(/\s+/);
  if (parts.length === 1) {
    return parts[0][0].toUpperCase();
  }
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
};

const Avatar: FC<AvatarProps> = ({
  src,
  name,
  size = "md",
  status,
  className,
}) => {
  const [imgError, setImgError] = useState(false);
  const showImage = src && !imgError;

  const inner = (
    <div className={classNames("relative inline-flex shrink-0", className)}>
      <div
        className={classNames(
          "flex items-center justify-center rounded-full font-medium ring-1 ring-gray-200/60 dark:ring-gray-700/60",
          sizeClasses[size],
          showImage
            ? "bg-gray-100 dark:bg-gray-800"
            : "bg-primary-100 text-primary-700 dark:bg-primary-900/50 dark:text-primary-300",
        )}
      >
        {showImage ? (
          <img
            src={src}
            alt={name ?? "avatar"}
            onError={() => setImgError(true)}
            className="h-full w-full rounded-full object-cover"
          />
        ) : name ? (
          getInitials(name)
        ) : (
          <IconUser />
        )}
      </div>

      {status && (
        <span
          className={classNames(
            "absolute bottom-0 right-0 rounded-full ring-white dark:ring-gray-950",
            statusSizeClasses[size],
            statusColorClasses[status],
          )}
        />
      )}
    </div>
  );

  return name ? <Tooltip content={name}>{inner}</Tooltip> : inner;
};

export default Avatar;
