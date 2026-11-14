import { useState } from "react";
import type { FormEvent } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";

import { resetPassword } from "../../api/auth";
import { useAsyncAction } from "../../hooks/useAsyncAction";
import { AuthCard } from "../../ui/AuthCard";
import { Field } from "../../ui/Field";
import { Message } from "../../ui/Message";
import { MIN_PASSWORD_LENGTH } from "./RegisterPage";

export function ResetPasswordPage() {
  const token = useSearchParams()[0].get("token") ?? "";
  const navigate = useNavigate();

  const [password, setPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [mismatch, setMismatch] = useState(false);
  const { pending, error, run } = useAsyncAction();

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    setMismatch(password !== confirmation);
    if (password !== confirmation) {
      return;
    }
    if (await run(() => resetPassword(token, password))) {
      navigate("/login", { replace: true, state: { notice: "Password changed. You can log in now." } });
    }
  }

  if (token === "") {
    return (
      <AuthCard title="Choose a new password">
        <Message kind="error">This link is invalid. Please request a new one.</Message>
        <Link to="/forgot-password">Request a reset link</Link>
      </AuthCard>
    );
  }

  return (
    <AuthCard title="Choose a new password">
      <form className="stack" onSubmit={handleSubmit}>
        <Field
          label="New password"
          type="password"
          autoComplete="new-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          minLength={MIN_PASSWORD_LENGTH}
          maxLength={72}
          hint={`At least ${MIN_PASSWORD_LENGTH} characters`}
          required
        />
        <Field
          label="Repeat new password"
          type="password"
          autoComplete="new-password"
          value={confirmation}
          onChange={(e) => setConfirmation(e.target.value)}
          required
        />
        {mismatch && <Message kind="error">The passwords do not match.</Message>}
        {error && <Message kind="error">{error}</Message>}
        <button type="submit" className="button" disabled={pending}>
          {pending ? "Saving…" : "Change password"}
        </button>
      </form>
    </AuthCard>
  );
}
