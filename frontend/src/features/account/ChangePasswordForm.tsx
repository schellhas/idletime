import { useState } from "react";
import type { FormEvent } from "react";

import { changePassword } from "../../api/user";
import { useAsyncAction } from "../../hooks/useAsyncAction";
import { Field } from "../../ui/Field";
import { Message } from "../../ui/Message";
import { MIN_PASSWORD_LENGTH } from "../auth/RegisterPage";

export function ChangePasswordForm() {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [changed, setChanged] = useState(false);
  const { pending, error, run } = useAsyncAction();

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    setChanged(false);
    if (await run(() => changePassword(current, next))) {
      setChanged(true);
      setCurrent("");
      setNext("");
    }
  }

  return (
    <form className="card stack" onSubmit={handleSubmit}>
      <h2>Change password</h2>
      <Field
        label="Current password"
        type="password"
        autoComplete="current-password"
        value={current}
        onChange={(e) => setCurrent(e.target.value)}
        required
      />
      <Field
        label="New password"
        type="password"
        autoComplete="new-password"
        value={next}
        onChange={(e) => setNext(e.target.value)}
        minLength={MIN_PASSWORD_LENGTH}
        maxLength={72}
        hint={`At least ${MIN_PASSWORD_LENGTH} characters. Other devices will be logged out.`}
        required
      />
      {error && <Message kind="error">{error}</Message>}
      {changed && <Message kind="success">Password changed.</Message>}
      <button type="submit" className="button" disabled={pending}>
        {pending ? "Saving…" : "Change password"}
      </button>
    </form>
  );
}
