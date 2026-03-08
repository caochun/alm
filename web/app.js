// ════════════════════════════════════════════════════════════════
// ALM — vanilla JS app (main)
// ════════════════════════════════════════════════════════════════

const API = '/api/v1'
const $ = (s) => document.querySelector(s)
const $$ = (s) => document.querySelectorAll(s)

// ── State ────────────────────────────────────────────────────
let selectedApp = ''
let selectedEnv = ''
let activeTab = 'architecture'
let graphData = null
let planData = null
let applyResult = null

// ── API helpers ──────────────────────────────────────────────
async function api(method, path, params = {}, body = null) {
  const url = new URL(API + path, location.origin)
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null && v !== false) url.searchParams.set(k, v)
  }
  const opts = { method }
  if (body) {
    opts.headers = { 'Content-Type': 'application/json' }
    opts.body = JSON.stringify(body)
  }
  const res = await fetch(url, opts)
  const data = await res.json()
  if (!res.ok) throw new Error(data.error || res.statusText)
  return data
}

// ── Init ─────────────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', async () => {
  initTabs()
  initPlanButtons()
  initApplyButtons()
  initStateButtons()
  initArchEditor()
  initDeployEditor()

  try {
    const apps = await api('GET', '/apps')
    const sel = $('#sel-app')
    sel.innerHTML = apps.map(a => `<option value="${a}">${a}</option>`).join('')
    if (apps.length) {
      selectedApp = apps[0]
      await loadEnvs()
    }
  } catch (e) {
    showGraphMsg('Failed to load apps: ' + e.message, true)
  }

  $('#sel-app').addEventListener('change', async (e) => {
    selectedApp = e.target.value
    selectedEnv = ''
    graphData = null
    planData = null
    applyResult = null
    switchTab('architecture')
    await loadEnvs()
  })

  $('#sel-env').addEventListener('change', async (e) => {
    selectedEnv = e.target.value
    planData = null
    applyResult = null
    await loadGraph()
  })
})

async function loadEnvs() {
  const sel = $('#sel-env')
  sel.disabled = true
  sel.innerHTML = ''
  try {
    const envs = await api('GET', `/apps/${encodeURIComponent(selectedApp)}/envs`)
    sel.innerHTML = envs.map(e => `<option value="${e}">${e}</option>`).join('')
    sel.disabled = envs.length === 0
    if (envs.length) {
      selectedEnv = envs[0]
    } else {
      selectedEnv = ''
    }
    await loadGraph()
  } catch (e) {
    showGraphMsg('Failed to load envs: ' + e.message, true)
  }
}

// ══════════════════════════════════════════════════════════════
// Tabs
// ══════════════════════════════════════════════════════════════
function initTabs() {
  $$('.tab-btn').forEach(btn => {
    btn.addEventListener('click', () => switchTab(btn.dataset.tab))
  })
}

function switchTab(tab) {
  activeTab = tab
  $$('.tab-btn').forEach(b => {
    const isActive = b.dataset.tab === tab
    b.classList.toggle('text-indigo-600', isActive)
    b.classList.toggle('border-indigo-500', isActive)
    b.classList.toggle('text-gray-500', !isActive)
    b.classList.toggle('border-transparent', !isActive)
  })
  $$('.tab-pane').forEach(p => p.classList.add('hidden'))
  $(`#tab-${tab}`).classList.remove('hidden')

  if (tab === 'state' && selectedApp && selectedEnv) loadState()
  if (tab === 'plan') renderPlanTab()
  if (tab === 'apply') renderApplyTab()
}

