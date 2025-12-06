import React, { useEffect, useRef } from 'react'
import { Network } from 'vis-network'
import { DataSet } from 'vis-data'
import './StateMachineGraph.css'

function StateMachineGraph({ graphData, currentStateId, onStateSelect, selectedStateId }) {
  const networkRef = useRef(null)
  const containerRef = useRef(null)

  useEffect(() => {
    if (!graphData || !containerRef.current) return

    const nodes = new DataSet(
      graphData.nodes.map(node => ({
        id: node.id,
        label: node.label,
        color: node.current ? '#4CAF50' : '#E0E0E0',
        shape: 'box',
        font: {
          size: 16,
          color: node.current ? '#fff' : '#333'
        },
        borderWidth: node.current ? 3 : 2,
        borderColor: node.current ? '#2E7D32' : '#BDBDBD',
        chosen: {
          node: (values) => {
            values.color = '#2196F3'
            values.borderColor = '#1976D2'
          }
        }
      }))
    )

    const edges = new DataSet(
      graphData.edges.map(edge => ({
        from: edge.from,
        to: edge.to,
        label: edge.label,
        arrows: 'to',
        color: { color: '#757575' },
        font: { size: 12, align: 'middle' }
      }))
    )

    const data = { nodes, edges }
    const options = {
      nodes: {
        shape: 'box',
        margin: 10,
        widthConstraint: {
          minimum: 120,
          maximum: 200
        }
      },
      edges: {
        smooth: {
          type: 'cubicBezier',
          forceDirection: 'horizontal',
          roundness: 0.4
        }
      },
      layout: {
        hierarchical: {
          direction: 'LR',
          sortMethod: 'directed',
          levelSeparation: 150,
          nodeSpacing: 200,
          treeSpacing: 200
        }
      },
      physics: {
        enabled: false
      },
      interaction: {
        hover: true,
        selectConnectedEdges: false
      }
    }

    const network = new Network(containerRef.current, data, options)

    // 高亮当前状态
    if (currentStateId) {
      setTimeout(() => {
        network.focus(currentStateId, {
          scale: 1.2,
          animation: true
        })
      }, 100)
    }

    // 处理节点点击
    network.on('click', (params) => {
      if (params.nodes.length > 0) {
        const nodeId = params.nodes[0]
        onStateSelect(nodeId)
      }
    })

    // 高亮选中的状态
    if (selectedStateId) {
      network.selectNodes([selectedStateId])
    }

    networkRef.current = network

    return () => {
      if (networkRef.current) {
        networkRef.current.destroy()
      }
    }
  }, [graphData, currentStateId, selectedStateId, onStateSelect])

  return (
    <div className="state-machine-graph">
      <div ref={containerRef} className="graph-container" />
      <div className="graph-legend">
        <div className="legend-item">
          <div className="legend-color current"></div>
          <span>当前状态</span>
        </div>
        <div className="legend-item">
          <div className="legend-color other"></div>
          <span>其他状态</span>
        </div>
      </div>
    </div>
  )
}

export default StateMachineGraph

