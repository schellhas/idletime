import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";

import * as authApi from "../../api/auth";
import { onUnauthenticated } from "../../api/http";
import { getCurrentUser } from "../../api/user";
import type { User } from "../../api/user";

type Status = "loading" | "authenticated" | "anonymous";

interface AuthContextValue {
  status: Status;
  user: User | null;
  login: (identifier: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  /** Replaces the cached user, e.g. after the profile was edited. */
  setUser: (user: User) => void;
  /** Forgets the session locally, e.g. after the account was deleted. */
  clearSession: () => void;
}

interface AuthState {
  status: Status;
  user: User | null;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>({ status: "loading", user: null });

  const clearSession = useCallback(() => setState({ status: "anonymous", user: null }), []);
  const setUser = useCallback((user: User) => setState({ status: "authenticated", user }), []);

  // Restore the session on page load.
  useEffect(() => {
    getCurrentUser().then(setUser, clearSession);
  }, [setUser, clearSession]);

  // Drop the local session as soon as any request reports it expired.
  useEffect(() => onUnauthenticated(clearSession), [clearSession]);

  const login = useCallback(
    async (identifier: string, password: string) => setUser(await authApi.login(identifier, password)),
    [setUser],
  );

  const logout = useCallback(async () => {
    try {
      await authApi.logout();
    } finally {
      clearSession();
    }
  }, [clearSession]);

  const value = useMemo<AuthContextValue>(
    () => ({ ...state, login, logout, setUser, clearSession }),
    [state, login, logout, setUser, clearSession],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used inside <AuthProvider>");
  }
  return context;
}

/** Returns the logged-in user. Only call it below <RequireAuth>. */
export function useCurrentUser(): User {
  const { user } = useAuth();
  if (!user) {
    throw new Error("useCurrentUser requires an authenticated user");
  }
  return user;
}
