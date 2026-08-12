import { type FC, type ReactNode, createElement } from "react";
import { render, render as rtlRender } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { FormikProvider, useFormik } from "formik";

const AuthenticatedRouter: FC<{ children: ReactNode }> = ({ children }) => (
  <MemoryRouter>{children}</MemoryRouter>
);

const AllProviders: FC<{ children: ReactNode }> = ({ children }) => (
  <MemoryRouter>{children}</MemoryRouter>
);

const customRender = (
  ui: ReturnType<typeof createElement>,
  options?: Parameters<typeof rtlRender>[1],
) => render(ui, { wrapper: AllProviders, ...options });

type FormValues = Record<string, unknown>;

const renderWithFormik = <T extends FormValues>({
  initialValues,
  onSubmit = () => {},
  children,
}: {
  initialValues: T;
  onSubmit?: (values: T) => void;
  children: (formik: ReturnType<typeof useFormik<T>>) => ReturnType<typeof createElement>;
}) => {
  const formik = useFormik({ initialValues, onSubmit, validateOnChange: false });
  return render(
    <MemoryRouter>
      <FormikProvider value={formik}>{children(formik)}</FormikProvider>
    </MemoryRouter>,
  );
};

export { customRender as render, AuthenticatedRouter, AllProviders, renderWithFormik };
export * from "@testing-library/react";