import type { Activity } from "../../api/activities";
import type { Timer } from "../../api/timer";
import type { TimeEntry } from "../../api/timeEntries";
import { useAsyncAction } from "../../hooks/useAsyncAction";
import { formatClock } from "../../lib/format";
import { ConfirmButton } from "../../ui/ConfirmButton";
import { Message } from "../../ui/Message";
import { useElapsedSeconds } from "./useTimer";

interface TimerCardProps {
  timer: Timer;
  /** Looked up from the library; may be missing while it loads. */
  activity: Activity | undefined;
  onStop: () => Promise<TimeEntry>;
  onCancel: () => Promise<void>;
  onStopped: (entry: TimeEntry) => void;
}

export function TimerCard({ timer, activity, onStop, onCancel, onStopped }: TimerCardProps) {
  const elapsed = useElapsedSeconds(timer.startedAt);
  const { pending, error, run } = useAsyncAction();

  return (
    <section className="card stack timer-card" aria-live="polite">
      <p className="muted">Now doing</p>
      <h2 className="timer-card__activity">{activity?.name ?? "…"}</h2>
      <p className="timer-card__clock" aria-label="Elapsed time">
        {formatClock(elapsed)}
      </p>
      {error && <Message kind="error">{error}</Message>}
      <div className="button-row">
        <ConfirmButton
          className="button button--secondary"
          confirmLabel="Discard time?"
          disabled={pending}
          onConfirm={() => void run(onCancel)}
        >
          Discard
        </ConfirmButton>
        <button
          type="button"
          className="button"
          disabled={pending}
          onClick={() => void run(async () => onStopped(await onStop()))}
        >
          {pending ? "Saving…" : "Stop & save"}
        </button>
      </div>
    </section>
  );
}
