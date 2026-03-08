// ════════════════════════════════════════════════════════════════
// arch-editor.js — App Architecture Visual Editor
// ════════════════════════════════════════════════════════════════

const SERVICE_TEMPLATES = [
  { label: 'Java 后端服务',  pipeline: 'java-webapp-pipeline', icon: '\u2615' },
  { label: 'Node.js 前端',  pipeline: 'nodejs-spa-pipeline',  icon: '\uD83D\uDFE2' },
  { label: '后台 Worker',   pipeline: 'java-webapp-pipeline', icon: '\u2699\uFE0F' },
  { label: '自定义服务',     pipeline: '',                     icon: '\uD83D\uDCE6' },
]

let archState = {
  mode: null,       // 'create' | 'edit'
  appName: '',
  description: '',
  services: [],     // [{ id, name, pipeline, repository, depends_on: [] }]
  nextId: 1,
  canvas: null,
  selectedId: null,
}

let archPipelines = [] // cached from API

function initArchEditor() {
  document.getElementById('btn-new-app').addEventListener('click', () => openArchEditor('create'))
  document.getElementById('btn-edit-arch').addEventListener('click', () => openArchEditor('edit'))
}

async function openArchEditor(mode) {
  archState.mode = mode
  archState.appName = ''
  archState.description = ''
  archState.services = []
  archState.nextId = 1
  archState.selectedId = null

  // Load pipelines
  if (!archPipelines.length) {
    try { archPipelines = await api('GET', '/pipelines') } catch (e) { alert('Failed to load pipelines: ' + e.message); return }
  }

  // Load existing data in edit mode
  if (mode === 'edit' && selectedApp) {
    try {
      const arch = await api('GET', `/apps/${encodeURIComponent(selectedApp)}/arch`)
      archState.appName = arch.Name
      archState.description = arch.Description || ''
      archState.services = (arch.Services || []).map((s, i) => ({
        id: 'svc-' + (i + 1),
        name: s.Name,
        pipeline: s.Pipeline,
        repository: s.Repository,
        depends_on: s.DependsOn || []
      }))
      archState.nextId = archState.services.length + 1
    } catch (e) { alert('Failed to load architecture: ' + e.message); return }
    document.getElementById('arch-editor-title').textContent = 'Edit Architecture: ' + selectedApp
  } else {
    document.getElementById('arch-editor-title').textContent = 'New Application'
  }

  document.getElementById('arch-editor-overlay').classList.remove('hidden')
  _archInitCanvas()
  _archRenderSidebar()
  _archRenderProps()
  _archRenderFooter()
}

function closeArchEditor() {
  document.getElementById('arch-editor-overlay').classList.add('hidden')
  archState.canvas = null
}

// ── Canvas ───────────────────────────────────────────────────

