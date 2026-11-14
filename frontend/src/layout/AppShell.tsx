import type { ReactNode } from "react";

interface AppShellProps {
  children: ReactNode;
}

export function AppShell({ children }: AppShellProps) {
  return (
    <div className="shell">
      <header className="shell__header">
        <span className="shell__brand">idletime</span>
      </header>
      <main className="shell__main">{children}</main>
    </div>
  );
}
