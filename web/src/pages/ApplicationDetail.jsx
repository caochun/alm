import React, { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { assetAPI } from '../services/api'
import StateMachineGraph from '../components/StateMachineGraph'
import StateDetail from '../components/StateDetail'
import FileBrowser from '../components/FileBrowser'
import './ApplicationDetail.css'

function ApplicationDetail() {
  const { appPath } = useParams()
  const navigate = useNavigate()
  const decodedAppPath = decodeURIComponent(appPath)
  
  const [asset, setAsset] = useState(null)
  const [graphData, setGraphData] = useState(null)
  const [currentState, setCurrentState] = useState(null)
  const [selectedStateId, setSelectedStateId] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [activeTab, setActiveTab] = useState('state-machine') // 'state-machine' or 'files'

  useEffect(() => {
    loadApplicationData()
  }, [decodedAppPath])

  const loadApplicationData = async () => {
    try {
      setLoading(true)
      setError(null)

      // 加载资产信息
      const assetRes = await assetAPI.getAsset(decodedAppPath)
      setAsset(assetRes.data)

      // 加载状态机graph
      const graphRes = await assetAPI.getStateMachineGraph(decodedAppPath)
      setGraphData(graphRes.data)

      // 加载当前状态
      const stateRes = await assetAPI.getCurrentState(decodedAppPath)
      setCurrentState(stateRes.data)
      setSelectedStateId(stateRes.data.currentState.id)

    } catch (err) {
      setError(err.message || '加载应用数据失败')
      console.error('Failed to load application data:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleStateSelect = (stateId) => {
    setSelectedStateId(stateId)
  }

  const handleTransition = async (toState, conditions) => {
    // 只负责刷新数据，状态转换已经在StateDetail中完成
    await loadApplicationData()
  }

  if (loading) {
    return <div className="loading">加载中...</div>
  }

  if (error) {
    return (
      <div className="error-container">
        <div className="error">错误: {error}</div>
        <button onClick={() => navigate('/')}>返回应用列表</button>
      </div>
    )
  }

  if (!asset || !graphData || !currentState) {
    return <div className="loading">数据加载中...</div>
  }

  return (
    <div className="application-detail">
      <div className="detail-header">
        <button className="back-button" onClick={() => navigate('/')}>
          ← 返回
        </button>
        <div>
          <h2>{asset.name}</h2>
          <p className="asset-id">ID: {asset.id}</p>
        </div>
      </div>

      <div className="detail-tabs">
        <button
          className={activeTab === 'state-machine' ? 'active' : ''}
          onClick={() => setActiveTab('state-machine')}
        >
          状态机
        </button>
        <button
          className={activeTab === 'files' ? 'active' : ''}
          onClick={() => setActiveTab('files')}
        >
          文件浏览
        </button>
      </div>

      <div className="detail-content">
        {activeTab === 'state-machine' ? (
          <>
            <div className="graph-section">
              <h3>状态机</h3>
              <StateMachineGraph
                graphData={graphData}
                currentStateId={currentState.currentState.id}
                onStateSelect={handleStateSelect}
                selectedStateId={selectedStateId}
              />
            </div>

            <div className="state-section">
              <StateDetail
                stateId={selectedStateId || currentState.currentState.id}
                currentState={currentState}
                graphData={graphData}
                onTransition={handleTransition}
                appPath={decodedAppPath}
                asset={asset}
              />
            </div>
          </>
        ) : (
          <div className="files-section">
            <FileBrowser appPath={decodedAppPath} />
          </div>
        )}
      </div>
    </div>
  )
}

export default ApplicationDetail

