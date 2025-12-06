import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { workspaceAPI } from '../services/api'
import './ApplicationList.css'

function ApplicationList() {
  const [applications, setApplications] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const navigate = useNavigate()

  useEffect(() => {
    loadApplications()
  }, [])

  const loadApplications = async () => {
    try {
      setLoading(true)
      const response = await workspaceAPI.getApplications()
      setApplications(response.data.applications || [])
      setError(null)
    } catch (err) {
      setError(err.message || '加载应用列表失败')
      console.error('Failed to load applications:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleSelectApplication = (appPath) => {
    navigate(`/app/${encodeURIComponent(appPath)}`)
  }

  if (loading) {
    return <div className="loading">加载中...</div>
  }

  if (error) {
    return <div className="error">错误: {error}</div>
  }

  return (
    <div className="application-list">
      <h2>应用列表</h2>
      {applications.length === 0 ? (
        <div className="empty-state">
          <p>暂无应用</p>
          <p className="hint">在 workspace 目录下创建包含 asset.yaml 的目录来添加应用</p>
        </div>
      ) : (
        <div className="app-grid">
          {applications.map((app) => (
            <div
              key={app.id}
              className="app-card"
              onClick={() => handleSelectApplication(app.path)}
            >
              <h3>{app.name}</h3>
              <p className="app-id">ID: {app.id}</p>
              <p className="app-description">{app.description || '无描述'}</p>
              <div className="app-path">{app.path}</div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export default ApplicationList

