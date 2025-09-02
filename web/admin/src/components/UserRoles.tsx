import React, { useState } from 'react'

export default function UserRoles() {
  const [userID, setUserID] = useState('')
  const [roles, setRoles] = useState<string[]>([])
  const [newRole, setNewRole] = useState('')

  const load = async () => {
    if (!userID) return
    const res = await fetch(`/users/${userID}/roles`)
    if (res.ok) {
      setRoles(await res.json())
    }
  }

  const add = async () => {
    if (!userID || !newRole) return
    await fetch(`/users/${userID}/roles`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ role: newRole }),
    })
    setNewRole('')
    load()
  }

  const remove = async (r: string) => {
    await fetch(`/users/${userID}/roles/${r}`, { method: 'DELETE' })
    load()
  }

  return (
    <div>
      <div>
        <input
          placeholder="User ID"
          value={userID}
          onChange={(e) => setUserID(e.target.value)}
        />
        <button onClick={load}>Load</button>
      </div>
      <ul>
        {roles.map((r) => (
          <li key={r}>
            {r} <button onClick={() => remove(r)}>remove</button>
          </li>
        ))}
      </ul>
      <div>
        <input
          placeholder="Role"
          value={newRole}
          onChange={(e) => setNewRole(e.target.value)}
        />
        <button onClick={add}>Add</button>
      </div>
    </div>
  )
}
