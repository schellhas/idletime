import { useState } from "react";
import type { FormEvent } from "react";
import { Link } from "react-router";

import { forgotPassword } from "../../api/auth";
import { useAsyncAction } from "../../hooks/useAsyncAction";
import { AuthCard } from "../../ui/AuthCard";
import { Field } from "../../ui/Field";
import { Message } from "../../ui/Message";

export function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [sent, setSent] = useState(false);
  const { pending, error, run } = useAsyncAction();

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    setSent(await run(() => forgotPassword(email)));
  }

  return (
    <AuthCard title="Reset your password">
      <form className="stack" onSubmit={handleSubmit}>
        <Field
          label="Email"
          type="email"
          autoComplete="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
        />
        {error && <Message kind="error">{error}</Message>}
        {sent && (
          <Message kind="success">If an account exists for this address, a reset link is on its way.</Message>
        )}
        <button type="submit" className="button" disabled={pending}>
          {pending ? "Sending…" : "Send reset link"}
        </button>
      </form>
      <p className="muted">
        <Link to="/login">Back to log in</Link>
      </p>
    </AuthCard>
  );
}
