// ════════════════════════════════════════════════════════════════
// deploy-editor.js — Deployment Environment Visual Editor
// ════════════════════════════════════════════════════════════════

const INFRA_TEMPLATES = [
  { label: 'MySQL',      icon: '\uD83D\uDDC4\uFE0F', type: 'mysql:8.0',    via: 'docker', image: 'mysql:8.0',                   defaultEnv: { MYSQL_ROOT_PASSWORD: 'root' }, defaultConfig: { port: 3306 } },
  { label: 'PostgreSQL', icon: '\uD83D\uDC18',       type: 'postgres:16',   via: 'docker', image: 'postgres:16',                  defaultEnv: { POSTGRES_PASSWORD: 'postgres' }, defaultConfig: { port: 5432 } },
  { label: 'Redis',      icon: '\u26A1',             type: 'redis:7',       via: 'docker', image: 'redis:7-alpine',               defaultEnv: {},                              defaultConfig: { port: 6379 } },
  { label: 'Kafka',      icon: '\uD83D\uDCE8',       type: 'kafka:3.6',     via: 'docker', image: 'confluentinc/cp-kafka:7.5.0',  defaultEnv: {},                              defaultConfig: { port: 9092 } },
  { label: 'Custom',     icon: '\uD83D\uDCE6',       type: '',              via: 'docker', image: '',                             defaultEnv: {},                              defaultConfig: {} },
]

let deployState = {
  mode: null,
  appName: '',
  envName: '',
  environment: 'development',
  archServices: [],   // from arch (read-only structure)
  serviceSpecs: [],   // [{ name, accepts, compute: { type, ports, resources } }]
  infraNodes: [],     // [{ id, name, type, provision, resources, config }]
  bindings: [],       // [{ id, service, infraId, env: {} }]
  ingress: null,      // { id, name, type, bind, routes, resources }
  canvas: null,
  selectedId: null,
  nextId: 1,
}

let deployPipelines = [] // cached

function initDeployEditor() {
  document.getElementById('btn-new-env').addEventListener('click', () => openDeployEditor('create'))
  document.getElementById('btn-edit-env').addEventListener('click', () => openDeployEditor('edit'))
}

