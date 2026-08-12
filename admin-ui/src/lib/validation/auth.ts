import * as Yup from "yup";

export const loginValidationSchema = (t: TranslateFunction) =>
	Yup.object({
		email: Yup.string().required(t("validation.email.required")),
		password: Yup.string().required(t("validation.password.required")),
	});
