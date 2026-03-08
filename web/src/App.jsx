import { useState, useEffect } from 'react'
import { listApps, listEnvs, getGraph } from './services/api'
import ArchGraph from './components/ArchGraph'
import PlanView from './components/PlanView'
import ApplyView from './components/ApplyView'
import StateView from './components/StateView'

const TABS = [
  { id: 'architecture', label: 'Architecture' },
  { id: 'plan',         label: 'Plan' },
  { id: 'apply',        label: 'Apply' },
  { id: 'state',        label: 'State' },
]

export default function App() {
  const [apps, setApps] = useState([])
  const [envs, setEnvs] = useState([])
  const [selectedApp, setSelectedApp] = useState('')
  const [selectedEnv, setSelectedEnv] = useState('')
  const [graphData, setGraphData] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [activeTab, setActiveTab] = useState('architecture')
  const [planData, setPlanData] = useState(null)

  // Load app list on mount
  useEffect(() => {
    listApps()
      .then(data => {
        setApps(data)
        if (data.length > 0) setSelectedApp(data[0])
      })
      .catch(() => setError('Failed to load app list'))
  }, [])

  // Load env list when app changes
  useEffect(() => {
    if (!selectedApp) return
    setSelectedEnv('')
    setGraphData(null)
    setError('')
    setPlanData(null)
    setActiveTab('architecture')
    listEnvs(selectedApp)
      .then(data => {
        setEnvs(data)
        if (data.length > 0) setSelectedEnv(data[0])
      })
      .catch(() => setError('Failed to load env list'))
  }, [selectedApp])

  // Load graph when app+env both selected
  useEffect(() => {
    if (!selectedApp || !selectedEnv) return
    setLoading(true)
    setError('')
    setPlanData(null)
    getGraph(selectedApp, selectedEnv)
      .then(data => {
        setGraphData(data)
        setLoading(false)
      })
      .catch(err => {
        setError('Failed to load graph: ' + (err.response?.data?.error || err.message))
        setLoading(false)
      })
  }, [selectedApp, selectedEnv])

  const handleApply = (plan) => {
    setPlanData(plan)
    setActiveTab('apply')
  }

  const handleApplyDone = () => {
    setActiveTab('state')
  }

  const ready = selectedApp && selectedEnv

  return (
    <>
      <div className="toolbar">
        <h1>ALM</h1>
        <div className="toolbar-divider" />

        <label>App</label>
        <select
          value={selectedApp}
          onChange={e => setSelectedApp(e.target.value)}
        >
          {apps.map(a => <option key={a} value={a}>{a}</option>)}
        </select>

        <label>Env</label>
        <select
          value={selectedEnv}
          onChange={e => setSelectedEnv(e.target.value)}
          disabled={envs.length === 0}
        >
          {envs.map(e => <option key={e} value={e}>{e}</option>)}
        </select>
      </div>

      {/* Tab bar */}
      <div className="tab-bar">
        {TABS.map(tab => (
          <button
            key={tab.id}
            className={`tab-item ${activeTab === tab.id ? 'active' : ''}`}
            onClick={() => setActiveTab(tab.id)}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Content area */}
      <div className="content-area">
        {activeTab === 'architecture' && (
          <div className="graph-container">
            {loading && <div className="state-message">Loading...</div>}
            {!loading && error && <div className="state-message error">{error}</div>}
            {!loading && !error && !graphData && (
              <div className="state-message">Select an app and environment</div>
            )}
            {!loading && !error && graphData && (
              <ArchGraph data={graphData} />
            )}
          </div>
        )}

        {activeTab === 'plan' && ready && (
          <PlanView app={selectedApp} env={selectedEnv} onApply={handleApply} />
        )}

        {activeTab === 'apply' && ready && (
          <ApplyView app={selectedApp} env={selectedEnv} plan={planData} onDone={handleApplyDone} />
        )}

        {activeTab === 'state' && ready && (
          <StateView app={selectedApp} env={selectedEnv} />
        )}

        {!ready && activeTab !== 'architecture' && (
          <div className="state-message">Select an app and environment first</div>
        )}
      </div>
    </>
  )
}