async function openDeployEditor(mode) {
  if (!selectedApp) { alert('Select an app first'); return }

  deployState.mode = mode
  deployState.appName = selectedApp
  deployState.envName = mode === 'edit' ? selectedEnv : ''
  deployState.environment = 'development'
  deployState.archServices = []
  deployState.serviceSpecs = []
  deployState.infraNodes = []
  deployState.bindings = []
  deployState.ingress = null
  deployState.selectedId = null
  deployState.nextId = 1

  // Load pipelines
  if (!deployPipelines.length) {
    try { deployPipelines = await api('GET', '/pipelines') } catch (e) { alert('Failed: ' + e.message); return }
  }

  // Load architecture
  try {
    const arch = await api('GET', `/apps/${encodeURIComponent(selectedApp)}/arch`)
    deployState.archServices = (arch.Services || []).map(s => ({
      name: s.Name, pipeline: s.Pipeline, repository: s.Repository, depends_on: s.DependsOn || []
    }))
  } catch (e) { alert('Failed to load architecture: ' + e.message); return }

  if (!deployState.archServices.length) {
    alert('App has no services defined. Edit the architecture first.')
    return
  }

  // Init service specs with defaults
  deployState.serviceSpecs = deployState.archServices.map(s => {
    const pt = deployPipelines.find(p => p.name === s.pipeline)
    return {
      name: s.name,
      accepts: pt && pt.deliverables.length ? pt.deliverables[pt.deliverables.length - 1] : '',
      compute: { type: 'docker-container', ports: [8080], resources: { cpu: '1', memory: '512Mi', storage: '', replicas: 1 } }
    }
  })

  // Load existing env data in edit mode
  if (mode === 'edit' && selectedEnv) {
    try {
      const env = await api('GET', `/apps/${encodeURIComponent(selectedApp)}/envs/${encodeURIComponent(selectedEnv)}`)
      deployState.envName = selectedEnv
      deployState.environment = env.Environment || 'development'

      // Merge service specs
      for (const es of (env.Services || [])) {
        const spec = deployState.serviceSpecs.find(s => s.name === es.Name)
        if (spec) {
          spec.accepts = es.Accepts
          if (es.Compute) {
            spec.compute = {
              type: es.Compute.Type,
              ports: es.Compute.Ports || [],
              resources: es.Compute.Resources ? {
                cpu: es.Compute.Resources.CPU, memory: es.Compute.Resources.Memory,
                storage: es.Compute.Resources.Storage, replicas: es.Compute.Resources.Replicas || 1
              } : spec.compute.resources
            }
          }
        }
      }

      // Load infra
      deployState.infraNodes = (env.Dependencies || []).map((d, i) => ({
        id: 'infra-' + (i + 1),
        name: d.Name, type: d.Type,
        provision: d.Provision ? { via: d.Provision.Via, image: d.Provision.Image || '', env: d.Provision.Env || {} } : { via: 'docker', image: '', env: {} },
        resources: d.Resources ? { cpu: d.Resources.CPU, memory: d.Resources.Memory, storage: d.Resources.Storage } : {},
        config: d.Config || {}
      }))
      deployState.nextId = deployState.infraNodes.length + 1

      // Load bindings
      let bindId = 1
      deployState.bindings = (env.Bindings || []).map(b => {
        const infraNode = deployState.infraNodes.find(n => {
          // Try to match binding to infra by checking env var interpolation
          return Object.values(b.Env || {}).some(v => typeof v === 'string' && v.includes('${' + n.name))
        })
        return {
          id: 'bind-' + bindId++,
          service: b.Service,
          infraId: infraNode ? infraNode.id : '',
          env: b.Env || {}
        }
      })

      // Load ingress
      if (env.Network && env.Network.Ingress && env.Network.Ingress.length) {
        const ig = env.Network.Ingress[0]
        deployState.ingress = {
          id: 'ingress-1',
          name: ig.Name, type: ig.Type || 'nginx',
          bind: ig.Bind ? { ip: ig.Bind.IP, http: ig.Bind.HTTP, https: ig.Bind.HTTPS } : { ip: '0.0.0.0', http: 80, https: 0 },
          routes: (ig.Routes || []).map(r => ({ path: r.Path, service: r.Service, port: r.Port })),
          resources: ig.Resources ? { cpu: ig.Resources.CPU, memory: ig.Resources.Memory } : { cpu: '0.5', memory: '256Mi' }
        }
      }
    } catch (e) { alert('Failed to load env: ' + e.message); return }
    document.getElementById('deploy-editor-title').textContent = 'Edit Environment: ' + selectedEnv
  } else {
    document.getElementById('deploy-editor-title').textContent = 'New Environment for ' + selectedApp
  }

  document.getElementById('deploy-editor-overlay').classList.remove('hidden')
  _deployInitCanvas()
  _deployRenderServicePanel()
  _deployRenderInfraSidebar()
  _deployRenderFooter()
}

function closeDeployEditor() {
  document.getElementById('deploy-editor-overlay').classList.add('hidden')
  deployState.canvas = null
}

// ── Canvas ───────────────────────────────────────────────────

