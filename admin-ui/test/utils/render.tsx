import { render, type render as rtlRender } from "@testing-library/react";
import { FormikProvider, useFormik } from "formik";
import type { createElement, FC, ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";

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
  children: (
    formik: ReturnType<typeof useFormik<T>>,
  ) => ReturnType<typeof createElement>;
}) => {
  const formik = useFormik({
    initialValues,
    onSubmit,
    validateOnChange: false,
  });
  return render(
    <MemoryRouter>
      <FormikProvider value={formik}>{children(formik)}</FormikProvider>
    </MemoryRouter>,
  );
};

export * from "@testing-library/react";
export {
  AllProviders,
  AuthenticatedRouter,
  customRender as render,
  renderWithFormik,
};
