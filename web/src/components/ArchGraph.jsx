import { useRef, useState, useCallback, useEffect } from 'react'
import './ArchGraph.css'

// ── Node dimensions ──────────────────────────────────────────────
const NODE_W = 230
const NODE_H = {
  service: 120,
  deliverable: 100,
  compute: 130,
  infra: 110,
  ingress: 130,
}

// ── Artifact type → emoji ────────────────────────────────────────
const ARTIFACT_ICON = {
  'docker-image': '🐳',
  'jar-file': '☕',
  'static-bundle': '📦',
  'binary': '⚙️',
}

// ── Edge colors ──────────────────────────────────────────────────
const EDGE_STYLE = {
  dependsOn:   { stroke: '#9ca3af', strokeWidth: 1.5, dash: '6 4' },
  pipeline:    { stroke: '#3b82f6', strokeWidth: 2,   dash: '' },
  provision:   { stroke: '#22c55e', strokeWidth: 2,   dash: '' },
  binding:     { stroke: '#a855f7', strokeWidth: 1.5, dash: '5 4' },
  route:       { stroke: '#f97316', strokeWidth: 2,   dash: '' },
}

// ── Column header labels ─────────────────────────────────────────
const COL_LABELS = [
  { label: 'Services',       x: 30 },
  { label: 'Deliverables',   x: 300 },
  { label: 'Compute',        x: 570 },
  { label: 'Infrastructure', x: 840 },
]

// ════════════════════════════════════════════════════════════════
// NodeCard — rendered inside <foreignObject>
// ════════════════════════════════════════════════════════════════
function NodeCard({ node }) {
  const { type, data } = node

  if (type === 'service') {
    return (
      <div className="node-card node-service">
        <div className="node-title">{data.name}</div>
        <div className="node-row">
          <span className="label">pipeline</span>
          <span className="value">{data.pipeline}</span>
        </div>
        <div className="node-row">
          <span className="label">repo</span>
          <span className="value">{repoShort(data.repository)}</span>
        </div>
      </div>
    )
  }

  if (type === 'deliverable') {
    const icon = ARTIFACT_ICON[data.artifactType] || '📄'
    return (
      <div className="node-card node-deliverable">
        <div className="artifact-icon">{icon}</div>
        <div className="artifact-type">{data.artifactType || '—'}</div>
        <div className="artifact-pipeline">{data.pipeline}</div>
      </div>
    )
  }

  if (type === 'compute') {
    return (
      <div className="node-card node-compute">
        <div className="node-title">{data.name}</div>
        <span className="node-badge">{data.computeType}</span>
        <div className="node-row">
          <span className="label">accepts</span>
          <span className="value">{data.accepts}</span>
        </div>
        {(data.cpu || data.memory) && (
          <div className="node-row">
            <span className="label">res</span>
            <span className="value">
              {[data.cpu && `CPU ${data.cpu}`, data.memory && `Mem ${data.memory}`, data.replicas > 1 && `×${data.replicas}`]
                .filter(Boolean).join('  ')}
            </span>
          </div>
        )}
      </div>
    )
  }

  if (type === 'infra') {
    return (
      <div className="node-card node-infra">
        <div className="node-title">{data.name}</div>
        <span className="node-badge">{data.resourceType}</span>
        <div className="node-row">
          <span className="label">via</span>
          <span className="value">{data.via}</span>
        </div>
      </div>
    )
  }

  if (type === 'ingress') {
    const routes = data.routes || []
    return (
      <div className="node-card node-ingress">
        <div className="node-title">{data.name}</div>
        <span className="node-badge">{data.ingressType}</span>
        {data.bind && (
          <div className="node-row">
            <span className="label">bind</span>
            <span className="value">
              {data.bind.ip}:{data.bind.http}
              {data.bind.https ? `/:${data.bind.https}` : ''}
            </span>
          </div>
        )}
        <div className="node-routes">
          {routes.slice(0, 3).map((r, i) => (
            <div key={i}>{r.path} → {r.service}:{r.port}</div>
          ))}
          {routes.length > 3 && <div>+{routes.length - 3} more…</div>}
        </div>
      </div>
    )
  }

  return null
}