function _archInitCanvas() {
  const wrap = document.getElementById('arch-canvas-wrap')
  wrap.innerHTML = ''
  const ctrl = createCanvasController(wrap, { portMode: 'bottom' })
  archState.canvas = ctrl

  // Place existing services
  archState.services.forEach((s, i) => {
    const pt = archPipelines.find(p => p.name === s.pipeline)
    ctrl.addNode(s.id, 'service', 80 + (i % 3) * 260, 60 + Math.floor(i / 3) * 140, {
      name: s.name,
      pipeline: s.pipeline,
      repository: s.repository,
      deliverables: pt ? pt.deliverables : []
    })
  })

  // Draw dependency edges
  archState.services.forEach(s => {
    (s.depends_on || []).forEach(depName => {
      const dep = archState.services.find(d => d.name === depName)
      if (dep) ctrl.addEdge(`dep-${s.id}-${dep.id}`, 'dependsOn', s.id, dep.id)
    })
  })

  // Event hooks
  ctrl.onNodeClick((id) => {
    archState.selectedId = id
    _archRenderProps()
  })

  ctrl.onSelectionChange((id) => {
    archState.selectedId = id
    _archRenderProps()
  })

  ctrl.onEdgeClick((edgeId) => {
    if (confirm('Delete this dependency?')) {
      // Remove from state
      const edge = ctrl.edges.find(e => e.id === edgeId)
      if (edge) {
        const svc = archState.services.find(s => s.id === edge.source)
        const tgt = archState.services.find(s => s.id === edge.target)
        if (svc && tgt) {
          svc.depends_on = svc.depends_on.filter(d => d !== tgt.name)
        }
      }
      ctrl.removeEdge(edgeId)
    }
  })

  ctrl.onConnectionComplete((sourceId, targetId) => {
    if (sourceId === targetId) return
    const src = archState.services.find(s => s.id === sourceId)
    const tgt = archState.services.find(s => s.id === targetId)
    if (!src || !tgt) return
    // Check duplicate
    if (src.depends_on.includes(tgt.name)) return
    // Check cycle (simple DFS)
    if (_archWouldCycle(sourceId, targetId)) {
      alert('Cannot add: would create a circular dependency')
      return
    }
    src.depends_on.push(tgt.name)
    ctrl.addEdge(`dep-${sourceId}-${targetId}`, 'dependsOn', sourceId, targetId)
  })

  ctrl.render()
}

function _archWouldCycle(sourceId, targetId) {
  // Would adding sourceId → targetId create a cycle?
  // i.e., can we reach sourceId from targetId via existing edges?
  const visited = new Set()
  function dfs(id) {
    if (id === sourceId) return true
    if (visited.has(id)) return false
    visited.add(id)
    const svc = archState.services.find(s => s.id === id)
    if (!svc) return false
    for (const depName of svc.depends_on) {
      const dep = archState.services.find(s => s.name === depName)
      if (dep && dfs(dep.id)) return true
    }
    return false
  }
  return dfs(targetId)
}

// ── Sidebar ──────────────────────────────────────────────────

function _archRenderSidebar() {
  const el = document.getElementById('arch-template-list')
  el.innerHTML = SERVICE_TEMPLATES.map((t, i) => `
    <button onclick="_archAddFromTemplate(${i})" class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-left hover:bg-indigo-50 transition-colors">
      <span class="text-xl leading-none">${t.icon}</span>
      <div>
        <div class="text-sm font-medium text-gray-900">${t.label}</div>
        <div class="text-xs text-gray-500">${t.pipeline || 'select pipeline'}</div>
      </div>
    </button>
  `).join('')
}

window._archAddFromTemplate = function (idx) {
  const t = SERVICE_TEMPLATES[idx]
  const id = 'svc-' + archState.nextId++
  const name = _archUniqueName(t.label)
  const pt = archPipelines.find(p => p.name === t.pipeline)

  const svc = {
    id,
    name,
    pipeline: t.pipeline,
    repository: '',
    depends_on: []
  }
  archState.services.push(svc)

  // Auto-layout: find free spot
  const existingNodes = Object.values(archState.canvas.nodes)
  let x = 80, y = 60
  if (existingNodes.length) {
    const maxY = Math.max(...existingNodes.map(n => n.y))
    y = maxY + 140
  }

  archState.canvas.addNode(id, 'service', x, y, {
    name,
    pipeline: t.pipeline,
    repository: '',
    deliverables: pt ? pt.deliverables : []
  })

  archState.selectedId = id
  archState.canvas._setSelected(id)
  _archRenderProps()
}

function _archUniqueName(templateLabel) {
  const base = templateLabel.replace(/[^a-zA-Z0-9]/g, '-').toLowerCase().replace(/-+/g, '-').replace(/^-|-$/g, '')
  const prefix = base.substring(0, 20) || 'service'
  let name = prefix
  let counter = 1
  while (archState.services.some(s => s.name === name)) {
    name = prefix + '-' + counter++
  }
  return name
}