function _deployInitCanvas() {
  const wrap = document.getElementById('deploy-canvas-wrap')
  wrap.innerHTML = ''
  const ctrl = createCanvasController(wrap, { portMode: 'side' })
  deployState.canvas = ctrl

  // Place service nodes (left column, fixed x)
  deployState.archServices.forEach((s, i) => {
    ctrl.addNode('svc-' + s.name, 'service', 60, 40 + i * 130, {
      name: s.name,
      pipeline: s.pipeline,
      deliverables: (() => { const pt = deployPipelines.find(p => p.name === s.pipeline); return pt ? pt.deliverables : [] })()
    }, false) // not fixed, allow drag for layout
  })

  // Place infra nodes (right column)
  deployState.infraNodes.forEach((inf, i) => {
    ctrl.addNode(inf.id, 'infra', 500, 40 + i * 120, {
      name: inf.name, type: inf.type, via: inf.provision.via
    })
  })

  // Place ingress if exists
  if (deployState.ingress) {
    const yOffset = Math.max(deployState.archServices.length, deployState.infraNodes.length) * 120 + 60
    ctrl.addNode(deployState.ingress.id, 'ingress', 280, yOffset, {
      name: deployState.ingress.name,
      type: deployState.ingress.type,
      routes: deployState.ingress.routes
    })
    // Draw route edges
    deployState.ingress.routes.forEach((r, i) => {
      const svcNode = ctrl.nodes['svc-' + r.service]
      if (svcNode) {
        ctrl.addEdge(`route-${i}`, 'route', deployState.ingress.id, 'svc-' + r.service, r.path)
      }
    })
  }

  // Draw binding edges
  deployState.bindings.forEach(b => {
    if (b.infraId && ctrl.nodes[b.infraId] && ctrl.nodes['svc-' + b.service]) {
      ctrl.addEdge(b.id, 'binding', b.infraId, 'svc-' + b.service)
    }
  })

  // Event hooks
  ctrl.onNodeClick((id, node) => {
    deployState.selectedId = id
    if (node.type === 'infra') {
      _deployShowInfraConfig(id)
    } else if (node.type === 'ingress') {
      _deployShowIngressConfig()
    }
  })

  ctrl.onSelectionChange((id) => { deployState.selectedId = id })

  ctrl.onEdgeClick((edgeId) => {
    const binding = deployState.bindings.find(b => b.id === edgeId)
    if (binding) {
      _deployShowBindingConfig(binding)
      return
    }
    // Route edge click
    if (edgeId.startsWith('route-')) {
      // Just select the ingress for now
      return
    }
  })

  ctrl.onConnectionComplete((sourceId, targetId) => {
    const srcNode = ctrl.nodes[sourceId]
    const tgtNode = ctrl.nodes[targetId]
    if (!srcNode || !tgtNode) return

    // infra → service = binding
    if (srcNode.type === 'infra' && tgtNode.type === 'service') {
      _deployCreateBinding(sourceId, tgtNode.data.name)
    } else if (srcNode.type === 'service' && tgtNode.type === 'infra') {
      _deployCreateBinding(targetId, srcNode.data.name)
    }
  })

  ctrl.render()
}

function _deployCreateBinding(infraId, serviceName) {
  // Check duplicate
  if (deployState.bindings.find(b => b.infraId === infraId && b.service === serviceName)) return

  const infra = deployState.infraNodes.find(n => n.id === infraId)
  if (!infra) return

  const bindId = 'bind-' + Date.now()
  const defaultEnv = {}

  // Auto-generate common binding env vars based on infra type
  if (infra.type.startsWith('mysql')) {
    defaultEnv['DB_HOST'] = '${' + infra.name + '.host}'
    defaultEnv['DB_PORT'] = '${' + infra.name + '.config.port}'
  } else if (infra.type.startsWith('postgres')) {
    defaultEnv['DB_HOST'] = '${' + infra.name + '.host}'
    defaultEnv['DB_PORT'] = '${' + infra.name + '.config.port}'
  } else if (infra.type.startsWith('redis')) {
    defaultEnv['REDIS_HOST'] = '${' + infra.name + '.host}'
    defaultEnv['REDIS_PORT'] = '${' + infra.name + '.config.port}'
  } else if (infra.type.startsWith('kafka')) {
    defaultEnv['KAFKA_BROKERS'] = '${' + infra.name + '.host}:${' + infra.name + '.config.port}'
  }

  const binding = { id: bindId, service: serviceName, infraId, env: defaultEnv }
  deployState.bindings.push(binding)
  deployState.canvas.addEdge(bindId, 'binding', infraId, 'svc-' + serviceName)

  // Show binding config
  _deployShowBindingConfig(binding)
}

// ── Service Panel (Left) ─────────────────────────────────────

