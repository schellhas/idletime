import { NavLink, Outlet } from "react-router";

import { useAuth } from "../features/auth/AuthContext";

export function AppShell() {
  const { status, user, logout } = useAuth();

  return (
    <div className="shell">
      <header className="shell__header">
        <NavLink to="/" className="shell__brand">
          idletime
        </NavLink>

        <nav className="nav" aria-label="Main">
          {status === "authenticated" && user ? (
            <>
              <NavLink to="/" end>
                Library
              </NavLink>
              <NavLink to="/account">{user.username}</NavLink>
              <button type="button" className="nav__button" onClick={() => void logout()}>
                Log out
              </button>
            </>
          ) : (
            status === "anonymous" && (
              <>
                <NavLink to="/login">Log in</NavLink>
                <NavLink to="/register">Sign up</NavLink>
              </>
            )
          )}
        </nav>
      </header>
      <main className="shell__main">
        <Outlet />
      </main>
    </div>
  );
}
