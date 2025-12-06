import React from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import ApplicationList from './pages/ApplicationList'
import ApplicationDetail from './pages/ApplicationDetail'
import './App.css'

function App() {
  return (
    <div className="app">
      <header className="app-header">
        <h1>ALM - 应用生命周期管理</h1>
      </header>
      <main className="app-main">
        <Routes>
          <Route path="/" element={<ApplicationList />} />
          <Route path="/app/:appPath" element={<ApplicationDetail />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </main>
    </div>
  )
}

export default App

