import { BrowserRouter, Navigate, Route, Routes } from "react-router";

import { AccountPage } from "./features/account/AccountPage";
import { AuthProvider } from "./features/auth/AuthContext";
import { ForgotPasswordPage } from "./features/auth/ForgotPasswordPage";
import { LoginPage } from "./features/auth/LoginPage";
import { RegisterPage } from "./features/auth/RegisterPage";
import { ResetPasswordPage } from "./features/auth/ResetPasswordPage";
import { PublicOnly, RequireAuth } from "./features/auth/RouteGuards";
import { VerifyEmailPage } from "./features/auth/VerifyEmailPage";
import { LibraryPage } from "./features/library/LibraryPage";
import { NowPage } from "./features/now/NowPage";
import { StatsPage } from "./features/stats/StatsPage";
import { AppShell } from "./layout/AppShell";

export function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route element={<AppShell />}>
            <Route element={<RequireAuth />}>
              <Route index element={<NowPage />} />
              <Route path="library" element={<LibraryPage />} />
              <Route path="stats" element={<StatsPage />} />
              <Route path="account" element={<AccountPage />} />
            </Route>

            <Route element={<PublicOnly />}>
              <Route path="login" element={<LoginPage />} />
              <Route path="register" element={<RegisterPage />} />
              <Route path="forgot-password" element={<ForgotPasswordPage />} />
            </Route>

            {/* Reachable whether or not someone is logged in: the links come from emails. */}
            <Route path="verify-email" element={<VerifyEmailPage />} />
            <Route path="reset-password" element={<ResetPasswordPage />} />

            <Route path="*" element={<Navigate to="/" replace />} />
          </Route>
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  );
}