// ── Properties Panel ─────────────────────────────────────────

function _archRenderProps() {
  const el = document.getElementById('arch-props-panel')
  if (!archState.selectedId) {
    el.innerHTML = '<p class="text-sm text-gray-400 text-center py-8">Click a service node to edit</p>'
    return
  }

  const svc = archState.services.find(s => s.id === archState.selectedId)
  if (!svc) {
    el.innerHTML = ''
    return
  }

  const pt = archPipelines.find(p => p.name === svc.pipeline)
  const deliverables = pt ? pt.deliverables : []
  const pipeOpts = archPipelines.map(p => `<option value="${p.name}" ${svc.pipeline === p.name ? 'selected' : ''}>${p.name}</option>`).join('')

  el.innerHTML = `
    <div class="space-y-3">
      <div>
        <label class="block text-xs font-medium text-gray-700 mb-1">Service Name</label>
        <input id="arch-prop-name" class="block w-full rounded-md border-0 py-1.5 text-sm shadow-sm ring-1 ring-inset ring-gray-300 focus:ring-2 focus:ring-indigo-600" value="${edEsc(svc.name)}">
      </div>
      <div>
        <label class="block text-xs font-medium text-gray-700 mb-1">Pipeline</label>
        <select id="arch-prop-pipeline" class="block w-full rounded-md border-0 py-1.5 text-sm shadow-sm ring-1 ring-inset ring-gray-300 focus:ring-2 focus:ring-indigo-600">
          <option value="">-- select --</option>${pipeOpts}
        </select>
      </div>
      <div>
        <label class="block text-xs font-medium text-gray-700 mb-1">Repository</label>
        <input id="arch-prop-repo" class="block w-full rounded-md border-0 py-1.5 text-sm shadow-sm ring-1 ring-inset ring-gray-300 focus:ring-2 focus:ring-indigo-600" placeholder="https://github.com/..." value="${edEsc(svc.repository)}">
      </div>
      <div>
        <label class="block text-xs font-medium text-gray-700 mb-1">Deliverables</label>
        <div class="flex flex-wrap gap-1">
          ${deliverables.length ? deliverables.map(d => `<span class="inline-flex items-center rounded-full bg-amber-100 text-amber-800 px-2 py-0.5 text-xs font-medium">${d}</span>`).join('') : '<span class="text-xs text-gray-400">Select a pipeline first</span>'}
        </div>
      </div>
      <div class="pt-2 border-t border-gray-200">
        <button onclick="_archDeleteService()" class="text-sm font-medium text-red-600 hover:text-red-500">Delete Service</button>
      </div>
    </div>`

  // Bind change events
  setTimeout(() => {
    const nameInput = document.getElementById('arch-prop-name')
    const pipeSelect = document.getElementById('arch-prop-pipeline')
    const repoInput = document.getElementById('arch-prop-repo')

    if (nameInput) nameInput.addEventListener('change', (e) => {
      const oldName = svc.name
      svc.name = e.target.value
      // Update depends_on references in other services
      archState.services.forEach(s => {
        s.depends_on = s.depends_on.map(d => d === oldName ? svc.name : d)
      })
      archState.canvas.updateNode(svc.id, { name: svc.name })
    })

    if (pipeSelect) pipeSelect.addEventListener('change', (e) => {
      svc.pipeline = e.target.value
      const pt = archPipelines.find(p => p.name === svc.pipeline)
      archState.canvas.updateNode(svc.id, { pipeline: svc.pipeline, deliverables: pt ? pt.deliverables : [] })
      _archRenderProps() // re-render to show updated deliverables
    })

    if (repoInput) repoInput.addEventListener('change', (e) => {
      svc.repository = e.target.value
      archState.canvas.updateNode(svc.id, { repository: svc.repository })
    })
  }, 0)
}