// ══════════════════════════════════════════════════════════════
// Architecture — SVG Graph (read-only viewer)
// ══════════════════════════════════════════════════════════════
const NODE_W = 230
const NODE_H = { service: 120, deliverable: 100, compute: 130, infra: 110, ingress: 130 }
const EDGE_STYLE = {
  dependsOn: { stroke: '#9ca3af', width: 1.5, dash: '6 4' },
  pipeline:  { stroke: '#6366f1', width: 2, dash: '' },
  provision: { stroke: '#22c55e', width: 2, dash: '' },
  binding:   { stroke: '#a855f7', width: 1.5, dash: '5 4' },
  route:     { stroke: '#f97316', width: 2, dash: '' },
}
const NODE_COLORS = {
  service:     { bg: '#eef2ff', border: '#a5b4fc', title: '#4338ca' },
  deliverable: { bg: '#fffbeb', border: '#fcd34d', title: '#92400e' },
  compute:     { bg: '#f0fdf4', border: '#86efac', title: '#15803d' },
  infra:       { bg: '#faf5ff', border: '#c4b5fd', title: '#6d28d9' },
  ingress:     { bg: '#fff7ed', border: '#fed7aa', title: '#c2410c' },
}
const ARTIFACT_ICON = { 'docker-image': '\u{1F433}', 'jar-file': '\u2615', 'static-bundle': '\u{1F4E6}', 'binary': '\u2699\uFE0F' }
const COL_LABELS = [
  { label: 'SERVICES', x: 30 }, { label: 'DELIVERABLES', x: 300 },
  { label: 'COMPUTE', x: 570 }, { label: 'INFRASTRUCTURE', x: 840 },
]

let positions = {}
let transform = { x: 20, y: 20, scale: 1 }
let panState = null
let dragState = null

function showGraphMsg(msg, isError = false) {
  const el = $('#graph-msg')
  el.textContent = msg
  el.className = `flex items-center justify-center h-full text-sm ${isError ? 'text-red-600' : 'text-gray-500'}`
  el.classList.remove('hidden')
  $('#graph-wrap').classList.add('hidden')
}

async function loadGraph() {
  if (!selectedApp) return
  showGraphMsg('Loading...')
  try {
    graphData = await api('GET', '/graph', { app: selectedApp, env: selectedEnv || undefined })
    positions = {}
    transform = { x: 20, y: 20, scale: 1 }
    renderGraph()
  } catch (e) {
    showGraphMsg('Failed: ' + e.message, true)
  }
}

function renderGraph() {
  if (!graphData) return
  $('#graph-msg').classList.add('hidden')
  const wrap = $('#graph-wrap')
  wrap.classList.remove('hidden')

  const nodeMap = buildNodeMap()
  const allNodes = Object.values(nodeMap)
  const maxX = allNodes.reduce((m, n) => Math.max(m, n.x + NODE_W + 60), 1100)
  const maxY = allNodes.reduce((m, n) => Math.max(m, n.y + (NODE_H[n.type] || 120) + 60), 500)
  const svgW = maxX
  const svgH = maxY + 20

  const defs = Object.entries(EDGE_STYLE).map(([t, s]) =>
    `<marker id="arrow-${t}" markerWidth="8" markerHeight="8" refX="7" refY="3" orient="auto">
       <path d="M0,0 L0,6 L8,3 z" fill="${s.stroke}" opacity="0.85"/>
     </marker>`
  ).join('')

  const colHeaders = COL_LABELS.map(c =>
    `<text x="${c.x}" y="28" font-size="11" font-weight="600" letter-spacing="0.05em" fill="#6b7280">${c.label}</text>`
  ).join('')

  const separators = [290, 560, 830].map(x =>
    `<line x1="${x}" y1="36" x2="${x}" y2="${svgH - 10}" stroke="#e5e7eb" stroke-width="1" stroke-dasharray="4 4"/>`
  ).join('')

  const edges = (graphData.edges || []).map(e => renderEdge(e, nodeMap)).join('')
  const nodes = (graphData.nodes || []).map(n => renderNode(nodeMap[n.id] || n)).join('')

  wrap.innerHTML = `<svg class="block" width="${svgW * transform.scale + transform.x + 40}" height="${svgH * transform.scale + transform.y + 40}">
    <defs>${defs}</defs>
    <g transform="translate(${transform.x},${transform.y}) scale(${transform.scale})">
      ${colHeaders}${separators}${edges}${nodes}
    </g>
  </svg>`

  wrap.onmousedown = (e) => {
    if (e.target.closest('.node-card')) return
    panState = { startX: e.clientX, startY: e.clientY, tx: transform.x, ty: transform.y }
  }
  wrap.onmousemove = (e) => {
    if (dragState) {
      const { id, startMX, startMY, startNX, startNY } = dragState
      const dx = (e.clientX - startMX) / transform.scale
      const dy = (e.clientY - startMY) / transform.scale
      positions[id] = { x: startNX + dx, y: startNY + dy }
      renderGraph()
    } else if (panState) {
      transform.x = panState.tx + (e.clientX - panState.startX)
      transform.y = panState.ty + (e.clientY - panState.startY)
      renderGraph()
    }
  }
  wrap.onmouseup = wrap.onmouseleave = () => { panState = null; dragState = null }
  wrap.addEventListener('wheel', (e) => {
    e.preventDefault()
    const factor = e.deltaY < 0 ? 1.1 : 0.9
    transform.scale = Math.min(Math.max(transform.scale * factor, 0.2), 3)
    renderGraph()
  }, { passive: false })

  wrap.querySelectorAll('[data-node-id]').forEach(fo => {
    fo.onmousedown = (e) => {
      e.stopPropagation()
      const id = fo.dataset.nodeId
      const n = nodeMap[id]
      dragState = { id, startMX: e.clientX, startMY: e.clientY, startNX: n.x, startNY: n.y }
    }
  })
}

