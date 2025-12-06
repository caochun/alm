import axios from 'axios'

const API_BASE_URL = '/api/v1'

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 工作空间API
export const workspaceAPI = {
  // 获取应用列表
  getApplications: () => api.get('/workspace/applications'),
}

// 资产API
export const assetAPI = {
  // 获取资产信息
  getAsset: (appPath) => api.get(`/asset?appPath=${appPath}`),
  
  // 获取所有状态
  getStates: (appPath) => api.get(`/asset/states?appPath=${appPath}`),
  
  // 获取当前状态
  getCurrentState: (appPath) => api.get(`/asset/current-state?appPath=${appPath}`),
  
  // 获取状态机graph
  getStateMachineGraph: (appPath) => api.get(`/asset/graph?appPath=${appPath}`),
  
  // 获取所有资产
  getAssets: (appPath) => api.get(`/asset/assets?appPath=${appPath}`),
  
  // 获取转换历史
  getHistory: (appPath) => api.get(`/asset/history?appPath=${appPath}`),
  
  // 执行状态转换
  transition: (appPath, data) => api.post(`/asset/transition?appPath=${appPath}`, data),
}

export default api

