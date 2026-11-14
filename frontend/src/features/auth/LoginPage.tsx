import { useState } from "react";
import type { FormEvent } from "react";
import { Link, useLocation, useNavigate } from "react-router";

import { useAsyncAction } from "../../hooks/useAsyncAction";
import { AuthCard } from "../../ui/AuthCard";
import { Field } from "../../ui/Field";
import { Message } from "../../ui/Message";
import { useAuth } from "./AuthContext";
import { ResendVerification } from "./ResendVerification";

interface LocationState {
  from?: string;
  notice?: string;
}

export function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const state = (useLocation().state ?? {}) as LocationState;

  const [identifier, setIdentifier] = useState("");
  const [password, setPassword] = useState("");
  const { pending, error, errorCode, run } = useAsyncAction();

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    if (await run(() => login(identifier, password))) {
      navigate(state.from ?? "/", { replace: true });
    }
  }

  return (
    <AuthCard title="Log in">
      {state.notice && <Message kind="success">{state.notice}</Message>}

      <form className="stack" onSubmit={handleSubmit}>
        <Field
          label="Username or email"
          autoComplete="username"
          value={identifier}
          onChange={(e) => setIdentifier(e.target.value)}
          required
        />
        <Field
          label="Password"
          type="password"
          autoComplete="current-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
        />
        {error && <Message kind="error">{error}</Message>}
        <button type="submit" className="button" disabled={pending}>
          {pending ? "Logging in…" : "Log in"}
        </button>
      </form>

      {errorCode === "email_not_verified" && (
        <ResendVerification initialEmail={identifier.includes("@") ? identifier : ""} />
      )}

      <p className="muted">
        <Link to="/forgot-password">Forgot your password?</Link>
      </p>
      <p className="muted">
        No account yet? <Link to="/register">Sign up</Link>
      </p>
    </AuthCard>
  );
}