function _deployRenderServicePanel() {
  const el = document.getElementById('deploy-service-list')
  el.innerHTML = deployState.serviceSpecs.map((s, i) => {
    const archSvc = deployState.archServices.find(a => a.name === s.name)
    const pt = archSvc ? deployPipelines.find(p => p.name === archSvc.pipeline) : null
    const delOpts = (pt ? pt.deliverables : []).map(d => `<option value="${d}" ${s.accepts === d ? 'selected' : ''}>${d}</option>`).join('')
    const compTypes = ['docker-container', 'kubernetes-pod', 'nginx-static', 'vm']
    const compOpts = compTypes.map(t => `<option value="${t}" ${s.compute.type === t ? 'selected' : ''}>${t}</option>`).join('')
    const r = s.compute.resources

    return `<div class="border-b border-gray-200 pb-3 mb-3 last:border-0">
      <div class="text-sm font-semibold text-indigo-700 mb-2">${edEsc(s.name)}</div>
      <div class="space-y-1.5">
        <div class="flex items-center gap-2">
          <label class="text-xs text-gray-500 w-16 shrink-0">accepts</label>
          <select class="flex-1 rounded border-0 py-1 text-xs ring-1 ring-gray-300 focus:ring-indigo-600" onchange="deployState.serviceSpecs[${i}].accepts=this.value">${delOpts}</select>
        </div>
        <div class="flex items-center gap-2">
          <label class="text-xs text-gray-500 w-16 shrink-0">compute</label>
          <select class="flex-1 rounded border-0 py-1 text-xs ring-1 ring-gray-300 focus:ring-indigo-600" onchange="deployState.serviceSpecs[${i}].compute.type=this.value">${compOpts}</select>
        </div>
        <div class="flex items-center gap-2">
          <label class="text-xs text-gray-500 w-16 shrink-0">ports</label>
          <input class="flex-1 rounded border-0 py-1 text-xs ring-1 ring-gray-300 focus:ring-indigo-600" value="${(s.compute.ports || []).join(',')}" onchange="deployState.serviceSpecs[${i}].compute.ports=this.value.split(',').map(Number).filter(n=>n)">
        </div>
        <div class="grid grid-cols-3 gap-1.5">
          <div><label class="text-[10px] text-gray-500">CPU</label><input class="w-full rounded border-0 py-1 text-xs ring-1 ring-gray-300 focus:ring-indigo-600" value="${r.cpu||''}" onchange="deployState.serviceSpecs[${i}].compute.resources.cpu=this.value"></div>
          <div><label class="text-[10px] text-gray-500">Mem</label><input class="w-full rounded border-0 py-1 text-xs ring-1 ring-gray-300 focus:ring-indigo-600" value="${r.memory||''}" onchange="deployState.serviceSpecs[${i}].compute.resources.memory=this.value"></div>
          <div><label class="text-[10px] text-gray-500">Replicas</label><input type="number" class="w-full rounded border-0 py-1 text-xs ring-1 ring-gray-300 focus:ring-indigo-600" value="${r.replicas||1}" onchange="deployState.serviceSpecs[${i}].compute.resources.replicas=+this.value"></div>
        </div>
      </div>
    </div>`
  }).join('')
}

// ── Infra Sidebar (Right) ────────────────────────────────────

function _deployRenderInfraSidebar() {
  const el = document.getElementById('deploy-infra-list')
  el.innerHTML = INFRA_TEMPLATES.map((t, i) => `
    <button onclick="_deployAddInfra(${i})" class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-left hover:bg-purple-50 transition-colors">
      <span class="text-lg leading-none">${t.icon}</span>
      <div>
        <div class="text-sm font-medium text-gray-900">${t.label}</div>
        <div class="text-xs text-gray-500">${t.type || 'custom'}</div>
      </div>
    </button>
  `).join('')

  // Add ingress button
  el.innerHTML += `
    <div class="mt-3 pt-3 border-t border-gray-200">
      <button onclick="_deployAddIngress()" class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-left hover:bg-orange-50 transition-colors">
        <span class="text-lg leading-none">\uD83C\uDF10</span>
        <div>
          <div class="text-sm font-medium text-gray-900">Ingress</div>
          <div class="text-xs text-gray-500">nginx / traefik</div>
        </div>
      </button>
    </div>`
}

window._deployAddInfra = function (idx) {
  const t = INFRA_TEMPLATES[idx]
  const id = 'infra-' + deployState.nextId++
  const name = t.label.toLowerCase().replace(/\s+/g, '-')

  // Unique name
  let finalName = name
  let counter = 1
  while (deployState.infraNodes.find(n => n.name === finalName)) {
    finalName = name + '-' + counter++
  }

  const infra = {
    id,
    name: finalName,
    type: t.type,
    provision: { via: t.via, image: t.image, env: { ...t.defaultEnv } },
    resources: { cpu: '1', memory: '1Gi', storage: '' },
    config: { ...t.defaultConfig }
  }
  deployState.infraNodes.push(infra)

  // Auto-layout on right side
  const infraCount = Object.values(deployState.canvas.nodes).filter(n => n.type === 'infra').length
  deployState.canvas.addNode(id, 'infra', 500, 40 + infraCount * 120, {
    name: finalName, type: t.type, via: t.via
  })
}

