import classNames from "classnames";
import { type ComponentPropsWithoutRef, forwardRef } from "react";
import { Link } from "react-router-dom";

type ButtonVariant =
  | "default"
  | "primary"
  | "secondary"
  | "ghost"
  | "destructive"
  | "outline"
  | "success"
  | "warning";
type ButtonSize = "sm" | "md" | "lg";

type ButtonAlign = "left" | "center" | "right";

interface ButtonProps extends Omit<ComponentPropsWithoutRef<"button">, "type"> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  align?: ButtonAlign;
  leftIcon?: IconElement;
  rightIcon?: IconElement;
  type?: "button" | "submit" | "reset";
  /**
   * Renders the button as a router link. A navigating control should be a real
   * anchor: middle-click, ctrl-click and "copy link address" all work, and
   * assistive tech announces it as a link rather than a button.
   *
   * Ignored when `disabled` is set -- an anchor has no disabled state, so a
   * disabled link would still navigate.
   */
  to?: string;
}

const variantClasses: Record<ButtonVariant, string> = {
  default:
    "bg-primary-600 text-white hover:bg-primary-500 hover:scale-[1.02] hover:ring-primary-400/55 active:scale-[0.97] active:bg-primary-700 active:ring-primary-700/65 focus-visible:ring-primary-500",
  primary:
    "bg-blue-600 text-white hover:bg-blue-500 hover:scale-[1.02] hover:ring-blue-400/55 active:scale-[0.97] active:bg-blue-700 active:ring-blue-700/65 focus-visible:ring-blue-500",
  secondary:
    "bg-secondary-600 text-white hover:bg-secondary-500 hover:scale-[1.02] hover:ring-secondary-400/55 active:scale-[0.97] active:bg-secondary-700 active:ring-secondary-700/65 focus-visible:ring-secondary-500",
  ghost:
    "bg-transparent text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 hover:!ring-0 hover:!ring-offset-0 active:bg-gray-200 dark:active:bg-gray-700 active:!ring-0 active:!ring-offset-0 focus-visible:ring-gray-400",
  destructive:
    "bg-red-600 text-white hover:bg-red-500 hover:scale-[1.02] hover:ring-red-400/55 active:scale-[0.97] active:bg-red-700 active:ring-red-700/65 focus-visible:ring-red-500",
  outline:
    "bg-transparent border border-gray-300 text-gray-700 dark:border-gray-600 dark:text-gray-300 hover:bg-gray-50 hover:border-gray-400 dark:hover:bg-gray-800 dark:hover:border-gray-500 hover:ring-gray-400/40 active:bg-gray-100 dark:active:bg-gray-700 active:ring-gray-500/50 focus-visible:ring-gray-400",
  success:
    "bg-green-600 text-white hover:bg-green-500 hover:scale-[1.02] hover:ring-green-400/55 active:scale-[0.97] active:bg-green-700 active:ring-green-700/65 focus-visible:ring-green-500",
  warning:
    "bg-amber-600 text-white hover:bg-amber-500 hover:scale-[1.02] hover:ring-amber-400/55 active:scale-[0.97] active:bg-amber-700 active:ring-amber-700/65 focus-visible:ring-amber-500",
};

const sizeClasses: Record<ButtonSize, string> = {
  sm: "h-8 px-2 text-xs",
  md: "h-9 px-3 text-sm",
  lg: "h-10 px-3.5 text-sm",
};

const gapClasses: Record<ButtonSize, string> = {
  sm: "gap-1.5",
  md: "gap-2",
  lg: "gap-2",
};

const iconSizes: Record<ButtonSize, number> = {
  sm: 14,
  md: 16,
  lg: 18,
};

const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      variant = "default",
      size = "md",
      align = "center",
      leftIcon: LeftIcon,
      rightIcon: RightIcon,
      className,
      type = "button",
      to,
      children,
      ...props
    },
    ref,
  ) => {
    const classes = classNames(
      "relative overflow-hidden inline-flex items-center rounded-lg font-medium transition-all duration-150 ease-out",
      align === "left" && "justify-start",
      align === "center" && "justify-center",
      align === "right" && "justify-end",
      "ring-offset-white dark:ring-offset-gray-950",
      "hover:ring-2 hover:ring-offset-2",
      "active:ring-2 active:ring-offset-1",
      "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2",
      "disabled:pointer-events-none disabled:opacity-50",
      variantClasses[variant],
      sizeClasses[size],
      className,
    );

    const iconSize = iconSizes[size];

    const content = (
      <span
        className={classNames(
          "relative inline-flex items-center",
          gapClasses[size],
        )}
      >
        {LeftIcon && <LeftIcon size={iconSize} className="shrink-0" />}
        {children}
        {RightIcon && <RightIcon size={iconSize} className="shrink-0" />}
      </span>
    );

    if (to && !props.disabled) {
      // The remaining props are typed for <button>, so their event handlers
      // name HTMLButtonElement while an anchor's name HTMLAnchorElement. The
      // handlers are the same functions either way -- only the element type in
      // the signature differs -- so this is re-typing, not a change of shape.
      const anchorProps = props as unknown as ComponentPropsWithoutRef<"a">;
      return (
        <Link {...anchorProps} to={to} className={classes}>
          {content}
        </Link>
      );
    }

    return (
      <button {...props} ref={ref} type={type} className={classes}>
        {content}
      </button>
    );
  },
);

export default Button;
