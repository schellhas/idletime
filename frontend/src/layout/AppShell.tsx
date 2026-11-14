import { useEffect, useState } from "react";
import { NavLink, Outlet, useLocation } from "react-router";

import { useAuth } from "../features/auth/AuthContext";

const MAIN_LINKS = [
  { to: "/", label: "Now", end: true },
  { to: "/library", label: "Library", end: false },
  { to: "/stats", label: "Progress", end: false },
  { to: "/account", label: "Account", end: false },
] as const;

const MENU_ID = "main-menu";

export function AppShell() {
  const { status, logout } = useAuth();
  const loggedIn = status === "authenticated";

  // On narrow screens the links of logged-in users live in a menu behind a burger button.
  const [menuOpen, setMenuOpen] = useState(false);
  const { pathname } = useLocation();

  useEffect(() => setMenuOpen(false), [pathname]);

  useEffect(() => {
    if (!menuOpen) {
      return;
    }
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setMenuOpen(false);
      }
    };
    document.addEventListener("keydown", closeOnEscape);
    return () => document.removeEventListener("keydown", closeOnEscape);
  }, [menuOpen]);

  return (
    <div className="shell">
      <header className="shell__header">
        <NavLink to="/" className="shell__brand">
          idletime
        </NavLink>

        {loggedIn && (
          <button
            type="button"
            className="nav__toggle"
            aria-label="Menu"
            aria-expanded={menuOpen}
            aria-controls={MENU_ID}
            onClick={() => setMenuOpen((open) => !open)}
          >
            <span className="nav__toggle-bars" aria-hidden="true" />
          </button>
        )}

        <nav
          id={MENU_ID}
          className={`nav${loggedIn ? " nav--collapsible" : ""}${menuOpen ? " nav--open" : ""}`}
          aria-label="Main"
        >
          {loggedIn && (
            <>
              {MAIN_LINKS.map((link) => (
                <NavLink key={link.to} to={link.to} end={link.end}>
                  {link.label}
                </NavLink>
              ))}
              <button type="button" className="nav__button" onClick={() => void logout()}>
                Log out
              </button>
            </>
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
    </div>
  );
}