window._deployAddIngress = function () {
  if (deployState.ingress) { alert('Ingress already exists'); return }

  const id = 'ingress-1'
  deployState.ingress = {
    id,
    name: 'gateway',
    type: 'nginx',
    bind: { ip: '0.0.0.0', http: 80, https: 0 },
    routes: [],
    resources: { cpu: '0.5', memory: '256Mi' }
  }

  const yOffset = Math.max(deployState.archServices.length, deployState.infraNodes.length) * 120 + 60
  deployState.canvas.addNode(id, 'ingress', 280, yOffset, {
    name: 'gateway', type: 'nginx', routes: []
  })
}

// ── Config Dialogs ───────────────────────────────────────────

function _deployShowInfraConfig(infraId) {
  const infra = deployState.infraNodes.find(n => n.id === infraId)
  if (!infra) return

  const modal = document.getElementById('deploy-config-modal')
  const envPairs = Object.entries(infra.provision.env || {}).map(([k, v]) => `${k}=${v}`).join('\n')
  const cfgPairs = Object.entries(infra.config || {}).map(([k, v]) => `${k}=${v}`).join('\n')
  const viaOpts = ['docker', 'terraform', 'helm', 'external'].map(v => `<option value="${v}" ${infra.provision.via === v ? 'selected' : ''}>${v}</option>`).join('')

  modal.innerHTML = `
    <div class="bg-white rounded-lg shadow-xl p-5 w-96 max-h-[80vh] overflow-y-auto">
      <h3 class="text-sm font-semibold text-gray-900 mb-3">Configure: ${edEsc(infra.name)}</h3>
      <div class="space-y-3">
        <div class="grid grid-cols-2 gap-2">
          <div><label class="text-xs font-medium text-gray-700">Name</label><input id="dcfg-name" class="w-full rounded border-0 py-1.5 text-sm ring-1 ring-gray-300 focus:ring-indigo-600" value="${edEsc(infra.name)}"></div>
          <div><label class="text-xs font-medium text-gray-700">Type</label><input id="dcfg-type" class="w-full rounded border-0 py-1.5 text-sm ring-1 ring-gray-300 focus:ring-indigo-600" value="${edEsc(infra.type)}"></div>
        </div>
        <div class="grid grid-cols-2 gap-2">
          <div><label class="text-xs font-medium text-gray-700">Via</label><select id="dcfg-via" class="w-full rounded border-0 py-1.5 text-sm ring-1 ring-gray-300 focus:ring-indigo-600">${viaOpts}</select></div>
          <div><label class="text-xs font-medium text-gray-700">Image</label><input id="dcfg-image" class="w-full rounded border-0 py-1.5 text-sm ring-1 ring-gray-300 focus:ring-indigo-600" value="${edEsc(infra.provision.image)}"></div>
        </div>
        <div class="grid grid-cols-3 gap-2">
          <div><label class="text-xs font-medium text-gray-700">CPU</label><input id="dcfg-cpu" class="w-full rounded border-0 py-1.5 text-sm ring-1 ring-gray-300 focus:ring-indigo-600" value="${infra.resources.cpu||''}"></div>
          <div><label class="text-xs font-medium text-gray-700">Memory</label><input id="dcfg-mem" class="w-full rounded border-0 py-1.5 text-sm ring-1 ring-gray-300 focus:ring-indigo-600" value="${infra.resources.memory||''}"></div>
          <div><label class="text-xs font-medium text-gray-700">Storage</label><input id="dcfg-sto" class="w-full rounded border-0 py-1.5 text-sm ring-1 ring-gray-300 focus:ring-indigo-600" value="${infra.resources.storage||''}"></div>
        </div>
        <div><label class="text-xs font-medium text-gray-700">Env Vars (K=V per line)</label><textarea id="dcfg-env" class="w-full rounded border-0 py-1.5 text-xs ring-1 ring-gray-300 focus:ring-indigo-600 h-16">${envPairs}</textarea></div>
        <div><label class="text-xs font-medium text-gray-700">Config (K=V per line)</label><textarea id="dcfg-cfg" class="w-full rounded border-0 py-1.5 text-xs ring-1 ring-gray-300 focus:ring-indigo-600 h-16">${cfgPairs}</textarea></div>
      </div>
      <div class="flex justify-between mt-4">
        <button onclick="_deployDeleteInfra('${infraId}')" class="text-sm text-red-600 hover:text-red-500">Delete</button>
        <div class="flex gap-2">
          <button onclick="_deployCloseConfig()" class="px-3 py-1.5 text-sm rounded ring-1 ring-gray-300 hover:bg-gray-50">Cancel</button>
          <button onclick="_deploySaveInfraConfig('${infraId}')" class="px-3 py-1.5 text-sm rounded bg-indigo-600 text-white hover:bg-indigo-500">OK</button>
        </div>
      </div>
    </div>`
  modal.classList.remove('hidden')
}

