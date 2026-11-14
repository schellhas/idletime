import type { ReactNode } from "react";

interface MessageProps {
  kind?: "info" | "error" | "success";
  children: ReactNode;
}

/** Inline status text. Errors are announced to screen readers. */
export function Message({ kind = "info", children }: MessageProps) {
  return (
    <p role={kind === "error" ? "alert" : "status"} className={`message message--${kind}`}>
      {children}
    </p>
  );
}