function buildNodeMap() {
  const map = {}
  for (const n of (graphData.nodes || [])) {
    const p = positions[n.id]
    map[n.id] = p ? { ...n, x: p.x, y: p.y } : { ...n }
  }
  return map
}

function renderNode(n) {
  const h = NODE_H[n.type] || 120
  const c = NODE_COLORS[n.type] || NODE_COLORS.service
  const d = n.data || {}
  let inner = ''

  if (n.type === 'service') {
    inner = `
      <div class="text-[13px] font-semibold truncate" style="color:${c.title}">${d.name}</div>
      <div class="flex items-center gap-1.5 text-[11px] text-gray-500 truncate"><span class="text-gray-400 shrink-0">pipeline</span>${d.pipeline}</div>
      <div class="flex items-center gap-1.5 text-[11px] text-gray-500 truncate"><span class="text-gray-400 shrink-0">repo</span>${repoShort(d.repository)}</div>`
  } else if (n.type === 'deliverable') {
    const icon = ARTIFACT_ICON[d.artifactType] || '\u{1F4C4}'
    inner = `
      <div class="text-2xl leading-none">${icon}</div>
      <div class="text-xs font-semibold" style="color:${c.title}">${d.artifactType || '\u2014'}</div>
      <div class="text-[10px]" style="color:#b45309">${d.pipeline}</div>`
  } else if (n.type === 'compute') {
    const res = [d.cpu && `CPU ${d.cpu}`, d.memory && `Mem ${d.memory}`, d.replicas > 1 && `\u00d7${d.replicas}`].filter(Boolean).join('  ')
    inner = `
      <div class="text-[13px] font-semibold truncate" style="color:${c.title}">${d.name}</div>
      <span class="inline-flex items-center rounded-md bg-gray-100 px-2 py-0.5 text-[10px] font-medium text-gray-600 w-fit">${d.computeType}</span>
      <div class="flex items-center gap-1.5 text-[11px] text-gray-500 truncate"><span class="text-gray-400 shrink-0">accepts</span>${d.accepts}</div>
      ${res ? `<div class="flex items-center gap-1.5 text-[11px] text-gray-500 truncate"><span class="text-gray-400 shrink-0">res</span>${res}</div>` : ''}`
  } else if (n.type === 'infra') {
    inner = `
      <div class="text-[13px] font-semibold truncate" style="color:${c.title}">${d.name}</div>
      <span class="inline-flex items-center rounded-md bg-gray-100 px-2 py-0.5 text-[10px] font-medium text-gray-600 w-fit">${d.resourceType}</span>
      <div class="flex items-center gap-1.5 text-[11px] text-gray-500 truncate"><span class="text-gray-400 shrink-0">via</span>${d.via}</div>`
  } else if (n.type === 'ingress') {
    const routes = (d.routes || []).slice(0, 3).map(r => `<div class="truncate">${r.path} \u2192 ${r.service}:${r.port}</div>`).join('')
    const more = (d.routes || []).length > 3 ? `<div>+${d.routes.length - 3} more\u2026</div>` : ''
    const bind = d.bind ? `${d.bind.ip}:${d.bind.http}${d.bind.https ? '/:' + d.bind.https : ''}` : ''
    inner = `
      <div class="text-[13px] font-semibold truncate" style="color:${c.title}">${d.name}</div>
      <span class="inline-flex items-center rounded-md bg-gray-100 px-2 py-0.5 text-[10px] font-medium text-gray-600 w-fit">${d.ingressType}</span>
      ${bind ? `<div class="flex items-center gap-1.5 text-[11px] text-gray-500 truncate"><span class="text-gray-400 shrink-0">bind</span>${bind}</div>` : ''}
      <div class="mt-0.5 text-[10px] text-gray-500">${routes}${more}</div>`
  }

  const align = n.type === 'deliverable' ? 'justify-center items-center text-center' : ''

  return `<foreignObject data-node-id="${n.id}" x="${n.x}" y="${n.y}" width="${NODE_W}" height="${h}" style="overflow:visible;cursor:move">
    <div xmlns="http://www.w3.org/1999/xhtml" style="width:${NODE_W}px;height:${h}px">
      <div class="node-card w-full h-full rounded-lg px-3 py-2.5 flex flex-col gap-1 overflow-hidden ${align}"
           style="background:${c.bg};border:1px solid ${c.border};box-shadow:0 1px 3px rgba(0,0,0,0.1);font-family:ui-sans-serif,system-ui,sans-serif">
        ${inner}
      </div>
    </div>
  </foreignObject>`
}