// ════════════════════════════════════════════════════════════════
// EdgePath — cubic bezier between nodes
// ════════════════════════════════════════════════════════════════
function EdgePath({ edge, nodeMap }) {
  const src = nodeMap[edge.source]
  const tgt = nodeMap[edge.target]
  if (!src || !tgt) return null

  const style = EDGE_STYLE[edge.type] || EDGE_STYLE.pipeline
  const srcH = NODE_H[src.type] || 120
  const tgtH = NODE_H[tgt.type] || 120

  let x1, y1, x2, y2, cx1, cy1, cx2, cy2

  if (edge.type === 'dependsOn') {
    // vertical within col 1: bottom-center of source → top-center of target
    x1 = src.x + NODE_W / 2
    y1 = src.y + srcH
    x2 = tgt.x + NODE_W / 2
    y2 = tgt.y
    const dy = (y2 - y1) / 2
    cx1 = x1 - 40
    cy1 = y1 + dy
    cx2 = x2 - 40
    cy2 = y1 + dy
  } else if (edge.type === 'route') {
    // reverse: right-center of ingress → right-center of compute, arc outward
    x1 = tgt.x + NODE_W    // compute right
    y1 = tgt.y + tgtH / 2
    x2 = src.x             // ingress left
    y2 = src.y + srcH / 2
    const midX = (x1 + x2) / 2 + 80
    cx1 = midX
    cy1 = y1
    cx2 = midX
    cy2 = y2
  } else {
    // horizontal: right-center of source → left-center of target
    x1 = src.x + NODE_W
    y1 = src.y + srcH / 2
    x2 = tgt.x
    y2 = tgt.y + tgtH / 2
    const dx = (x2 - x1) * 0.5
    cx1 = x1 + dx
    cy1 = y1
    cx2 = x2 - dx
    cy2 = y2
  }

  const d = `M ${x1} ${y1} C ${cx1} ${cy1}, ${cx2} ${cy2}, ${x2} ${y2}`

  // midpoint for label
  const mx = (x1 + x2) / 2
  const my = (y1 + y2) / 2

  const markerId = `arrow-${edge.type}`

  return (
    <g>
      <path
        d={d}
        fill="none"
        stroke={style.stroke}
        strokeWidth={style.strokeWidth}
        strokeDasharray={style.dash}
        markerEnd={`url(#${markerId})`}
        opacity={0.85}
      />
      {edge.label && (
        <text
          x={mx}
          y={my - 6}
          textAnchor="middle"
          fontSize={10}
          fill={style.stroke}
          fontWeight="600"
          opacity={0.9}
        >
          {edge.label}
        </text>
      )}
    </g>
  )
}

