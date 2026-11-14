import type { ReactNode } from "react";

interface AuthCardProps {
  title: string;
  children: ReactNode;
}

/** Centered card used by all public (logged-out) pages. */
export function AuthCard({ title, children }: AuthCardProps) {
  return (
    <section className="card auth-card">
      <h1>{title}</h1>
      <div className="stack">{children}</div>
    </section>
  );
}
