import React, { useState, useEffect } from 'react'
import { assetAPI } from '../services/api'
import ExecutionModal from './ExecutionModal'
import './StateDetail.css'

function StateDetail({ stateId, currentState, graphData, onTransition, appPath, asset }) {
  const [stateAssets, setStateAssets] = useState([])
  const [loading, setLoading] = useState(false)
  const [executionModal, setExecutionModal] = useState({
    isOpen: false,
    data: null
  })

  useEffect(() => {
    if (stateId) {
      loadStateAssets()
    }
  }, [stateId, appPath])

  const loadStateAssets = async () => {
    try {
      setLoading(true)
      const response = await assetAPI.getAssets(appPath)
      const allAssets = response.data.assets || []
      
      // 过滤出当前状态的资产
      const filtered = allAssets.filter(asset => {
        // 这里需要根据实际API返回的数据结构调整
        return asset.stateId === stateId || 
               (stateId === currentState?.currentState?.id && asset.state === currentState?.currentState?.name)
      })
      
      setStateAssets(filtered)
    } catch (err) {
      console.error('Failed to load state assets:', err)
    } finally {
      setLoading(false)
    }
  }

  // 获取状态信息
  const stateInfo = graphData?.nodes?.find(n => n.id === stateId)
  const isCurrentState = stateId === currentState?.currentState?.id

  // 获取可用转换（仅当前状态）
  const availableTransitions = isCurrentState ? (currentState?.availableTransitions || []) : []

  const handleActionClick = async (transition) => {
    if (!transition) return

    // 检查是否是当前状态的转换
    if (!isCurrentState) {
      alert('只能从当前状态执行转换操作')
      return
    }

    const conditions = {}
    // 根据action类型设置conditions
    if (transition.action?.id === 'git-clone') {
      // 从asset配置中获取repository
      if (asset?.config?.application?.git?.repository) {
        conditions.repository = asset.config.application.git.repository
      } else {
        // 默认值
        conditions.repository = 'https://github.com/spring-projects/spring-petclinic.git'
      }
    }

    // 打开模态框，显示执行中状态
    setExecutionModal({
      isOpen: true,
      data: {
        action: transition.action?.name || transition.action,
        fromState: currentState?.currentState?.name,
        toState: transition.toState?.name,
        result: undefined, // 执行中
        timestamp: new Date().toISOString()
      }
    })

    try {
      // 执行状态转换
      const response = await assetAPI.transition(appPath, {
        toState: transition.toState.id,
        conditions,
        operator: 'user'
      })

      // 更新模态框，显示执行结果
      setExecutionModal({
        isOpen: true,
        data: {
          action: transition.action?.name || transition.action,
          fromState: response.data.fromState,
          toState: response.data.toState,
          result: response.data.result,
          timestamp: response.data.timestamp
        }
      })

      // 通知父组件刷新数据（不传递参数，只触发刷新）
      if (onTransition) {
        // 延迟一下，让用户看到结果
        setTimeout(() => {
          onTransition(transition.toState.id, conditions)
        }, 100)
      }
    } catch (error) {
      // 显示错误信息
      setExecutionModal({
        isOpen: true,
        data: {
          action: transition.action?.name || transition.action,
          fromState: currentState?.currentState?.name,
          toState: transition.toState?.name,
          result: {
            success: false,
            status: 'FAILED',
            error: error.response?.data?.error || error.message || '执行失败'
          },
          timestamp: new Date().toISOString()
        }
      })
    }
  }

  const handleCloseModal = () => {
    setExecutionModal({
      isOpen: false,
      data: null
    })
  }

  return (
    <div className="state-detail">
      <div className="state-header">
        <h3>{stateInfo?.label || '状态详情'}</h3>
        {isCurrentState && <span className="current-badge">当前状态</span>}
      </div>

      {isCurrentState && (
        <>
          {/* 可用转换 */}
          {availableTransitions.length > 0 ? (
            <div className="section">
              <h4>可用转换</h4>
              <div className="action-list">
                {availableTransitions.map((trans, index) => (
                  <button
                    key={index}
                    className="action-button"
                    onClick={() => handleActionClick(trans)}
                  >
                    <div className="action-name">{trans.action?.name}</div>
                    <div className="action-target">→ {trans.toState?.name}</div>
                    {trans.action?.description && (
                      <div className="action-desc">{trans.action.description}</div>
                    )}
                  </button>
                ))}
              </div>
            </div>
          ) : (
            <div className="section">
              <p className="hint">当前状态没有可用的转换操作</p>
            </div>
          )}

          {/* 当前状态的资产 */}
          <div className="section">
            <h4>资产列表</h4>
            {loading ? (
              <div className="loading">加载中...</div>
            ) : stateAssets.length > 0 ? (
              <div className="asset-list">
                {stateAssets.map((asset, index) => (
                  <div key={index} className="asset-item">
                    <div className="asset-name">{asset.name}</div>
                    <div className="asset-type">类型: {asset.type}</div>
                    <div className="asset-location">位置: {asset.location}</div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="empty">该状态下暂无资产</div>
            )}
          </div>
        </>
      )}

      {!isCurrentState && (
        <div className="section">
          <p className="hint">选择当前状态以查看详情和可用操作</p>
        </div>
      )}

      <ExecutionModal
        isOpen={executionModal.isOpen}
        onClose={handleCloseModal}
        executionData={executionModal.data}
      />
    </div>
  )
}

export default StateDetail

