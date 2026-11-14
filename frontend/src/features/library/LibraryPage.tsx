import { Message } from "../../ui/Message";
import { LibraryTree } from "./LibraryTree";
import { NewActivityForm } from "./NewActivityForm";
import { NewCategoryForm } from "./NewCategoryForm";
import { useLibrary } from "./useLibrary";

export function LibraryPage() {
  const library = useLibrary();

  return (
    <div className="page">
      <section className="card stack">
        <h2>Library</h2>
        <p className="muted">
          Multipliers say how much time something should get compared to its siblings. A category with
          ×2 gets twice the time of a sibling with ×1, no matter how many activities it holds.
        </p>
        {library.loading && <Message>Loading…</Message>}
        {library.error && <Message kind="error">{library.error}</Message>}
        {!library.loading && !library.error && <LibraryTree nodes={library.tree} library={library} />}
      </section>

      <div className="stack stack--wide">
        <NewCategoryForm categories={library.categories} onSubmit={library.createCategory} />
        <NewActivityForm categories={library.categories} onSubmit={library.createActivity} />
      </div>
    </div>
  );
}
