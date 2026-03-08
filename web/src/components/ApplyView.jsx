import { useState } from 'react'
import { postApply } from '../services/api'
import './ApplyView.css'

const STATUS_COLOR = {
  pending:   '#d1d5db',
  running:   '#f59e0b',
  succeeded: '#22c55e',
  failed:    '#ef4444',
  skipped:   '#e5e7eb',
}

const STATUS_TAG = {
  pending:   '    ',
  running:   ' .. ',
  succeeded: ' ok ',
  failed:    'FAIL',
  skipped:   'skip',
}

export default function ApplyView({ app, env, plan, onDone }) {
  const [result, setResult] = useState(null)
  const [executing, setExecuting] = useState(false)
  const [error, setError] = useState('')
  const [expandedStep, setExpandedStep] = useState(null)

  const data = result || plan

  const startApply = () => {
    setExecuting(true)
    setError('')
    postApply(app, env, false, false)
      .then(data => setResult(data))
      .catch(err => {
        const resp = err.response?.data
        if (resp?.plan) setResult(resp.plan)
        setError(resp?.error || err.message)
      })
      .finally(() => setExecuting(false))
  }

  if (!data) {
    return (
      <div className="apply-view">
        <div className="apply-empty">No plan available. Generate a plan first.</div>
      </div>
    )
  }

  const phaseStats = data.phases.map(phase => {
    let done = 0, total = phase.steps.length
    for (const s of phase.steps) {
      if (s.status === 'succeeded' || s.status === 'skipped') done++
    }
    return { name: phase.name, done, total }
  })

  return (
    <div className="apply-view">
      {/* Phase progress bars */}
      <div className="apply-progress">
        {phaseStats.map(p => (
          <div key={p.name} className="progress-item">
            <div className="progress-label">
              <span className="progress-name">{p.name}</span>
              <span className="progress-count">{p.done}/{p.total}</span>
            </div>
            <div className="progress-bar">
              <div
                className="progress-fill"
                style={{ width: p.total > 0 ? `${(p.done / p.total) * 100}%` : '0%' }}
              />
            </div>
          </div>
        ))}
      </div>

      {error && <div className="apply-error">{error}</div>}

      {/* Step list */}
      <div className="apply-steps">
        {data.phases.map(phase => (
          <div key={phase.name} className="apply-phase-group">
            <div className="apply-phase-title">{phase.name}</div>
            {phase.steps.map(step => (
              <div key={step.id}>
                <div
                  className={`apply-step apply-step-${step.status}`}
                  onClick={() => setExpandedStep(expandedStep === step.id ? null : step.id)}
                >
                  <span
                    className="apply-step-tag"
                    style={{ color: STATUS_COLOR[step.status] }}
                  >
                    [{STATUS_TAG[step.status] || '??'}]
                  </span>
                  <span className="apply-step-desc">{step.description}</span>
                </div>
                {expandedStep === step.id && (step.output || step.error) && (
                  <div className="apply-step-detail">
                    {step.output && <pre className="step-output">{step.output}</pre>}
                    {step.error && <pre className="step-error-text">{step.error}</pre>}
                  </div>
                )}
              </div>
            ))}
          </div>
        ))}
      </div>

      {/* Actions */}
      <div className="apply-actions">
        {!result && (
          <button
            className="btn btn-primary btn-lg"
            onClick={startApply}
            disabled={executing}
          >
            {executing ? 'Executing...' : 'Start Apply'}
          </button>
        )}
        {result && (
          <button className="btn btn-primary" onClick={() => onDone()}>
            View State
          </button>
        )}
      </div>
    </div>
  )
}
