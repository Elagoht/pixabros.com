import * as Yup from "yup";

// Mirrors auth.ValidatePassword on the Go side: bcrypt caps the input at 72
// bytes, so a longer password would be silently truncated.
const PASSWORD_MIN = 8;
const PASSWORD_MAX = 72;

export const loginValidationSchema = (t: TranslateFunction) =>
  Yup.object({
    username: Yup.string().required(t("validation.username.required")),
    password: Yup.string().required(t("validation.password.required")),
  });

export const changePasswordValidationSchema = (t: TranslateFunction) =>
  Yup.object({
    current_password: Yup.string().required(t("validation.password.required")),
    new_password: Yup.string()
      .required(t("validation.password.required"))
      .min(PASSWORD_MIN, t("validation.password.min"))
      .max(PASSWORD_MAX, t("validation.password.max")),
    confirm_password: Yup.string()
      .required(t("validation.confirmPassword.required"))
      .oneOf([Yup.ref("new_password")], t("validation.confirmPassword.match")),
  });
