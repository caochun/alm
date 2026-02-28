import axios from 'axios'

const client = axios.create({ baseURL: '/api/v1' })

export const listApps = () =>
  client.get('/apps').then(r => r.data)

export const listEnvs = (app) =>
  client.get(`/apps/${encodeURIComponent(app)}/envs`).then(r => r.data)

export const getGraph = (app, env) =>
  client.get('/graph', { params: { app, env } }).then(r => r.data)
