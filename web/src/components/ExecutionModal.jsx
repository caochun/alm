import React from 'react'
import './ExecutionModal.css'

function ExecutionModal({ isOpen, onClose, executionData }) {
  if (!isOpen) return null

  const { action, fromState, toState, result, timestamp } = executionData || {}
  const isSuccess = result?.success
  const isCompleted = result !== undefined

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h3>执行详情</h3>
          {isCompleted && (
            <button className="modal-close" onClick={onClose}>×</button>
          )}
        </div>

        <div className="modal-body">
          {action && (
            <div className="execution-info">
              <div className="info-row">
                <span className="info-label">动作:</span>
                <span className="info-value">
                  {typeof action === 'string' ? action : (action.name || action.id || '未知动作')}
                </span>
              </div>
              {fromState && (
                <div className="info-row">
                  <span className="info-label">从状态:</span>
                  <span className="info-value">{fromState}</span>
                </div>
              )}
              {toState && (
                <div className="info-row">
                  <span className="info-label">到状态:</span>
                  <span className="info-value">{toState}</span>
                </div>
              )}
              {timestamp && (
                <div className="info-row">
                  <span className="info-label">执行时间:</span>
                  <span className="info-value">{new Date(timestamp).toLocaleString()}</span>
                </div>
              )}
            </div>
          )}

          {!isCompleted && (
            <div className="execution-status">
              <div className="status-loading">
                <div className="spinner"></div>
                <span>执行中...</span>
              </div>
            </div>
          )}

          {isCompleted && (
            <>
              <div className="execution-status">
                <div className={`status-badge ${isSuccess ? 'success' : 'error'}`}>
                  {isSuccess ? '✓ 执行成功' : '✗ 执行失败'}
                </div>
                {result.status && (
                  <div className="status-text">状态: {result.status}</div>
                )}
              </div>

              {result.output && (
                <div className="execution-output">
                  <div className="output-header">执行输出:</div>
                  <pre className="output-content">{result.output}</pre>
                </div>
              )}

              {result.error && (
                <div className="execution-error">
                  <div className="error-header">错误信息:</div>
                  <pre className="error-content">{result.error}</pre>
                </div>
              )}
            </>
          )}
        </div>

        {isCompleted && (
          <div className="modal-footer">
            <button className="btn-confirm" onClick={onClose}>
              确认
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

export default ExecutionModal

