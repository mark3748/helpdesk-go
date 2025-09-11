import { useState, useEffect } from 'react';
import { listKBArticles, type KBArticle } from '../api';

export default function KnowledgeBase() {
  const [items, setItems] = useState<KBArticle[]>([]);
  const [selected, setSelected] = useState<KBArticle | null>(null);

  useEffect(() => {
    listKBArticles()
      .then(setItems)
      .catch(() => setItems([]));
  }, []);

  return (
    <div className="mx-auto max-w-2xl space-y-4 p-4">
      <h2 className="text-2xl font-semibold">Knowledge Base</h2>
      <ul className="space-y-2">
        {items.map(a => (
          <li key={a.id}>
            <button
              className="text-left text-blue-600 hover:underline"
              onClick={() => setSelected(a)}
            >
              {a.title}
            </button>
          </li>
        ))}
      </ul>
      {selected && (
        <article className="space-y-2 rounded border p-4">
          <h3 className="text-xl font-semibold">{selected.title}</h3>
          <p>{selected.body_md}</p>
        </article>
      )}
    </div>
  );
}
