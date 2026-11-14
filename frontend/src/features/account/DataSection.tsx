import { useState } from "react";
import type { FormEvent } from "react";
import { useNavigate } from "react-router";

import { EXPORT_URL, deleteAccount } from "../../api/user";
import { useAsyncAction } from "../../hooks/useAsyncAction";
import { Field } from "../../ui/Field";
import { Message } from "../../ui/Message";
import { useAuth } from "../auth/AuthContext";

/** Data export and account deletion. */
export function DataSection() {
  const { clearSession } = useAuth();
  const navigate = useNavigate();

  const [confirming, setConfirming] = useState(false);
  const [password, setPassword] = useState("");
  const { pending, error, run } = useAsyncAction();

  async function handleDelete(event: FormEvent) {
    event.preventDefault();
    if (await run(() => deleteAccount(password))) {
      clearSession();
      navigate("/register", { replace: true });
    }
  }

  return (
    <section className="card stack">
      <h2>Your data</h2>

      <div className="stack">
        <p className="muted">Download everything stored about you as a JSON file.</p>
        <a className="button button--secondary" href={EXPORT_URL} download>
          Download my data
        </a>
      </div>

      <hr />

      {!confirming ? (
        <div className="stack">
          <p className="muted">
            Deleting your account permanently removes your profile and all your data. This cannot be undone.
          </p>
          <button type="button" className="button button--danger" onClick={() => setConfirming(true)}>
            Delete account…
          </button>
        </div>
      ) : (
        <form className="stack" onSubmit={handleDelete}>
          <Message kind="error">This permanently deletes your account and all your data.</Message>
          <Field
            label="Confirm with your password"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
          {error && <Message kind="error">{error}</Message>}
          <div className="button-row">
            <button type="button" className="button button--secondary" onClick={() => setConfirming(false)}>
              Cancel
            </button>
            <button type="submit" className="button button--danger" disabled={pending}>
              {pending ? "Deleting…" : "Delete everything"}
            </button>
          </div>
        </form>
      )}
    </section>
  );
}
