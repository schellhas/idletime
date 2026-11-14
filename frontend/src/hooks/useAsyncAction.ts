import { useCallback, useState } from "react";

import { ApiError, errorMessage } from "../api/http";

interface AsyncActionState {
  /** True while an action is running. */
  pending: boolean;
  /** Message of the last failure, or null. */
  error: string | null;
  /** API error code of the last failure (e.g. "email_not_verified"), if it came from the API. */
  errorCode: string | null;
  /** Runs action, tracking pending and error state. Resolves to true on success. */
  run: (action: () => Promise<unknown>) => Promise<boolean>;
}

/** Shared pending/error handling for form submissions and button actions. */
export function useAsyncAction(): AsyncActionState {
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [errorCode, setErrorCode] = useState<string | null>(null);

  const run = useCallback(async (action: () => Promise<unknown>) => {
    setPending(true);
    setError(null);
    setErrorCode(null);
    try {
      await action();
      return true;
    } catch (e) {
      setError(errorMessage(e));
      setErrorCode(e instanceof ApiError ? e.code : null);
      return false;
    } finally {
      setPending(false);
    }
  }, []);

  return { pending, error, errorCode, run };
}
