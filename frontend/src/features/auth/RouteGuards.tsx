import { Navigate, Outlet, useLocation } from "react-router";

import { Message } from "../../ui/Message";
import { useAuth } from "./AuthContext";

/** Renders child routes only for logged-in users; everyone else is sent to the login. */
export function RequireAuth() {
  const { status } = useAuth();
  const location = useLocation();

  if (status === "loading") {
    return <Message>Loading…</Message>;
  }
  if (status === "anonymous") {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  }
  return <Outlet />;
}

/** Renders child routes only for logged-out users (login, sign-up, ...). */
export function PublicOnly() {
  const { status } = useAuth();

  if (status === "loading") {
    return <Message>Loading…</Message>;
  }
  if (status === "authenticated") {
    return <Navigate to="/" replace />;
  }
  return <Outlet />;
}
