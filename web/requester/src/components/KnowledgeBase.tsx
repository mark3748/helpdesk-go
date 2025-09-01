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
    <div>
      <h2>Knowledge Base</h2>
      <ul>
        {articles.map(a => (
          <li key={a.id}>
            <button onClick={() => setSelected(a)}>{a.title}</button>
          </li>
        ))}
      </ul>
      {selected && (
        <article>
          <h3>{selected.title}</h3>
          <p>{selected.body}</p>
        </article>
      )}
    </div>
  );
}
