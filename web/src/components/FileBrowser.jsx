import React, { useState, useEffect } from 'react'
import { assetAPI } from '../services/api'
import './FileBrowser.css'

function FileBrowser({ appPath }) {
  const [files, setFiles] = useState([])
  const [currentPath, setCurrentPath] = useState('')
  const [fileContent, setFileContent] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  useEffect(() => {
    if (appPath) {
      loadFiles('')
    }
  }, [appPath])

  const loadFiles = async (path) => {
    try {
      setLoading(true)
      setError(null)
      setFileContent(null)

      const response = await fetch(`/api/v1/files?appPath=${encodeURIComponent(appPath)}&path=${encodeURIComponent(path)}`)
      const data = await response.json()

      if (data.type === 'file') {
        setFileContent(data)
        setFiles([])
      } else {
        setFiles(data.files || [])
        setFileContent(null)
      }
      setCurrentPath(path)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleFileClick = (file) => {
    if (file.type === 'directory') {
      loadFiles(file.path)
    } else {
      loadFiles(file.path)
    }
  }

  const handleBack = () => {
    if (currentPath === '') return
    const parentPath = currentPath.split('/').slice(0, -1).join('/')
    loadFiles(parentPath)
  }

  if (loading) {
    return <div className="file-browser-loading">加载中...</div>
  }

  if (error) {
    return <div className="file-browser-error">错误: {error}</div>
  }

  return (
    <div className="file-browser">
      <div className="file-browser-header">
        <div className="file-browser-path">
          {currentPath !== '' && (
            <button className="back-button" onClick={handleBack}>
              ← 返回
            </button>
          )}
          <span className="path-text">/{currentPath || '根目录'}</span>
        </div>
      </div>

      {fileContent ? (
        <div className="file-content-viewer">
          <div className="file-content-header">
            <span>{fileContent.path}</span>
            <button onClick={() => setFileContent(null)}>关闭</button>
          </div>
          <pre className="file-content">{fileContent.content}</pre>
        </div>
      ) : (
        <div className="file-list">
          {files.length === 0 ? (
            <div className="empty-directory">目录为空</div>
          ) : (
            files.map((file, index) => (
              <div
                key={index}
                className={`file-item ${file.type}`}
                onClick={() => handleFileClick(file)}
              >
                <span className="file-icon">
                  {file.type === 'directory' ? '📁' : '📄'}
                </span>
                <span className="file-name">{file.name}</span>
                {file.type === 'file' && (
                  <span className="file-size">{(file.size / 1024).toFixed(2)} KB</span>
                )}
              </div>
            ))
          )}
        </div>
      )}
    </div>
  )
}

export default FileBrowser

