import { useEffect, useState } from 'react'

interface Device {
  id: string
  hostname: string
  os: string
  ip: string
  enrolled_at: string
  last_seen: string
}

const API_BASE = import.meta.env.VITE_API_URL ?? ''

export default function DeviceList() {
  const [devices, setDevices] = useState<Device[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetch(`${API_BASE}/api/devices`)
      .then((r) => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        return r.json() as Promise<Device[]>
      })
      .then((data) => {
        setDevices(data ?? [])
        setLoading(false)
      })
      .catch((err: unknown) => {
        setError(err instanceof Error ? err.message : String(err))
        setLoading(false)
      })
  }, [])

  if (loading) return <p>Loading devices…</p>
  if (error) return <p style={{ color: 'red' }}>Error: {error}</p>
  if (devices.length === 0) return <p>No devices enrolled yet.</p>

  return (
    <table>
      <thead>
        <tr>
          <th>Hostname</th>
          <th>OS</th>
          <th>IP</th>
          <th>Enrolled At</th>
          <th>Last Seen</th>
        </tr>
      </thead>
      <tbody>
        {devices.map((d) => (
          <tr key={d.id}>
            <td>{d.hostname}</td>
            <td>{d.os}</td>
            <td>{d.ip}</td>
            <td>{new Date(d.enrolled_at).toLocaleString()}</td>
            <td>{new Date(d.last_seen).toLocaleString()}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}
