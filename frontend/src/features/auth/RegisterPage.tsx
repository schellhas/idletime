import { useState } from "react";
import type { FormEvent } from "react";
import { Link } from "react-router";

import { register } from "../../api/auth";
import { useAsyncAction } from "../../hooks/useAsyncAction";
import { AuthCard } from "../../ui/AuthCard";
import { Field } from "../../ui/Field";
import { Message } from "../../ui/Message";
import { ResendVerification } from "./ResendVerification";

export const MIN_PASSWORD_LENGTH = 10;

export function RegisterPage() {
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [registeredEmail, setRegisteredEmail] = useState<string | null>(null);
  const { pending, error, run } = useAsyncAction();

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    if (await run(() => register(username, email, password))) {
      setRegisteredEmail(email);
    }
  }

  if (registeredEmail !== null) {
    return (
      <AuthCard title="Check your email">
        <p>
          We sent a confirmation link to <strong>{registeredEmail}</strong>. Open it to activate your
          account, then log in.
        </p>
        <p className="muted">Nothing arrived? Check your spam folder or request a new link:</p>
        <ResendVerification initialEmail={registeredEmail} />
        <p className="muted">
          <Link to="/login">Back to log in</Link>
        </p>
      </AuthCard>
    );
  }

  return (
    <AuthCard title="Sign up">
      <form className="stack" onSubmit={handleSubmit}>
        <Field
          label="Username"
          autoComplete="username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          minLength={3}
          maxLength={32}
          pattern="[A-Za-z0-9._\-]+"
          hint="3-32 characters: letters, digits, . _ -"
          required
        />
        <Field
          label="Email"
          type="email"
          autoComplete="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
        />
        <Field
          label="Password"
          type="password"
          autoComplete="new-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          minLength={MIN_PASSWORD_LENGTH}
          maxLength={72}
          hint={`At least ${MIN_PASSWORD_LENGTH} characters`}
          required
        />
        {error && <Message kind="error">{error}</Message>}
        <button type="submit" className="button" disabled={pending}>
          {pending ? "Creating account…" : "Create account"}
        </button>
      </form>
      <p className="muted">
        Already registered? <Link to="/login">Log in</Link>
      </p>
    </AuthCard>
  );
}