window._deploySaveInfraConfig = function (infraId) {
  const infra = deployState.infraNodes.find(n => n.id === infraId)
  if (!infra) return

  const oldName = infra.name
  infra.name = document.getElementById('dcfg-name').value
  infra.type = document.getElementById('dcfg-type').value
  infra.provision.via = document.getElementById('dcfg-via').value
  infra.provision.image = document.getElementById('dcfg-image').value
  infra.resources.cpu = document.getElementById('dcfg-cpu').value
  infra.resources.memory = document.getElementById('dcfg-mem').value
  infra.resources.storage = document.getElementById('dcfg-sto').value
  infra.provision.env = _deployParseKV(document.getElementById('dcfg-env').value)
  infra.config = _deployParseKVAny(document.getElementById('dcfg-cfg').value)

  // Update canvas node
  deployState.canvas.updateNode(infraId, { name: infra.name, type: infra.type, via: infra.provision.via })

  // Update binding env vars if name changed
  if (oldName !== infra.name) {
    deployState.bindings.forEach(b => {
      if (b.infraId === infraId) {
        const newEnv = {}
        for (const [k, v] of Object.entries(b.env)) {
          newEnv[k] = typeof v === 'string' ? v.replace(new RegExp('\\$\\{' + oldName, 'g'), '${' + infra.name) : v
        }
        b.env = newEnv
      }
    })
  }

  _deployCloseConfig()
}

window._deployDeleteInfra = function (infraId) {
  if (!confirm('Delete this infrastructure resource?')) return
  deployState.infraNodes = deployState.infraNodes.filter(n => n.id !== infraId)
  deployState.bindings = deployState.bindings.filter(b => b.infraId !== infraId)
  deployState.canvas.removeNode(infraId)
  _deployCloseConfig()
}

function _deployShowBindingConfig(binding) {
  const modal = document.getElementById('deploy-config-modal')
  const envPairs = Object.entries(binding.env || {}).map(([k, v]) => `${k}=${v}`).join('\n')
  const infra = deployState.infraNodes.find(n => n.id === binding.infraId)

  modal.innerHTML = `
    <div class="bg-white rounded-lg shadow-xl p-5 w-96">
      <h3 class="text-sm font-semibold text-gray-900 mb-1">Binding: ${edEsc(binding.service)} \u2190 ${edEsc(infra ? infra.name : '?')}</h3>
      <p class="text-xs text-gray-500 mb-3">Use \${${edEsc(infra ? infra.name : 'name')}.field} for interpolation</p>
      <div>
        <label class="text-xs font-medium text-gray-700">Environment Variables (K=V per line)</label>
        <textarea id="bcfg-env" class="w-full rounded border-0 py-1.5 text-xs ring-1 ring-gray-300 focus:ring-indigo-600 h-28 font-mono">${envPairs}</textarea>
      </div>
      <div class="flex justify-between mt-4">
        <button onclick="_deployDeleteBinding('${binding.id}')" class="text-sm text-red-600 hover:text-red-500">Delete</button>
        <div class="flex gap-2">
          <button onclick="_deployCloseConfig()" class="px-3 py-1.5 text-sm rounded ring-1 ring-gray-300 hover:bg-gray-50">Cancel</button>
          <button onclick="_deploySaveBindingConfig('${binding.id}')" class="px-3 py-1.5 text-sm rounded bg-indigo-600 text-white hover:bg-indigo-500">OK</button>
        </div>
      </div>
    </div>`
  modal.classList.remove('hidden')
}

window._deploySaveBindingConfig = function (bindId) {
  const binding = deployState.bindings.find(b => b.id === bindId)
  if (!binding) return
  binding.env = _deployParseKV(document.getElementById('bcfg-env').value)
  _deployCloseConfig()
}

window._deployDeleteBinding = function (bindId) {
  deployState.bindings = deployState.bindings.filter(b => b.id !== bindId)
  deployState.canvas.removeEdge(bindId)
  _deployCloseConfig()
}

