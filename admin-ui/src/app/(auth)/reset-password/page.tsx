import type { FC } from "react";
import AuthGuard from "@/components/guards/AuthGuard";
import ResetPasswordForm from "@/forms/ResetPasswordForm";

const ResetPasswordPage: FC = () => (
  <AuthGuard>
    <div className="flex min-h-screen items-center justify-center bg-gray-50 p-4 dark:bg-gray-950">
      <ResetPasswordForm />
    </div>
  </AuthGuard>
);

export default ResetPasswordPage;
