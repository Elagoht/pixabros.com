import { IconLoader2 } from "@tabler/icons-react";
import classNames from "classnames";
import { useFormikContext } from "formik";
import { type ComponentPropsWithoutRef, forwardRef } from "react";

import Button from "./Button";

type SubmitButtonProps = Omit<
  ComponentPropsWithoutRef<typeof Button>,
  "type"
> & {
  loadingText?: string;
};

const LoadingIcon = ((props: IconProps) => (
  <IconLoader2
    {...props}
    className={classNames("animate-spin", props.className)}
  />
)) as IconElement;

const SubmitButton = forwardRef<HTMLButtonElement, SubmitButtonProps>(
  ({ children, loadingText, disabled, ...props }, ref) => {
    const { isSubmitting } = useFormikContext();

    return (
      <Button
        ref={ref}
        type="submit"
        disabled={isSubmitting || disabled}
        rightIcon={isSubmitting ? LoadingIcon : undefined}
        {...props}
      >
        {isSubmitting && loadingText ? loadingText : children}
      </Button>
    );
  },
);

export default SubmitButton;
