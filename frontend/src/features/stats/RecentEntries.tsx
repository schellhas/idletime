import { useCallback, useEffect, useState } from "react";

import type { Activity } from "../../api/activities";
import { errorMessage } from "../../api/http";
import { deleteTimeEntry, listTimeEntries } from "../../api/timeEntries";
import type { TimeEntry } from "../../api/timeEntries";
import { formatDateTime, formatMinutes } from "../../lib/format";
import { ConfirmButton } from "../../ui/ConfirmButton";
import { Message } from "../../ui/Message";

const ENTRY_LIMIT = 20;

interface RecentEntriesProps {
  activities: Activity[];
  /** Called after an entry was deleted so totals and charts can refresh. */
  onChanged: () => void;
  /** Changes whenever time was logged elsewhere, to reload the list. */
  version: number;
}

/** The latest time entries, each deletable. */
export function RecentEntries({ activities, onChanged, version }: RecentEntriesProps) {
  const [entries, setEntries] = useState<TimeEntry[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      setEntries(await listTimeEntries(ENTRY_LIMIT));
      setError(null);
    } catch (e) {
      setError(errorMessage(e));
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load, version]);

  async function remove(id: number) {
    try {
      await deleteTimeEntry(id);
      await load();
      onChanged();
    } catch (e) {
      setError(errorMessage(e));
    }
  }

  const nameOf = (id: number) => activities.find((a) => a.id === id)?.name ?? "Deleted activity";

  return (
    <section className="card stack">
      <h2>Recent entries</h2>
      {error && <Message kind="error">{error}</Message>}
      {entries?.length === 0 && <Message>No time logged yet.</Message>}
      <ul className="entries">
        {entries?.map((entry) => (
          <li key={entry.id} className="entries__row">
            <div className="entries__main">
              <span>{nameOf(entry.activityId)}</span>
              <span className="muted">
                {formatDateTime(entry.loggedAt)}
                {entry.note && ` · ${entry.note}`}
              </span>
            </div>
            <span className="badge">{formatMinutes(entry.minutes)}</span>
            <ConfirmButton onConfirm={() => void remove(entry.id)}>Delete</ConfirmButton>
          </li>
        ))}
      </ul>
    </section>
  );
}