function renderEdge(edge, nodeMap) {
  const src = nodeMap[edge.source], tgt = nodeMap[edge.target]
  if (!src || !tgt) return ''
  const st = EDGE_STYLE[edge.type] || EDGE_STYLE.pipeline
  const srcH = NODE_H[src.type] || 120, tgtH = NODE_H[tgt.type] || 120
  let x1, y1, x2, y2, cx1, cy1, cx2, cy2

  if (edge.type === 'dependsOn') {
    x1 = src.x + NODE_W / 2; y1 = src.y + srcH
    x2 = tgt.x + NODE_W / 2; y2 = tgt.y
    const dy = (y2 - y1) / 2
    cx1 = x1 - 40; cy1 = y1 + dy; cx2 = x2 - 40; cy2 = y1 + dy
  } else if (edge.type === 'route') {
    x1 = tgt.x + NODE_W; y1 = tgt.y + tgtH / 2
    x2 = src.x; y2 = src.y + srcH / 2
    const midX = (x1 + x2) / 2 + 80
    cx1 = midX; cy1 = y1; cx2 = midX; cy2 = y2
  } else {
    x1 = src.x + NODE_W; y1 = src.y + srcH / 2
    x2 = tgt.x; y2 = tgt.y + tgtH / 2
    const dx = (x2 - x1) * 0.5
    cx1 = x1 + dx; cy1 = y1; cx2 = x2 - dx; cy2 = y2
  }

  const d = `M ${x1} ${y1} C ${cx1} ${cy1}, ${cx2} ${cy2}, ${x2} ${y2}`
  const mx = (x1 + x2) / 2, my = (y1 + y2) / 2
  const label = edge.label ? `<text x="${mx}" y="${my - 6}" text-anchor="middle" font-size="10" fill="${st.stroke}" font-weight="600" opacity="0.9">${edge.label}</text>` : ''

  return `<g>
    <path d="${d}" fill="none" stroke="${st.stroke}" stroke-width="${st.width}" stroke-dasharray="${st.dash}" marker-end="url(#arrow-${edge.type})" opacity="0.85"/>
    ${label}
  </g>`
}

