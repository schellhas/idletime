import { useEffect, useState } from "react";
import type { ReactNode } from "react";

const CONFIRM_WINDOW_MS = 4000;

interface ConfirmButtonProps {
  onConfirm: () => void;
  children: ReactNode;
  /** Label shown after the first click. */
  confirmLabel?: string;
  className?: string;
  disabled?: boolean;
}

/** A button for destructive actions: the first click arms it, a second click confirms. */
export function ConfirmButton({
  onConfirm,
  children,
  confirmLabel = "Sure?",
  className = "link-button link-button--danger",
  disabled,
}: ConfirmButtonProps) {
  const [armed, setArmed] = useState(false);

  // Disarm automatically so a stray later click cannot delete something.
  useEffect(() => {
    if (!armed) {
      return;
    }
    const timer = setTimeout(() => setArmed(false), CONFIRM_WINDOW_MS);
    return () => clearTimeout(timer);
  }, [armed]);

  return (
    <button
      type="button"
      className={className}
      disabled={disabled}
      onClick={() => {
        if (armed) {
          setArmed(false);
          onConfirm();
        } else {
          setArmed(true);
        }
      }}
    >
      {armed ? confirmLabel : children}
    </button>
  );
}