// ════════════════════════════════════════════════════════════════
// ArchGraph — main component
// ════════════════════════════════════════════════════════════════
export default function ArchGraph({ data }) {
  const { nodes = [], edges = [] } = data

  // ── per-node position overrides (populated when nodes are dragged) ──
  const [positions, setPositions] = useState({})

  // reset positions when data changes
  useEffect(() => { setPositions({}) }, [data])

  // build nodeMap merging API positions with any dragged overrides
  const nodeMap = {}
  for (const n of nodes) {
    const p = positions[n.id]
    nodeMap[n.id] = p ? { ...n, x: p.x, y: p.y } : n
  }

  // compute SVG dimensions from current (possibly dragged) positions
  const allNodes = Object.values(nodeMap)
  const maxX = allNodes.reduce((m, n) => Math.max(m, n.x + NODE_W + 60), 1100)
  const maxY = allNodes.reduce((m, n) => Math.max(m, n.y + (NODE_H[n.type] || 120) + 60), 500)
  const svgW = maxX
  const svgH = maxY + 20

  // ── pan/zoom ──
  const [transform, setTransform] = useState({ x: 20, y: 20, scale: 1 })
  const transformRef = useRef(transform)
  useEffect(() => { transformRef.current = transform }, [transform])

  const panDragging = useRef(false)
  const panStart = useRef({ x: 0, y: 0, tx: 0, ty: 0 })

  // ── node dragging ──
  const nodeDrag = useRef(null) // { id, startMX, startMY, startNX, startNY }

  const wrapRef = useRef(null)

  const onWheel = useCallback((e) => {
    e.preventDefault()
    setTransform(t => {
      const factor = e.deltaY < 0 ? 1.1 : 0.9
      const newScale = Math.min(Math.max(t.scale * factor, 0.2), 3)
      return { ...t, scale: newScale }
    })
  }, [])

  useEffect(() => {
    const el = wrapRef.current
    if (!el) return
    el.addEventListener('wheel', onWheel, { passive: false })
    return () => el.removeEventListener('wheel', onWheel)
  }, [onWheel])

  // canvas pan — only fires when no node is being dragged
  const onCanvasMouseDown = (e) => {
    panDragging.current = true
    panStart.current = { x: e.clientX, y: e.clientY, tx: transform.x, ty: transform.y }
  }

  // node drag start — stops propagation so canvas pan doesn't fire
  const onNodeMouseDown = useCallback((e, node) => {
    e.stopPropagation()
    const cur = nodeMap[node.id] || node
    nodeDrag.current = {
      id: node.id,
      startMX: e.clientX,
      startMY: e.clientY,
      startNX: cur.x,
      startNY: cur.y,
    }
  }, [nodeMap])

  const onMouseMove = (e) => {
    if (nodeDrag.current) {
      const { id, startMX, startMY, startNX, startNY } = nodeDrag.current
      const scale = transformRef.current.scale
      const dx = (e.clientX - startMX) / scale
      const dy = (e.clientY - startMY) / scale
      setPositions(p => ({ ...p, [id]: { x: startNX + dx, y: startNY + dy } }))
    } else if (panDragging.current) {
      const dx = e.clientX - panStart.current.x
      const dy = e.clientY - panStart.current.y
      setTransform(t => ({ ...t, x: panStart.current.tx + dx, y: panStart.current.ty + dy }))
    }
  }

  const onMouseUp = () => {
    panDragging.current = false
    nodeDrag.current = null
  }

  // reset view when data changes
  useEffect(() => {
    setTransform({ x: 20, y: 20, scale: 1 })
  }, [data])

  return (
    <div
      className="arch-graph-wrap"
      ref={wrapRef}
      onMouseDown={onCanvasMouseDown}
      onMouseMove={onMouseMove}
      onMouseUp={onMouseUp}
      onMouseLeave={onMouseUp}
    >
      <svg
        className="arch-graph-svg"
        width={svgW * transform.scale + transform.x + 40}
        height={svgH * transform.scale + transform.y + 40}
      >
        {/* Arrow markers */}
        <defs>
          {Object.entries(EDGE_STYLE).map(([type, s]) => (
            <marker
              key={type}
              id={`arrow-${type}`}
              markerWidth="8" markerHeight="8"
              refX="7" refY="3"
              orient="auto"
            >
              <path d="M0,0 L0,6 L8,3 z" fill={s.stroke} opacity={0.85} />
            </marker>
          ))}
        </defs>

        {/* Transformed content group */}
        <g transform={`translate(${transform.x},${transform.y}) scale(${transform.scale})`}>

          {/* Column headers */}
          {COL_LABELS.map(col => (
            <text
              key={col.label}
              className="col-header"
              x={col.x}
              y={28}
            >
              {col.label}
            </text>
          ))}

          {/* Column separator lines */}
          {[290, 560, 830].map(x => (
            <line
              key={x}
              x1={x} y1={36}
              x2={x} y2={svgH - 10}
              stroke="#e5e7eb"
              strokeWidth={1}
              strokeDasharray="4 4"
            />
          ))}

          {/* Edges (below nodes) — use nodeMap so edges follow dragged nodes */}
          {edges.map(edge => (
            <EdgePath key={edge.id} edge={edge} nodeMap={nodeMap} />
          ))}

          {/* Nodes */}
          {nodes.map(node => {
            const cur = nodeMap[node.id]
            const h = NODE_H[node.type] || 120
            return (
              <foreignObject
                key={node.id}
                x={cur.x}
                y={cur.y}
                width={NODE_W}
                height={h}
                style={{ overflow: 'visible', cursor: 'move' }}
                onMouseDown={e => onNodeMouseDown(e, node)}
              >
                <div xmlns="http://www.w3.org/1999/xhtml" style={{ width: NODE_W, height: h }}>
                  <NodeCard node={node} />
                </div>
              </foreignObject>
            )
          })}
        </g>
      </svg>
    </div>
  )
}

// ── Helpers ──────────────────────────────────────────────────────
function repoShort(url) {
  if (!url) return '—'
  try {
    const u = new URL(url)
    return u.pathname.replace(/^\//, '').replace(/\.git$/, '')
  } catch {
    return url
  }
}