function repoShort(url) {
  if (!url) return '\u2014'
  try { return new URL(url).pathname.replace(/^\//, '').replace(/\.git$/, '') } catch { return url }
}

// ══════════════════════════════════════════════════════════════
// Plan
// ══════════════════════════════════════════════════════════════
const STATUS_DOT = { pending: '#6366f1', skipped: '#d1d5db', succeeded: '#22c55e', failed: '#ef4444', running: '#f59e0b' }

function initPlanButtons() {
  $('#btn-plan').addEventListener('click', () => generatePlan(false))
  $('#btn-plan-force').addEventListener('click', () => generatePlan(true))
}

async function generatePlan(force) {
  if (!selectedApp || !selectedEnv) return
  $('#btn-plan').textContent = 'Generating...'
  $('#btn-plan').disabled = true
  $('#plan-error').classList.add('hidden')
  try {
    planData = await api('POST', '/plan', { app: selectedApp, env: selectedEnv, force: force || undefined })
    applyResult = null
    renderPlanTab()
  } catch (e) {
    $('#plan-error').textContent = e.message
    $('#plan-error').classList.remove('hidden')
  } finally {
    $('#btn-plan').textContent = 'Generate Plan'
    $('#btn-plan').disabled = false
  }
}

function renderPlanTab() {
  const body = $('#plan-body')
  const empty = $('#plan-empty')
  if (!planData) { body.innerHTML = ''; empty.classList.remove('hidden'); return }
  empty.classList.add('hidden')

  let pending = 0, skipped = 0, total = 0
  for (const ph of planData.phases) for (const s of ph.steps) {
    total++
    if (s.status === 'pending') pending++
    if (s.status === 'skipped') skipped++
  }

  let html = `<div class="text-sm font-medium text-gray-900 mb-4 pb-3 border-b border-gray-200">${planData.app} / ${planData.environment}</div>`

  for (const phase of planData.phases) {
    const pPending = phase.steps.filter(s => s.status === 'pending').length
    const pSkipped = phase.steps.filter(s => s.status === 'skipped').length
    const hasPending = pPending > 0

    html += `<div class="mb-3 overflow-hidden rounded-lg ring-1 ring-gray-200">
      <div class="flex items-center gap-2 px-4 py-3 text-sm cursor-pointer select-none ${hasPending ? 'bg-indigo-50' : 'bg-gray-50'} hover:bg-gray-100" onclick="this.nextElementSibling.classList.toggle('hidden')">
        <span class="text-gray-400 text-xs w-3">\u25be</span>
        <span class="font-semibold capitalize text-gray-900">${phase.name}</span>
        <span class="text-gray-500 text-xs">${phase.steps.length} step${phase.steps.length !== 1 ? 's' : ''}</span>
        ${pPending > 0 ? `<span class="inline-flex items-center rounded-full bg-indigo-100 px-2 py-0.5 text-xs font-medium text-indigo-700">${pPending} pending</span>` : ''}
        ${pSkipped > 0 ? `<span class="inline-flex items-center rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600">${pSkipped} skipped</span>` : ''}
      </div>
      <div class="divide-y divide-gray-100">`

    for (const step of phase.steps) {
      const color = STATUS_DOT[step.status] || '#d1d5db'
      const dim = step.status === 'skipped' ? 'text-gray-400' : 'text-gray-700'
      html += `<div class="flex items-center gap-3 px-4 py-2.5 text-sm ${dim}">
        <span class="h-2 w-2 rounded-full shrink-0" style="background:${color}"></span>
        <span class="flex-1 font-mono text-xs">${step.description}</span>
        ${step.status === 'skipped' && !step.error ? '<span class="text-xs text-gray-400">unchanged</span>' : ''}
      </div>`
    }
    html += `</div></div>`
  }

  html += `<div class="flex gap-6 pt-4 mt-2 text-sm text-gray-500">
    <span><span class="font-semibold text-gray-900">${pending}</span> pending</span>
    <span><span class="font-semibold text-gray-900">${skipped}</span> skipped</span>
    <span><span class="font-semibold text-gray-900">${total}</span> total</span>
  </div>`

  if (pending > 0) {
    html += `<div class="pt-4"><button onclick="applyFromPlan()" class="inline-flex items-center rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500">Apply Now</button></div>`
  } else {
    html += `<div class="pt-4 text-sm font-medium text-green-600">All steps are up to date.</div>`
  }

  body.innerHTML = html
}

window.applyFromPlan = function () {
  applyResult = null
  switchTab('apply')
}

// ══════════════════════════════════════════════════════════════
// Apply
// ══════════════════════════════════════════════════════════════
function initApplyButtons() {}

function renderApplyTab() {
  const data = applyResult || planData
  const empty = $('#apply-empty')
  const bars = $('#apply-bars')
  const steps = $('#apply-steps')
  const actions = $('#apply-actions')
  const errEl = $('#apply-error')
  errEl.classList.add('hidden')

  if (!data) {
    empty.classList.remove('hidden')
    bars.innerHTML = ''; steps.innerHTML = ''; actions.innerHTML = ''
    return
  }
  empty.classList.add('hidden')

  bars.innerHTML = data.phases.map(ph => {
    const done = ph.steps.filter(s => s.status === 'succeeded' || s.status === 'skipped').length
    const total = ph.steps.length
    const pct = total > 0 ? (done / total) * 100 : 0
    return `<div class="rounded-lg bg-white p-4 ring-1 ring-gray-200">
      <div class="flex justify-between items-center mb-2">
        <span class="text-sm font-medium capitalize text-gray-900">${ph.name}</span>
        <span class="text-xs text-gray-500 font-mono">${done}/${total}</span>
      </div>
      <div class="h-1.5 w-full rounded-full bg-gray-200">
        <div class="h-1.5 rounded-full bg-indigo-600 transition-all" style="width:${pct}%"></div>
      </div>
    </div>`
  }).join('')

  const STATUS_TAG = { pending: '    ', running: ' .. ', succeeded: ' ok ', failed: 'FAIL', skipped: 'skip' }
  const TAG_COLOR = { pending: 'text-gray-300', running: 'text-amber-500', succeeded: 'text-green-600', failed: 'text-red-600', skipped: 'text-gray-300' }
  const DESC_COLOR = { skipped: 'text-gray-400', failed: 'text-red-700', succeeded: 'text-green-700' }

  steps.innerHTML = data.phases.map(ph => {
    const rows = ph.steps.map(s => {
      const detail = (s.output || s.error)
        ? `<div class="hidden ml-8 mt-0.5 mb-2 pl-3 border-l-2 border-gray-200">
             ${s.output ? `<pre class="text-xs text-gray-500 whitespace-pre-wrap break-all m-0">${esc(s.output)}</pre>` : ''}
             ${s.error ? `<pre class="text-xs text-red-600 whitespace-pre-wrap break-all m-0">${esc(s.error)}</pre>` : ''}
           </div>` : ''
      return `<div>
        <div class="flex items-center gap-2 px-2 py-1.5 rounded-md cursor-pointer hover:bg-gray-100" onclick="this.nextElementSibling&&this.nextElementSibling.classList.toggle('hidden')">
          <span class="font-mono text-xs font-semibold shrink-0 ${TAG_COLOR[s.status] || ''}">[${STATUS_TAG[s.status] || '??'}]</span>
          <span class="font-mono text-xs ${DESC_COLOR[s.status] || 'text-gray-700'}">${s.description}</span>
        </div>${detail}</div>`
    }).join('')
    return `<div class="mb-4">
      <div class="text-xs font-semibold uppercase tracking-wide text-gray-500 py-2">${ph.name}</div>
      ${rows}
    </div>`
  }).join('')

  if (!applyResult) {
    actions.innerHTML = `<button id="btn-apply" class="inline-flex items-center rounded-md bg-indigo-600 px-5 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500" onclick="startApply()">Start Apply</button>`
  } else {
    actions.innerHTML = `<button class="inline-flex items-center rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500" onclick="switchTab('state')">View State</button>`
  }
}

window.startApply = async function () {
  if (!selectedApp || !selectedEnv) return
  const btn = $('#btn-apply')
  if (btn) { btn.textContent = 'Executing...'; btn.disabled = true }
  $('#apply-error').classList.add('hidden')
  try {
    applyResult = await api('POST', '/apply', { app: selectedApp, env: selectedEnv })
  } catch (e) {
    $('#apply-error').textContent = e.message
    $('#apply-error').classList.remove('hidden')
  }
  renderApplyTab()
}

// ══════════════════════════════════════════════════════════════
// State
// ══════════════════════════════════════════════════════════════
function initStateButtons() {
  $('#btn-state-refresh').addEventListener('click', loadState)
}

async function loadState() {
  if (!selectedApp || !selectedEnv) return
  const body = $('#state-body')
  const empty = $('#state-empty')
  const time = $('#state-time')
  body.innerHTML = '<div class="text-gray-500 text-sm py-8 text-center">Loading...</div>'
  empty.textContent = ''

  try {
    const state = await api('GET', '/state', { app: selectedApp, env: selectedEnv })
    const hasSvc = state.services && Object.keys(state.services).length > 0
    const hasInfra = state.infra && Object.keys(state.infra).length > 0
    const hasIngress = state.ingress && Object.keys(state.ingress).length > 0

    if (state.updatedAt && state.updatedAt !== '0001-01-01T00:00:00Z') {
      time.textContent = 'Last updated: ' + new Date(state.updatedAt).toLocaleString()
    } else {
      time.textContent = ''
    }

    if (!hasSvc && !hasInfra && !hasIngress) {
      body.innerHTML = ''
      empty.textContent = 'No deployment state found. Run Apply to deploy.'
      return
    }

    let html = ''
    if (hasSvc) html += stateTable('Services', state.services, 'deployedAt', [{ key: 'artifactRef', label: 'Artifact' }])
    if (hasInfra) html += stateTable('Infrastructure', state.infra, 'provisionedAt', [])
    if (hasIngress) html += stateTable('Ingress', state.ingress, 'configuredAt', [])
    body.innerHTML = html
    empty.textContent = ''
  } catch (e) {
    body.innerHTML = `<div class="rounded-md bg-red-50 p-4 text-sm text-red-700">${esc(e.message)}</div>`
  }
}

function stateTable(title, items, timeField, extraCols) {
  const entries = Object.entries(items)
  const ths = extraCols.map(c => `<th class="px-3 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500">${c.label}</th>`).join('')

  const rows = entries.map(([name, item]) => {
    const statusCls = { running: 'bg-green-50 text-green-700 ring-green-600/20', stopped: 'bg-gray-50 text-gray-600 ring-gray-500/10', failed: 'bg-red-50 text-red-700 ring-red-600/10' }
    const badge = `<span class="inline-flex items-center rounded-md px-2 py-1 text-xs font-medium ring-1 ring-inset ${statusCls[item.status] || 'bg-gray-50 text-gray-600 ring-gray-500/10'}">${item.status || 'unknown'}</span>`
    const extras = extraCols.map(c => `<td class="whitespace-nowrap px-3 py-4 text-sm font-mono text-gray-500">${item[c.key] || '\u2014'}</td>`).join('')
    return `<tr>
      <td class="whitespace-nowrap px-3 py-4 text-sm font-medium text-gray-900">${name}</td>
      <td class="whitespace-nowrap px-3 py-4 text-sm">${badge}</td>
      ${extras}
      <td class="whitespace-nowrap px-3 py-4 text-sm text-gray-500">${timeAgo(item[timeField])}</td>
    </tr>`
  }).join('')

  return `<div class="mb-8">
    <h3 class="text-sm font-semibold text-gray-900 mb-3">${title}</h3>
    <div class="overflow-hidden rounded-lg ring-1 ring-gray-200">
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50"><tr>
          <th class="px-3 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500">Name</th>
          <th class="px-3 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500">Status</th>
          ${ths}
          <th class="px-3 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500">Time</th>
        </tr></thead>
        <tbody class="divide-y divide-gray-100 bg-white">${rows}</tbody>
      </table>
    </div>
  </div>`
}

// ── Util ─────────────────────────────────────────────────────
function timeAgo(dateStr) {
  if (!dateStr) return '\u2014'
  const d = new Date(dateStr)
  if (isNaN(d.getTime())) return '\u2014'
  const mins = Math.floor((Date.now() - d.getTime()) / 60000)
  if (mins < 1) return 'just now'
  if (mins < 60) return mins + 'm ago'
  const hours = Math.floor(mins / 60)
  if (hours < 24) return hours + 'h ago'
  return Math.floor(hours / 24) + 'd ago'
}

function esc(s) {
  const el = document.createElement('div')
  el.textContent = s
  return el.innerHTML
}
