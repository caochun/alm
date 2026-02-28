import { useState, useEffect } from 'react'
import { listApps, listEnvs, getGraph } from './services/api'
import ArchGraph from './components/ArchGraph'

export default function App() {
  const [apps, setApps] = useState([])
  const [envs, setEnvs] = useState([])
  const [selectedApp, setSelectedApp] = useState('')
  const [selectedEnv, setSelectedEnv] = useState('')
  const [graphData, setGraphData] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  // Load app list on mount
  useEffect(() => {
    listApps()
      .then(data => {
        setApps(data)
        if (data.length > 0) setSelectedApp(data[0])
      })
      .catch(() => setError('无法加载应用列表'))
  }, [])

  // Load env list when app changes
  useEffect(() => {
    if (!selectedApp) return
    setSelectedEnv('')
    setGraphData(null)
    setError('')
    listEnvs(selectedApp)
      .then(data => {
        setEnvs(data)
        if (data.length > 0) setSelectedEnv(data[0])
      })
      .catch(() => setError('无法加载环境列表'))
  }, [selectedApp])

  // Load graph when app+env both selected
  useEffect(() => {
    if (!selectedApp || !selectedEnv) return
    setLoading(true)
    setError('')
    getGraph(selectedApp, selectedEnv)
      .then(data => {
        setGraphData(data)
        setLoading(false)
      })
      .catch(err => {
        setError('加载图形数据失败: ' + (err.response?.data?.error || err.message))
        setLoading(false)
      })
  }, [selectedApp, selectedEnv])

  return (
    <>
      <div className="toolbar">
        <h1>ALM</h1>
        <div className="toolbar-divider" />

        <label>应用</label>
        <select
          value={selectedApp}
          onChange={e => setSelectedApp(e.target.value)}
        >
          {apps.map(a => <option key={a} value={a}>{a}</option>)}
        </select>

        <label>环境</label>
        <select
          value={selectedEnv}
          onChange={e => setSelectedEnv(e.target.value)}
          disabled={envs.length === 0}
        >
          {envs.map(e => <option key={e} value={e}>{e}</option>)}
        </select>

        <div className="toolbar-spacer" />
        <span className="toolbar-hint">滚轮缩放 · 拖拽平移</span>
      </div>

      <div className="graph-container">
        {loading && <div className="state-message">加载中…</div>}
        {!loading && error && <div className="state-message error">{error}</div>}
        {!loading && !error && !graphData && (
          <div className="state-message">请选择应用和环境</div>
        )}
        {!loading && !error && graphData && (
          <ArchGraph data={graphData} />
        )}
      </div>
    </>
  )
}