function _deployShowIngressConfig() {
  if (!deployState.ingress) return
  const ig = deployState.ingress
  const modal = document.getElementById('deploy-config-modal')
  const typeOpts = ['nginx', 'traefik', 'haproxy'].map(t => `<option value="${t}" ${ig.type === t ? 'selected' : ''}>${t}</option>`).join('')
  const routeLines = ig.routes.map(r => `${r.path} ${r.service}:${r.port}`).join('\n')

  modal.innerHTML = `
    <div class="bg-white rounded-lg shadow-xl p-5 w-96">
      <h3 class="text-sm font-semibold text-gray-900 mb-3">Ingress Configuration</h3>
      <div class="space-y-3">
        <div class="grid grid-cols-2 gap-2">
          <div><label class="text-xs font-medium text-gray-700">Name</label><input id="icfg-name" class="w-full rounded border-0 py-1.5 text-sm ring-1 ring-gray-300 focus:ring-indigo-600" value="${edEsc(ig.name)}"></div>
          <div><label class="text-xs font-medium text-gray-700">Type</label><select id="icfg-type" class="w-full rounded border-0 py-1.5 text-sm ring-1 ring-gray-300 focus:ring-indigo-600">${typeOpts}</select></div>
        </div>
        <div><label class="text-xs font-medium text-gray-700">Bind (IP:HTTP:HTTPS)</label><input id="icfg-bind" class="w-full rounded border-0 py-1.5 text-sm ring-1 ring-gray-300 focus:ring-indigo-600" value="${ig.bind.ip}:${ig.bind.http}:${ig.bind.https}"></div>
        <div><label class="text-xs font-medium text-gray-700">Routes (path service:port, one per line)</label><textarea id="icfg-routes" class="w-full rounded border-0 py-1.5 text-xs ring-1 ring-gray-300 focus:ring-indigo-600 h-20 font-mono">${routeLines}</textarea></div>
      </div>
      <div class="flex justify-between mt-4">
        <button onclick="_deployDeleteIngress()" class="text-sm text-red-600 hover:text-red-500">Delete</button>
        <div class="flex gap-2">
          <button onclick="_deployCloseConfig()" class="px-3 py-1.5 text-sm rounded ring-1 ring-gray-300 hover:bg-gray-50">Cancel</button>
          <button onclick="_deploySaveIngressConfig()" class="px-3 py-1.5 text-sm rounded bg-indigo-600 text-white hover:bg-indigo-500">OK</button>
        </div>
      </div>
    </div>`
  modal.classList.remove('hidden')
}

window._deploySaveIngressConfig = function () {
  const ig = deployState.ingress
  if (!ig) return
  ig.name = document.getElementById('icfg-name').value
  ig.type = document.getElementById('icfg-type').value
  const bindParts = document.getElementById('icfg-bind').value.split(':')
  ig.bind = { ip: bindParts[0] || '0.0.0.0', http: parseInt(bindParts[1]) || 80, https: parseInt(bindParts[2]) || 0 }

  // Parse routes and update canvas edges
  const oldRoutes = ig.routes
  ig.routes = document.getElementById('icfg-routes').value.split('\n').filter(l => l.trim()).map(line => {
    const [path, rest] = line.trim().split(/\s+/)
    const [svc, port] = (rest || '').split(':')
    return { path: path || '/', service: svc || '', port: parseInt(port) || 80 }
  })

  // Remove old route edges and add new ones
  oldRoutes.forEach((_, i) => deployState.canvas.removeEdge(`route-${i}`))
  ig.routes.forEach((r, i) => {
    if (deployState.canvas.nodes['svc-' + r.service]) {
      deployState.canvas.addEdge(`route-${i}`, 'route', ig.id, 'svc-' + r.service, r.path)
    }
  })

  deployState.canvas.updateNode(ig.id, { name: ig.name, type: ig.type, routes: ig.routes })
  _deployCloseConfig()
}

window._deployDeleteIngress = function () {
  if (!confirm('Delete ingress?')) return
  deployState.canvas.removeNode(deployState.ingress.id)
  deployState.ingress = null
  _deployCloseConfig()
}

window._deployCloseConfig = function () {
  document.getElementById('deploy-config-modal').classList.add('hidden')
}

// ── Footer ───────────────────────────────────────────────────

