import { useState } from 'react';

interface Article {
  id: number;
  title: string;
  body: string;
}

const articles: Article[] = [
  { id: 1, title: 'Reset Password', body: 'Visit the password portal and follow instructions.' },
  { id: 2, title: 'VPN Setup', body: 'Download the VPN client and login with your company credentials.' },
  { id: 3, title: 'Email Forwarding', body: 'In settings, add forwarding address and confirm.' },
];

export default function KnowledgeBase() {
  const [selected, setSelected] = useState<Article | null>(null);
  return (
    <div className="mx-auto max-w-2xl space-y-4 p-4">
      <h2 className="text-2xl font-semibold">Knowledge Base</h2>
      <ul className="space-y-2">
        {articles.map(a => (
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
          <p>{selected.body}</p>
        </article>
      )}
    </div>
  );
}