window._archDeleteService = function () {
  if (!archState.selectedId) return
  const svc = archState.services.find(s => s.id === archState.selectedId)
  if (!svc) return
  if (!confirm(`Delete service "${svc.name}"?`)) return

  // Remove from depends_on of other services
  archState.services.forEach(s => {
    s.depends_on = s.depends_on.filter(d => d !== svc.name)
  })
  archState.services = archState.services.filter(s => s.id !== archState.selectedId)
  archState.canvas.removeNode(archState.selectedId)
  archState.selectedId = null
  _archRenderProps()
}

// ── Footer ───────────────────────────────────────────────────

function _archRenderFooter() {
  const footer = document.getElementById('arch-editor-footer')
  const isCreate = archState.mode === 'create'

  footer.innerHTML = `
    <div class="flex items-center gap-3 flex-1">
      ${isCreate ? `
        <label class="text-sm font-medium text-gray-700">App Name</label>
        <input id="arch-app-name" class="rounded-md border-0 py-1.5 px-3 text-sm shadow-sm ring-1 ring-inset ring-gray-300 focus:ring-2 focus:ring-indigo-600 w-40" placeholder="my-app" value="${edEsc(archState.appName)}">
      ` : ''}
      <label class="text-sm font-medium text-gray-700">Description</label>
      <input id="arch-app-desc" class="rounded-md border-0 py-1.5 px-3 text-sm shadow-sm ring-1 ring-inset ring-gray-300 focus:ring-2 focus:ring-indigo-600 flex-1" placeholder="Application description" value="${edEsc(archState.description)}">
    </div>
    <div id="arch-save-error" class="text-sm text-red-600 mx-3"></div>
    <div class="flex gap-3">
      <button onclick="closeArchEditor()" class="inline-flex items-center rounded-md bg-white px-4 py-2 text-sm font-semibold text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 hover:bg-gray-50">Cancel</button>
      <button onclick="_archSave()" id="arch-save-btn" class="inline-flex items-center rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500">Save</button>
    </div>`

  // Bind inputs
  setTimeout(() => {
    const nameInput = document.getElementById('arch-app-name')
    const descInput = document.getElementById('arch-app-desc')
    if (nameInput) nameInput.addEventListener('input', (e) => { archState.appName = e.target.value })
    if (descInput) descInput.addEventListener('input', (e) => { archState.description = e.target.value })
  }, 0)
}

window._archSave = async function () {
  const errEl = document.getElementById('arch-save-error')
  const btn = document.getElementById('arch-save-btn')
  errEl.textContent = ''

  // Validate
  if (archState.mode === 'create' && !archState.appName) { errEl.textContent = 'App name is required'; return }
  if (!archState.services.length) { errEl.textContent = 'Add at least one service'; return }
  for (const s of archState.services) {
    if (!s.name) { errEl.textContent = 'All services must have a name'; return }
    if (!s.pipeline) { errEl.textContent = `Service "${s.name}" must select a pipeline`; return }
  }

  btn.textContent = 'Saving...'; btn.disabled = true

  const archBody = {
    description: archState.description,
    services: archState.services.map(s => ({
      name: s.name,
      pipeline: s.pipeline,
      repository: s.repository,
      depends_on: s.depends_on
    }))
  }

  try {
    if (archState.mode === 'create') {
      await api('POST', '/apps', {}, { name: archState.appName, arch: archBody })
    } else {
      await api('PUT', `/apps/${encodeURIComponent(selectedApp)}/arch`, {}, archBody)
    }

    closeArchEditor()
    // Refresh app list
    const apps = await api('GET', '/apps')
    const sel = document.getElementById('sel-app')
    sel.innerHTML = apps.map(a => `<option value="${a}">${a}</option>`).join('')
    const target = archState.mode === 'create' ? archState.appName : selectedApp
    sel.value = target
    selectedApp = target
    await loadEnvs()
  } catch (e) {
    errEl.textContent = e.message
    btn.textContent = 'Save'; btn.disabled = false
  }
}
