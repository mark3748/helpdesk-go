import { useState } from 'react'

export default function TestConnection() {
  const [logPath, setLogPath] = useState('')

  const handleClick = async () => {
    const res = await fetch('/test-connection', {
      method: 'POST',
      credentials: 'include',
    })
    const data = await res.json()
    setLogPath(data.log_path)
  }

  return (
    <div>
      <button onClick={handleClick}>Test Connection</button>
      {logPath && <p>Logs written to {logPath}</p>}
    </div>
  )
}
