import { useState } from "react";

import type { TimeEntry } from "../../api/timeEntries";
import { formatMinutes } from "../../lib/format";
import { Message } from "../../ui/Message";
import { useLibrary } from "../library/useLibrary";
import { SuggestionPanel } from "./SuggestionPanel";
import { TimerCard } from "./TimerCard";
import { useTimer } from "./useTimer";

/** Home page: shows the running timer, or suggests what to do next. */
export function NowPage() {
  const library = useLibrary();
  const { timer, loading, error, start, stop, cancel } = useTimer();

  const [notice, setNotice] = useState<string | null>(null);
  // Incremented whenever totals changed, which makes the suggestion panel ask again.
  const [refreshSignal, setRefreshSignal] = useState(0);

  const activityName = (id: number) => library.activities.find((a) => a.id === id)?.name ?? "activity";

  function logged(minutes: number, name: string) {
    setNotice(`Saved ${formatMinutes(minutes)} for ${name}.`);
    setRefreshSignal((n) => n + 1);
    void library.reload();
  }

  if (loading || library.loading) {
    return <Message>Loading…</Message>;
  }
  if (error || library.error) {
    return <Message kind="error">{error ?? library.error}</Message>;
  }

  return (
    <div className="now stack stack--wide">
      {notice && <Message kind="success">{notice}</Message>}

      {timer ? (
        <TimerCard
          timer={timer}
          activity={library.activities.find((a) => a.id === timer.activityId)}
          onStop={stop}
          onCancel={cancel}
          onStopped={(entry: TimeEntry) => logged(entry.minutes, activityName(entry.activityId))}
        />
      ) : (
        <SuggestionPanel
          categories={library.categories}
          refreshSignal={refreshSignal}
          onLogged={logged}
          onStart={async (activityId) => {
            setNotice(null);
            await start(activityId);
          }}
        />
      )}
    </div>
  );
}