function _deployRenderFooter() {
  const footer = document.getElementById('deploy-editor-footer')
  const envTypes = ['local', 'development', 'staging', 'production']
  const envOpts = envTypes.map(t => `<option value="${t}" ${deployState.environment === t ? 'selected' : ''}>${t}</option>`).join('')

  footer.innerHTML = `
    <div class="flex items-center gap-3 flex-1">
      <label class="text-sm font-medium text-gray-700">Env Name</label>
      <input id="deploy-env-name" class="rounded-md border-0 py-1.5 px-3 text-sm shadow-sm ring-1 ring-inset ring-gray-300 focus:ring-2 focus:ring-indigo-600 w-28" value="${edEsc(deployState.envName)}" ${deployState.mode === 'edit' ? 'disabled' : ''}>
      <label class="text-sm font-medium text-gray-700">Type</label>
      <select id="deploy-env-type" class="rounded-md border-0 py-1.5 text-sm shadow-sm ring-1 ring-inset ring-gray-300 focus:ring-2 focus:ring-indigo-600">${envOpts}</select>
    </div>
    <div id="deploy-save-error" class="text-sm text-red-600 mx-3"></div>
    <div class="flex gap-3">
      <button onclick="closeDeployEditor()" class="inline-flex items-center rounded-md bg-white px-4 py-2 text-sm font-semibold text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 hover:bg-gray-50">Cancel</button>
      <button onclick="_deploySave()" id="deploy-save-btn" class="inline-flex items-center rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500">Save</button>
    </div>`

  setTimeout(() => {
    const nameInput = document.getElementById('deploy-env-name')
    const typeSelect = document.getElementById('deploy-env-type')
    if (nameInput) nameInput.addEventListener('input', (e) => { deployState.envName = e.target.value })
    if (typeSelect) typeSelect.addEventListener('change', (e) => { deployState.environment = e.target.value })
  }, 0)
}

window._deploySave = async function () {
  const errEl = document.getElementById('deploy-save-error')
  const btn = document.getElementById('deploy-save-btn')
  errEl.textContent = ''

  if (!deployState.envName) { errEl.textContent = 'Environment name is required'; return }
  for (const s of deployState.serviceSpecs) {
    if (!s.accepts) { errEl.textContent = `Service "${s.name}" must have accepts type`; return }
  }

  btn.textContent = 'Saving...'; btn.disabled = true

  // Build body
  const body = {
    envName: deployState.envName,
    environment: deployState.environment,
    services: deployState.serviceSpecs.map(s => ({
      name: s.name,
      accepts: s.accepts,
      compute: {
        type: s.compute.type,
        resources: s.compute.resources,
        ports: s.compute.ports
      }
    })),
    dependencies: deployState.infraNodes.map(n => ({
      name: n.name,
      type: n.type,
      provision: { via: n.provision.via, image: n.provision.image, env: n.provision.env },
      resources: n.resources,
      config: n.config
    })),
    bindings: _deployBuildBindings(),
    network: deployState.ingress ? {
      ingress: [{
        name: deployState.ingress.name,
        type: deployState.ingress.type,
        bind: deployState.ingress.bind,
        routes: deployState.ingress.routes,
        resources: deployState.ingress.resources
      }]
    } : { ingress: [] }
  }

  try {
    if (deployState.mode === 'create') {
      await api('POST', `/apps/${encodeURIComponent(deployState.appName)}/envs`, {}, body)
    } else {
      await api('PUT', `/apps/${encodeURIComponent(deployState.appName)}/envs/${encodeURIComponent(deployState.envName)}`, {}, body)
    }
    closeDeployEditor()
    await loadEnvs()
  } catch (e) {
    errEl.textContent = e.message
    btn.textContent = 'Save'; btn.disabled = false
  }
}

function _deployBuildBindings() {
  // Merge bindings by service
  const byService = {}
  for (const b of deployState.bindings) {
    if (!byService[b.service]) byService[b.service] = {}
    Object.assign(byService[b.service], b.env)
  }
  return Object.entries(byService).map(([service, env]) => ({ service, env }))
}

// ── Helpers ──────────────────────────────────────────────────

function _deployParseKV(text) {
  const map = {}
  for (const line of text.split('\n')) {
    const idx = line.indexOf('=')
    if (idx > 0) map[line.slice(0, idx).trim()] = line.slice(idx + 1).trim()
  }
  return map
}

function _deployParseKVAny(text) {
  const map = {}
  for (const line of text.split('\n')) {
    const idx = line.indexOf('=')
    if (idx > 0) {
      const v = line.slice(idx + 1).trim()
      map[line.slice(0, idx).trim()] = isNaN(v) ? v : Number(v)
    }
  }
  return map
}
