import { useState } from "react";
import type { FormEvent } from "react";

import { resendVerification } from "../../api/auth";
import { useAsyncAction } from "../../hooks/useAsyncAction";
import { Field } from "../../ui/Field";
import { Message } from "../../ui/Message";

interface ResendVerificationProps {
  initialEmail?: string;
}

/** Lets a user request a new confirmation email. */
export function ResendVerification({ initialEmail = "" }: ResendVerificationProps) {
  const [email, setEmail] = useState(initialEmail);
  const [sent, setSent] = useState(false);
  const { pending, error, run } = useAsyncAction();

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    setSent(await run(() => resendVerification(email)));
  }

  return (
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
      {sent && <Message kind="success">If this address needs confirming, a new email is on its way.</Message>}
      <button type="submit" className="button button--secondary" disabled={pending}>
        {pending ? "Sending…" : "Resend confirmation email"}
      </button>
    </form>
  );
}
