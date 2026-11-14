import { useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router";

import { verifyEmail } from "../../api/auth";
import { useAsyncAction } from "../../hooks/useAsyncAction";
import { AuthCard } from "../../ui/AuthCard";
import { Message } from "../../ui/Message";
import { ResendVerification } from "./ResendVerification";

type Outcome = "verifying" | "verified" | "failed";

export function VerifyEmailPage() {
  const [params] = useSearchParams();
  const token = params.get("token") ?? "";

  const [outcome, setOutcome] = useState<Outcome>(token === "" ? "failed" : "verifying");
  const { error, run } = useAsyncAction();
  const started = useRef(false); // a token works once, so never verify twice (React dev re-runs effects)

  useEffect(() => {
    if (token === "" || started.current) {
      return;
    }
    started.current = true;
    void run(() => verifyEmail(token)).then((ok) => setOutcome(ok ? "verified" : "failed"));
  }, [token, run]);

  return (
    <AuthCard title="Confirm your email">
      {outcome === "verifying" && <Message>Confirming your email address…</Message>}
      {outcome === "verified" && (
        <>
          <Message kind="success">Your email address is confirmed. You can log in now.</Message>
          <Link to="/login" className="button">
            Log in
          </Link>
        </>
      )}
      {outcome === "failed" && (
        <>
          <Message kind="error">{error ?? "This link is invalid or has expired."}</Message>
          <p className="muted">Request a new confirmation email:</p>
          <ResendVerification />
        </>
      )}
    </AuthCard>
  );
}
