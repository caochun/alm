import { useState, useEffect } from 'react'
import { getState } from '../services/api'
import './StateView.css'

export default function StateView({ app, env }) {
  const [state, setState] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const load = () => {
    setLoading(true)
    setError('')
    getState(app, env)
      .then(data => setState(data))
      .catch(err => setError(err.response?.data?.error || err.message))
      .finally(() => setLoading(false))
  }

  useEffect(() => { load() }, [app, env])

  if (loading) {
    return <div className="state-view"><div className="state-loading">Loading...</div></div>
  }

  if (error) {
    return (
      <div className="state-view">
        <div className="state-error-msg">{error}</div>
        <button className="btn btn-outline" onClick={load}>Retry</button>
      </div>
    )
  }

  if (!state) return null

  const hasServices = state.services && Object.keys(state.services).length > 0
  const hasInfra = state.infra && Object.keys(state.infra).length > 0
  const hasIngress = state.ingress && Object.keys(state.ingress).length > 0
  const isEmpty = !hasServices && !hasInfra && !hasIngress

  return (
    <div className="state-view">
      <div className="state-header">
        <span className="state-title">Reported State</span>
        <button className="btn btn-outline btn-sm" onClick={load}>Refresh</button>
      </div>

      {state.updatedAt && (
        <div className="state-updated">
          Last updated: {new Date(state.updatedAt).toLocaleString()}
        </div>
      )}

      {isEmpty && (
        <div className="state-empty-msg">
          No deployment state found. Run Apply to deploy.
        </div>
      )}

      {hasServices && (
        <StateSection title="Services" items={state.services} timeField="deployedAt" fields={[
          { key: 'artifactRef', label: 'Artifact' },
        ]} />
      )}

      {hasInfra && (
        <StateSection title="Infrastructure" items={state.infra} timeField="provisionedAt" fields={[
          { key: 'outputs', label: 'Outputs', render: v => v ? Object.entries(v).map(([k,val]) => `${k}=${val}`).join(', ') : '' },
        ]} />
      )}

      {hasIngress && (
        <StateSection title="Ingress" items={state.ingress} timeField="configuredAt" fields={[]} />
      )}
    </div>
  )
}

function StateSection({ title, items, timeField, fields }) {
  const entries = Object.entries(items)
  return (
    <div className="state-section">
      <div className="section-title">{title}</div>
      <table className="state-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Status</th>
            {fields.map(f => <th key={f.key}>{f.label}</th>)}
            <th>Time</th>
          </tr>
        </thead>
        <tbody>
          {entries.map(([name, item]) => (
            <tr key={name}>
              <td className="state-name">{name}</td>
              <td>
                <span className={`status-badge status-${item.status || 'unknown'}`}>
                  {item.status || 'unknown'}
                </span>
              </td>
              {fields.map(f => (
                <td key={f.key} className="state-detail">
                  {f.render ? f.render(item[f.key]) : (item[f.key] || '—')}
                </td>
              ))}
              <td className="state-time">{timeAgo(item[timeField])}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function timeAgo(dateStr) {
  if (!dateStr) return '—'
  const d = new Date(dateStr)
  if (isNaN(d.getTime())) return '—'
  const diff = Date.now() - d.getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  return `${days}d ago`
}
