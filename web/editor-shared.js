// ════════════════════════════════════════════════════════════════
// editor-shared.js — Shared canvas primitives for visual editors
// ════════════════════════════════════════════════════════════════

const ED_NODE_W = 220
const ED_NODE_H = { service: 100, infra: 90, ingress: 110 }
const ED_CANVAS_W = 3000
const ED_CANVAS_H = 2000

const ED_COLORS = {
  service: { bg: '#eef2ff', border: '#a5b4fc', title: '#4338ca' },
  infra:   { bg: '#faf5ff', border: '#c4b5fd', title: '#6d28d9' },
  ingress: { bg: '#fff7ed', border: '#fed7aa', title: '#c2410c' },
}

const ED_EDGE_COLORS = {
  dependsOn: { stroke: '#9ca3af', dash: '6 4' },
  binding:   { stroke: '#a855f7', dash: '5 4' },
  route:     { stroke: '#f97316', dash: '' },
}

// ── Utility ──────────────────────────────────────────────────
function edEsc(s) {
  const el = document.createElement('div')
  el.textContent = s
  return el.innerHTML
}

function edRepoShort(url) {
  if (!url) return '\u2014'
  try { return new URL(url).pathname.replace(/^\//, '').replace(/\.git$/, '') } catch { return url }
}

// ── Canvas Controller ────────────────────────────────────────
function createCanvasController(wrapEl, opts = {}) {
  const ctrl = {
    wrapEl,
    nodes: {},       // id -> { id, type, x, y, data, fixed }
    edges: [],       // [{ id, type, source, target, label }]
    transform: { x: 20, y: 20, scale: 1 },
    selectedId: null,
    _panState: null,
    _dragState: null,
    _connState: null, // { sourceId, startX, startY }
    _onNodeClick: null,
    _onEdgeClick: null,
    _onConnectionComplete: null,
    _onSelectionChange: null,
    _portMode: opts.portMode || 'bottom', // 'bottom' for arch, 'side' for deploy

    onNodeClick(cb) { this._onNodeClick = cb },
    onEdgeClick(cb) { this._onEdgeClick = cb },
    onConnectionComplete(cb) { this._onConnectionComplete = cb },
    onSelectionChange(cb) { this._onSelectionChange = cb },

    addNode(id, type, x, y, data, fixed = false) {
      this.nodes[id] = { id, type, x, y, data: data || {}, fixed }
      this.render()
    },

    removeNode(id) {
      delete this.nodes[id]
      this.edges = this.edges.filter(e => e.source !== id && e.target !== id)
      if (this.selectedId === id) this._setSelected(null)
      this.render()
    },

    updateNode(id, data) {
      if (this.nodes[id]) {
        Object.assign(this.nodes[id].data, data)
        this.render()
      }
    },

    addEdge(id, type, source, target, label) {
      if (this.edges.find(e => e.id === id)) return
      this.edges.push({ id, type, source, target, label })
      this.render()
    },

    removeEdge(id) {
      this.edges = this.edges.filter(e => e.id !== id)
      this.render()
    },

    _setSelected(id) {
      this.selectedId = id
      if (this._onSelectionChange) this._onSelectionChange(id)
    },

    // Convert screen coords to SVG coords
    _toSvg(clientX, clientY) {
      const rect = this.wrapEl.getBoundingClientRect()
      return {
        x: (clientX - rect.left - this.transform.x) / this.transform.scale,
        y: (clientY - rect.top - this.transform.y) / this.transform.scale
      }
    },

    // Find node at SVG coordinates
    _nodeAt(svgX, svgY) {
      for (const n of Object.values(this.nodes)) {
        const h = ED_NODE_H[n.type] || 90
        if (svgX >= n.x && svgX <= n.x + ED_NODE_W && svgY >= n.y && svgY <= n.y + h) {
          return n
        }
      }
      return null
    },

    render() {
      const defs = Object.entries(ED_EDGE_COLORS).map(([t, s]) =>
        `<marker id="ed-arrow-${t}" markerWidth="8" markerHeight="8" refX="7" refY="3" orient="auto">
           <path d="M0,0 L0,6 L8,3 z" fill="${s.stroke}" opacity="0.8"/>
         </marker>`
      ).join('')

      const edgesHtml = this.edges.map(e => this._renderEdge(e)).join('')
      const nodesHtml = Object.values(this.nodes).map(n => this._renderNode(n)).join('')
      const connLine = this._connState ? this._renderConnLine() : ''

      wrapEl.innerHTML = `<svg width="${ED_CANVAS_W}" height="${ED_CANVAS_H}" class="block">
        <defs>${defs}</defs>
        <g transform="translate(${this.transform.x},${this.transform.y}) scale(${this.transform.scale})">
          ${edgesHtml}${nodesHtml}${connLine}
        </g>
      </svg>`

      this._bindEvents()
    },

    _renderNode(n) {
      const h = ED_NODE_H[n.type] || 90
      const c = ED_COLORS[n.type] || ED_COLORS.service
      const selected = n.id === this.selectedId
      const borderW = selected ? 2.5 : 1
      const shadow = selected ? '0 0 0 3px rgba(99,102,241,0.3)' : '0 1px 3px rgba(0,0,0,0.1)'
      const d = n.data || {}

      let inner = ''
      if (n.type === 'service') {
        const delivStr = (d.deliverables || []).map(del =>
          `<span class="inline-flex items-center rounded bg-amber-100 text-amber-800 px-1.5 py-0.5 text-[9px] font-medium">${del}</span>`
        ).join(' ')
        inner = `
          <div class="text-[13px] font-semibold truncate" style="color:${c.title}">${edEsc(d.name || 'unnamed')}</div>
          <div class="text-[11px] text-gray-500 truncate">${edEsc(d.pipeline || 'no pipeline')}</div>
          ${d.repository ? `<div class="text-[10px] text-gray-400 truncate">${edRepoShort(d.repository)}</div>` : ''}
          ${delivStr ? `<div class="flex flex-wrap gap-0.5 mt-auto">${delivStr}</div>` : ''}`
      } else if (n.type === 'infra') {
        inner = `
          <div class="text-[13px] font-semibold truncate" style="color:${c.title}">${edEsc(d.name || 'unnamed')}</div>
          <span class="inline-flex items-center rounded-md bg-gray-100 px-2 py-0.5 text-[10px] font-medium text-gray-600">${edEsc(d.type || '')}</span>
          <div class="text-[11px] text-gray-500 truncate">via ${edEsc(d.via || 'docker')}</div>`
      } else if (n.type === 'ingress') {
        const routes = (d.routes || []).slice(0, 3).map(r =>
          `<div class="truncate text-[10px]">${r.path} \u2192 ${r.service}:${r.port}</div>`
        ).join('')
        inner = `
          <div class="text-[13px] font-semibold truncate" style="color:${c.title}">${edEsc(d.name || 'ingress')}</div>
          <span class="inline-flex items-center rounded-md bg-gray-100 px-2 py-0.5 text-[10px] font-medium text-gray-600">${edEsc(d.type || 'nginx')}</span>
          <div class="text-gray-500 mt-0.5">${routes}</div>`
      }

      // Connection port
      const portMode = this._portMode
      let portHtml = ''
      if (portMode === 'bottom') {
        // Bottom center port for arch editor (dependsOn connections)
        portHtml = `<circle data-port="${n.id}" cx="${n.x + ED_NODE_W / 2}" cy="${n.y + h + 6}" r="5" fill="white" stroke="#9ca3af" stroke-width="1.5" class="cursor-crosshair" style="pointer-events:all"/>`
      } else {
        // Side ports for deploy editor
        if (n.type === 'infra') {
          portHtml = `<circle data-port="${n.id}" cx="${n.x}" cy="${n.y + h / 2}" r="5" fill="white" stroke="#a855f7" stroke-width="1.5" class="cursor-crosshair" style="pointer-events:all"/>`
        } else if (n.type === 'service') {
          portHtml = `<circle data-port="${n.id}" cx="${n.x + ED_NODE_W}" cy="${n.y + h / 2}" r="5" fill="white" stroke="#a855f7" stroke-width="1.5" class="cursor-crosshair" style="pointer-events:all"/>`
        }
      }

      return `<foreignObject data-node-id="${n.id}" x="${n.x}" y="${n.y}" width="${ED_NODE_W}" height="${h}" style="overflow:visible;cursor:${n.fixed ? 'default' : 'move'}">
        <div xmlns="http://www.w3.org/1999/xhtml" style="width:${ED_NODE_W}px;height:${h}px">
          <div class="w-full h-full rounded-lg px-3 py-2 flex flex-col gap-0.5 overflow-hidden"
               style="background:${c.bg};border:${borderW}px solid ${selected ? '#6366f1' : c.border};box-shadow:${shadow};font-family:ui-sans-serif,system-ui,sans-serif">
            ${inner}
          </div>
        </div>
      </foreignObject>${portHtml}`
    },

    _renderEdge(e) {
      const src = this.nodes[e.source], tgt = this.nodes[e.target]
      if (!src || !tgt) return ''
      const style = ED_EDGE_COLORS[e.type] || ED_EDGE_COLORS.dependsOn
      const srcH = ED_NODE_H[src.type] || 90
      const tgtH = ED_NODE_H[tgt.type] || 90

      let x1, y1, x2, y2, cx1, cy1, cx2, cy2

      if (e.type === 'dependsOn') {
        x1 = src.x + ED_NODE_W / 2; y1 = src.y + srcH
        x2 = tgt.x + ED_NODE_W / 2; y2 = tgt.y
        const dy = (y2 - y1) / 2
        cx1 = x1; cy1 = y1 + dy; cx2 = x2; cy2 = y2 - dy
      } else if (e.type === 'route') {
        x1 = src.x; y1 = src.y + srcH / 2
        x2 = tgt.x + ED_NODE_W; y2 = tgt.y + tgtH / 2
        const dx = Math.abs(x2 - x1) * 0.4
        cx1 = x1 - dx; cy1 = y1; cx2 = x2 + dx; cy2 = y2
      } else {
        // binding: infra (left port) → service (right port)
        x1 = src.x; y1 = src.y + srcH / 2
        x2 = tgt.x + ED_NODE_W; y2 = tgt.y + tgtH / 2
        const dx = Math.abs(x2 - x1) * 0.4
        cx1 = x1 - dx; cy1 = y1; cx2 = x2 + dx; cy2 = y2
      }

      const d = `M ${x1} ${y1} C ${cx1} ${cy1}, ${cx2} ${cy2}, ${x2} ${y2}`
      const label = e.label ? (() => {
        const mx = (x1 + x2) / 2, my = (y1 + y2) / 2
        return `<text x="${mx}" y="${my - 6}" text-anchor="middle" font-size="10" fill="${style.stroke}" font-weight="500">${e.label}</text>`
      })() : ''

      return `<g data-edge-id="${e.id}" class="cursor-pointer" style="pointer-events:stroke">
        <path d="${d}" fill="none" stroke="transparent" stroke-width="12"/>
        <path d="${d}" fill="none" stroke="${style.stroke}" stroke-width="2" stroke-dasharray="${style.dash}" marker-end="url(#ed-arrow-${e.type})" opacity="0.8"/>
        ${label}
      </g>`
    },

    _renderConnLine() {
      const cs = this._connState
      if (!cs) return ''
      return `<line x1="${cs.startX}" y1="${cs.startY}" x2="${cs.curX}" y2="${cs.curY}" stroke="#6366f1" stroke-width="2" stroke-dasharray="6 4" opacity="0.6"/>`
    },

    _bindEvents() {
      const svg = wrapEl.querySelector('svg')
      if (!svg) return
      const self = this

      // Pan on background
      svg.addEventListener('mousedown', (e) => {
        if (e.target.closest('[data-node-id]') || e.target.closest('[data-port]') || e.target.closest('[data-edge-id]')) return
        self._panState = { startX: e.clientX, startY: e.clientY, tx: self.transform.x, ty: self.transform.y }
        // Deselect on background click
        self._setSelected(null)
      })

      svg.addEventListener('mousemove', (e) => {
        if (self._connState) {
          const p = self._toSvg(e.clientX, e.clientY)
          self._connState.curX = p.x
          self._connState.curY = p.y
          // Re-render just the connection line for performance
          const existing = svg.querySelector('.conn-temp')
          if (existing) {
            existing.setAttribute('x2', p.x)
            existing.setAttribute('y2', p.y)
          } else {
            const g = svg.querySelector('g')
            const line = document.createElementNS('http://www.w3.org/2000/svg', 'line')
            line.classList.add('conn-temp')
            line.setAttribute('x1', self._connState.startX)
            line.setAttribute('y1', self._connState.startY)
            line.setAttribute('x2', p.x)
            line.setAttribute('y2', p.y)
            line.setAttribute('stroke', '#6366f1')
            line.setAttribute('stroke-width', '2')
            line.setAttribute('stroke-dasharray', '6 4')
            line.setAttribute('opacity', '0.6')
            g.appendChild(line)
          }
          return
        }
        if (self._dragState) {
          const { id, startMX, startMY, startNX, startNY } = self._dragState
          const dx = (e.clientX - startMX) / self.transform.scale
          const dy = (e.clientY - startMY) / self.transform.scale
          self.nodes[id].x = startNX + dx
          self.nodes[id].y = startNY + dy
          self.render()
        } else if (self._panState) {
          self.transform.x = self._panState.tx + (e.clientX - self._panState.startX)
          self.transform.y = self._panState.ty + (e.clientY - self._panState.startY)
          self.render()
        }
      })

      svg.addEventListener('mouseup', (e) => {
        if (self._connState) {
          const p = self._toSvg(e.clientX, e.clientY)
          const target = self._nodeAt(p.x, p.y)
          if (target && target.id !== self._connState.sourceId && self._onConnectionComplete) {
            self._onConnectionComplete(self._connState.sourceId, target.id)
          }
          self._connState = null
          self.render()
        }
        self._panState = null
        self._dragState = null
      })

      svg.addEventListener('mouseleave', () => {
        self._panState = null
        self._dragState = null
        if (self._connState) {
          self._connState = null
          self.render()
        }
      })

      svg.addEventListener('wheel', (e) => {
        e.preventDefault()
        const factor = e.deltaY < 0 ? 1.1 : 0.9
        self.transform.scale = Math.min(Math.max(self.transform.scale * factor, 0.3), 3)
        self.render()
      }, { passive: false })

      // Node click & drag
      svg.querySelectorAll('[data-node-id]').forEach(fo => {
        fo.addEventListener('mousedown', (e) => {
          e.stopPropagation()
          const id = fo.dataset.nodeId
          const n = self.nodes[id]
          if (!n) return
          self._setSelected(id)
          if (self._onNodeClick) self._onNodeClick(id, n)
          if (!n.fixed) {
            self._dragState = { id, startMX: e.clientX, startMY: e.clientY, startNX: n.x, startNY: n.y }
          }
        })
      })

      // Port mousedown → start connection
      svg.querySelectorAll('[data-port]').forEach(circle => {
        circle.addEventListener('mousedown', (e) => {
          e.stopPropagation()
          const sourceId = circle.dataset.port
          const n = self.nodes[sourceId]
          if (!n) return
          const h = ED_NODE_H[n.type] || 90
          let startX, startY
          if (self._portMode === 'bottom') {
            startX = n.x + ED_NODE_W / 2
            startY = n.y + h + 6
          } else {
            if (n.type === 'infra') {
              startX = n.x
              startY = n.y + h / 2
            } else {
              startX = n.x + ED_NODE_W
              startY = n.y + h / 2
            }
          }
          self._connState = { sourceId, startX, startY, curX: startX, curY: startY }
        })
      })

      // Edge click
      svg.querySelectorAll('[data-edge-id]').forEach(g => {
        g.addEventListener('click', (e) => {
          e.stopPropagation()
          if (self._onEdgeClick) self._onEdgeClick(g.dataset.edgeId)
        })
      })
    }
  }

  return ctrl
}
