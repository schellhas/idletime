import { useState } from "react";

import type { Activity } from "../../api/activities";
import { formatMinutes } from "../../lib/format";
import { ConfirmButton } from "../../ui/ConfirmButton";
import { Message } from "../../ui/Message";
import { EditActivityForm } from "./EditActivityForm";
import { EditCategoryForm } from "./EditCategoryForm";
import type { LibraryNode } from "./libraryTree";
import type { Library } from "./useLibrary";

interface LibraryTreeProps {
  nodes: LibraryNode[];
  library: Library;
}

/** The user's categories and activities as an editable tree. */
export function LibraryTree({ nodes, library }: LibraryTreeProps) {
  // Only one row is edited at a time, identified as "category:3" or "activity:7".
  const [editing, setEditing] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  if (nodes.length === 0) {
    return <Message>No categories yet. Add your first one.</Message>;
  }

  async function remove(action: () => Promise<void>) {
    setActionError(null);
    try {
      await action();
    } catch (e) {
      setActionError(e instanceof Error ? e.message : "Something went wrong.");
    }
  }

  const renderActivity = (activity: Activity) => {
    const key = `activity:${activity.id}`;
    if (editing === key) {
      return (
        <li key={key}>
          <EditActivityForm
            activity={activity}
            categories={library.categories}
            onSave={(patch) => library.updateActivity(activity.id, patch)}
            onCancel={() => setEditing(null)}
          />
        </li>
      );
    }
    return (
      <li key={key} className="tree__row tree__row--activity">
        <div className="tree__main">
          <span>{activity.name}</span>
          <span className="badge">×{activity.multiplier}</span>
          {activity.minimumMinutes > 0 && <span className="badge">min {activity.minimumMinutes} min</span>}
          <span className="badge badge--soft">{formatMinutes(activity.trackedMinutes)} tracked</span>
        </div>
        <div className="tree__actions">
          <button type="button" className="link-button" onClick={() => setEditing(key)}>
            Edit
          </button>
          <ConfirmButton onConfirm={() => void remove(() => library.removeActivity(activity.id))}>Delete</ConfirmButton>
        </div>
      </li>
    );
  };

  const renderNode = (node: LibraryNode) => {
    const key = `category:${node.category.id}`;
    return (
      <li key={key}>
        {editing === key ? (
          <EditCategoryForm
            category={node.category}
            categories={library.categories}
            onSave={(patch) => library.updateCategory(node.category.id, patch)}
            onCancel={() => setEditing(null)}
          />
        ) : (
          <div className="tree__row tree__row--category">
            <div className="tree__main">
              <strong>{node.category.name}</strong>
              <span className="badge">×{node.category.multiplier}</span>
            </div>
            <div className="tree__actions">
              <button type="button" className="link-button" onClick={() => setEditing(key)}>
                Edit
              </button>
              <ConfirmButton
                confirmLabel="Delete all?"
                onConfirm={() => void remove(() => library.removeCategory(node.category.id))}
              >
                Delete
              </ConfirmButton>
            </div>
          </div>
        )}
        {(node.children.length > 0 || node.activities.length > 0) && (
          <ul className="tree">
            {node.activities.map(renderActivity)}
            {node.children.map(renderNode)}
          </ul>
        )}
      </li>
    );
  };

  return (
    <>
      {actionError && <Message kind="error">{actionError}</Message>}
      <ul className="tree tree--root">{nodes.map(renderNode)}</ul>
    </>
  );
}
