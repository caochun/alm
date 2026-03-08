import { useState } from 'react'
import { getPlan } from '../services/api'
import './PlanView.css'

const STATUS_DOT = {
  pending:   '#3b82f6',
  skipped:   '#d1d5db',
  succeeded: '#22c55e',
  failed:    '#ef4444',
  running:   '#f59e0b',
}

export default function PlanView({ app, env, onApply }) {
  const [plan, setPlan] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [collapsed, setCollapsed] = useState({})

  const generate = (force = false) => {
    setLoading(true)
    setError('')
    getPlan(app, env, force)
      .then(data => { setPlan(data); setCollapsed({}) })
      .catch(err => setError(err.response?.data?.error || err.message))
      .finally(() => setLoading(false))
  }

  const togglePhase = (name) => {
    setCollapsed(c => ({ ...c, [name]: !c[name] }))
  }

  const counts = plan ? countSteps(plan) : null

  return (
    <div className="plan-view">
      <div className="plan-toolbar">
        <button className="btn btn-primary" onClick={() => generate(false)} disabled={loading}>
          {loading ? 'Generating...' : 'Generate Plan'}
        </button>
        <button className="btn btn-outline" onClick={() => generate(true)} disabled={loading}>
          Force Regenerate
        </button>
      </div>

      {error && <div className="plan-error">{error}</div>}

      {plan && (
        <>
          <div className="plan-header">
            {plan.app} / {plan.environment}
          </div>

          {plan.phases.map(phase => {
            const phaseCounts = phaseStepCounts(phase)
            const isCollapsed = collapsed[phase.name]
            const hasPending = phaseCounts.pending > 0

            return (
              <div key={phase.name} className="plan-phase">
                <div
                  className={`phase-header ${hasPending ? 'has-pending' : ''}`}
                  onClick={() => togglePhase(phase.name)}
                >
                  <span className="phase-toggle">{isCollapsed ? '\u25b8' : '\u25be'}</span>
                  <span className="phase-name">{phase.name}</span>
                  <span className="phase-count">
                    {phase.steps.length} step{phase.steps.length !== 1 ? 's' : ''}
                  </span>
                  {phaseCounts.pending > 0 && (
                    <span className="phase-badge pending">{phaseCounts.pending} pending</span>
                  )}
                  {phaseCounts.skipped > 0 && (
                    <span className="phase-badge skipped">{phaseCounts.skipped} skipped</span>
                  )}
                </div>

                {!isCollapsed && (
                  <div className="phase-steps">
                    {phase.steps.map(step => (
                      <div key={step.id} className={`step-row step-${step.status}`}>
                        <span
                          className="step-dot"
                          style={{ background: STATUS_DOT[step.status] || '#d1d5db' }}
                        />
                        <span className="step-desc">{step.description}</span>
                        {step.status === 'skipped' && !step.error && (
                          <span className="step-tag">unchanged</span>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )
          })}

          <div className="plan-summary">
            <span className="summary-item">
              <strong>{counts.pending}</strong> pending
            </span>
            <span className="summary-item">
              <strong>{counts.skipped}</strong> skipped
            </span>
            <span className="summary-item">
              <strong>{counts.total}</strong> total
            </span>
          </div>

          {counts.pending > 0 && (
            <div className="plan-actions">
              <button className="btn btn-primary btn-lg" onClick={() => onApply(plan)}>
                Apply Now
              </button>
            </div>
          )}

          {counts.pending === 0 && (
            <div className="plan-uptodate">All steps are up to date.</div>
          )}
        </>
      )}

      {!plan && !loading && !error && (
        <div className="plan-empty">Click "Generate Plan" to see what will be executed.</div>
      )}
    </div>
  )
}

function countSteps(plan) {
  let pending = 0, skipped = 0, total = 0
  for (const phase of plan.phases) {
    for (const step of phase.steps) {
      total++
      if (step.status === 'pending') pending++
      if (step.status === 'skipped') skipped++
    }
  }
  return { pending, skipped, total }
}

function phaseStepCounts(phase) {
  let pending = 0, skipped = 0
  for (const step of phase.steps) {
    if (step.status === 'pending') pending++
    if (step.status === 'skipped') skipped++
  }
  return { pending, skipped }
}
