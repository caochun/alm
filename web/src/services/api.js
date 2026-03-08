import axios from 'axios'

const client = axios.create({ baseURL: '/api/v1' })

export const listApps = () =>
  client.get('/apps').then(r => r.data)

export const listEnvs = (app) =>
  client.get(`/apps/${encodeURIComponent(app)}/envs`).then(r => r.data)

export const getGraph = (app, env) =>
  client.get('/graph', { params: { app, env } }).then(r => r.data)

export const getPlan = (app, env, force = false) =>
  client.post('/plan', null, { params: { app, env, force } }).then(r => r.data)

export const postApply = (app, env, dryRun = false, force = false) =>
  client.post('/apply', null, { params: { app, env, dryRun, force } }).then(r => r.data)

export const getState = (app, env) =>
  client.get('/state', { params: { app, env } }).then(r => r.data)
