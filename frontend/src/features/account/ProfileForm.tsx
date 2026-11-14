import { useState } from "react";
import type { FormEvent } from "react";

import { updateUsername } from "../../api/user";
import { useAsyncAction } from "../../hooks/useAsyncAction";
import { Field } from "../../ui/Field";
import { Message } from "../../ui/Message";
import { useAuth, useCurrentUser } from "../auth/AuthContext";

export function ProfileForm() {
  const user = useCurrentUser();
  const { setUser } = useAuth();

  const [username, setUsernameInput] = useState(user.username);
  const [saved, setSaved] = useState(false);
  const { pending, error, run } = useAsyncAction();

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    setSaved(false);
    setSaved(await run(async () => setUser(await updateUsername(username))));
  }

  return (
    <form className="card stack" onSubmit={handleSubmit}>
      <h2>Profile</h2>
      <Field label="Email" value={user.email} readOnly hint="Your email address cannot be changed yet." />
      <Field
        label="Username"
        value={username}
        onChange={(e) => setUsernameInput(e.target.value)}
        minLength={3}
        maxLength={32}
        pattern="[A-Za-z0-9._\-]+"
        required
      />
      {error && <Message kind="error">{error}</Message>}
      {saved && <Message kind="success">Saved.</Message>}
      <button type="submit" className="button" disabled={pending || username === user.username}>
        {pending ? "Saving…" : "Save username"}
      </button>
    </form>
  );
}
