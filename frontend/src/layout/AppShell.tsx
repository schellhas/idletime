import { NavLink, Outlet } from "react-router";

import { useAuth } from "../features/auth/AuthContext";

const MAIN_LINKS = [
  { to: "/", label: "Now", end: true },
  { to: "/library", label: "Library", end: false },
  { to: "/stats", label: "Progress", end: false },
  { to: "/account", label: "Account", end: false },
] as const;

export function AppShell() {
  const { status, logout } = useAuth();
  const loggedIn = status === "authenticated";

  return (
    <div className={`shell${loggedIn ? " shell--with-tabbar" : ""}`}>
      <header className="shell__header">
        <NavLink to="/" className="shell__brand">
          idletime
        </NavLink>

        {/* Top navigation: wide screens. */}
        <nav className="nav nav--top" aria-label="Main">
          {loggedIn &&
            MAIN_LINKS.map((link) => (
              <NavLink key={link.to} to={link.to} end={link.end}>
                {link.label}
              </NavLink>
            ))}
          {loggedIn && (
            <button type="button" className="nav__button" onClick={() => void logout()}>
              Log out
            </button>
          )}
          {status === "anonymous" && (
            <>
              <NavLink to="/login">Log in</NavLink>
              <NavLink to="/register">Sign up</NavLink>
            </>
          )}
        </nav>
      </header>

      <main className="shell__main">
        <Outlet />
      </main>

      {/* Bottom tab bar: phones, where it is within thumb reach. */}
      {loggedIn && (
        <nav className="tabbar" aria-label="Main">
          {MAIN_LINKS.map((link) => (
            <NavLink key={link.to} to={link.to} end={link.end}>
              {link.label}
            </NavLink>
          ))}
        </nav>
      )}
    </div>
  );
}
