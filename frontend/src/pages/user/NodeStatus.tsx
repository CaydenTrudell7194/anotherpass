import { useEffect, useMemo, useRef, useState } from 'react'
import { Alert, Empty, Progress, Spin, Tooltip, Badge } from 'antd'
import {
  ArrowDownOutlined,
  ArrowUpOutlined,
  CheckCircleFilled,
  ClockCircleOutlined,
  CloudServerOutlined,
  DisconnectOutlined,
  WifiOutlined,
  HddOutlined,
  DashboardOutlined,
  GlobalOutlined,
  ClusterOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons'
import api from '../../api'

type Metrics = {
  hostname: string
  platform: string
  platform_version: string
  arch: string
  version: string
  cpu_model: string
  cpu_percent: number
  load1: number
  load5: number
  load15: number
  process_count: number
  mem_total: number
  mem_used: number
  swap_total: number
  swap_used: number
  disk_total: number
  disk_used: number
  net_in_speed: number
  net_out_speed: number
  net_in_transfer: number
  net_out_transfer: number
  tcp_conn_count: number
  udp_conn_count: number
  uptime_seconds: number
  boot_time: number
  ip4_geo?: string
  ip6_geo?: string
}

type Node = {
  id: number
  device_group_id: number
  name: string
  ip: string
  ip6?: string
  ip4_geo?: string
  ip6_geo?: string
  online: boolean
  last_heartbeat: string
  last_update: string
  metrics: Metrics
}

type Group = { id: number; name: string; nodes: Node[] }
type Snapshot = { server_time: number; groups: Group[] }
type ConnectionState = 'connecting' | 'connected' | 'disconnected'

const number = (value: unknown) => Number.isFinite(Number(value)) ? Number(value) : 0

const formatBytes = (value: number) => {
  const bytes = number(value)
  if (bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  const amount = bytes / 1024 ** index
  return `${amount.toFixed(amount >= 100 || index === 0 ? 0 : amount >= 10 ? 1 : 2)} ${units[index]}`
}

const formatSpeed = (value: number) => `${formatBytes(value)}/s`

const formatTime = (value?: string) => {
  if (!value || value.startsWith('0001')) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '-' : date.toLocaleString('zh-CN', { hour12: false })
}

const formatUnixTime = (value?: number) => value && value > 0 ? new Date(value * 1000).toLocaleString('zh-CN', { hour12: false }) : '-'

const countryFlag = (code?: string) => code && code.length === 2
  ? String.fromCodePoint(code.charCodeAt(0) - 0x41 + 0x1F1E6, code.charCodeAt(1) - 0x41 + 0x1F1E6)
  : ''

const formatUptime = (seconds: number) => {
  const total = Math.max(0, Math.floor(number(seconds)))
  const days = Math.floor(total / 86400)
  const hours = Math.floor((total % 86400) / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  if (days) return `${days}天 ${hours}小时`
  if (hours) return `${hours}小时 ${minutes}分`
  return `${minutes}分钟`
}

const percent = (used: number, total: number) => total > 0
  ? Math.min(100, Math.max(0, number(used) / number(total) * 100))
  : 0

const progressColor = (value: number) => value >= 90 ? '#ef4444' : value >= 75 ? '#f59e0b' : '#3b82f6'

export default function NodeStatus() {
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null)
  const [connection, setConnection] = useState<ConnectionState>('connecting')
  const [error, setError] = useState('')
  const generationRef = useRef(0)

  useEffect(() => {
    const generation = ++generationRef.current
    let disposed = false
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null
    let socket: WebSocket | null = null
    let ticketController: AbortController | null = null

    const isCurrent = () => !disposed && generationRef.current === generation

    const scheduleReconnect = () => {
      if (!isCurrent() || reconnectTimer) return
      setConnection('disconnected')
      reconnectTimer = setTimeout(() => {
        reconnectTimer = null
        connect()
      }, 3000)
    }

    const connect = async () => {
      if (!isCurrent()) return
      setConnection('connecting')
      ticketController?.abort()
      ticketController = new AbortController()

      try {
        const response = await api.post<{ ticket: string }>('/node-monitor/ticket', undefined, {
          signal: ticketController.signal,
        })
        if (!isCurrent()) return
        if (!response.data?.ticket) throw new Error('未获取到连接凭证')

        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
        const url = `${protocol}//${window.location.host}/api/node-monitor/ws?ticket=${encodeURIComponent(response.data.ticket)}`
        socket = new WebSocket(url)

        socket.onopen = () => {
          if (!isCurrent()) return socket?.close()
          setConnection('connected')
          setError('')
        }
        socket.onmessage = event => {
          if (!isCurrent()) return
          try {
            const next = JSON.parse(event.data) as Snapshot
            if (!Array.isArray(next.groups)) throw new Error('invalid snapshot')
            setSnapshot(next)
            setConnection('connected')
            setError('')
          } catch {
            setError('监控数据格式异常，正在等待下一次更新')
          }
        }
        socket.onerror = () => {
          if (isCurrent()) setError('实时监控连接异常，正在尝试恢复')
        }
        socket.onclose = () => {
          socket = null
          scheduleReconnect()
        }
      } catch (reason: any) {
        if (!isCurrent() || reason?.name === 'CanceledError' || reason?.code === 'ERR_CANCELED') return
        setError(reason?.response?.data?.error || reason?.message || '无法连接实时监控服务')
        scheduleReconnect()
      }
    }

    connect()
    return () => {
      disposed = true
      generationRef.current++
      ticketController?.abort()
      if (reconnectTimer) clearTimeout(reconnectTimer)
      if (socket) {
        socket.onopen = null
        socket.onmessage = null
        socket.onerror = null
        socket.onclose = null
        socket.close()
      }
    }
  }, [])

  const totals = useMemo(() => {
    const nodes = snapshot?.groups.flatMap(group => group.nodes || []) || []
    return {
      total: nodes.length,
      online: nodes.filter(node => node.online).length,
      up: nodes.reduce((sum, node) => sum + number(node.metrics?.net_out_speed), 0),
      down: nodes.reduce((sum, node) => sum + number(node.metrics?.net_in_speed), 0),
    }
  }, [snapshot])

  return (
    <main className="glass-monitor">
      {/* 顶部面板 Header */}
      <header className="glass-monitor__header">
        <div className="glass-monitor__title-area">
          <div className="glass-monitor__icon-badge">
            <CloudServerOutlined />
          </div>
          <div>
            <span className="glass-monitor__sub">REAL-TIME MONITOR</span>
            <h1>节点探针</h1>
          </div>
        </div>
        <div className={`glass-monitor__status-badge is-${connection}`}>
          {connection === 'connected' ? <WifiOutlined className="pulse-icon" /> : <DisconnectOutlined />}
          <span>
            {connection === 'connected' ? '已连接 · 1S 实时更新' : connection === 'connecting' ? '正在连接...' : '连接中断 · 3S 后重连'}
          </span>
        </div>
      </header>

      {/* 汇总统计信息面板 */}
      <section className="glass-monitor__summary">
        <div className="glass-card summary-card">
          <div className="summary-card__icon text-emerald">
            <ClusterOutlined />
          </div>
          <div className="summary-card__content">
            <span className="summary-card__label">在线节点</span>
            <div className="summary-card__val">
              <span className="text-emerald">{totals.online}</span>
              <span className="summary-card__total">/ {totals.total}</span>
            </div>
          </div>
        </div>

        <div className="glass-card summary-card">
          <div className="summary-card__icon text-rose">
            <ArrowUpOutlined />
          </div>
          <div className="summary-card__content">
            <span className="summary-card__label">总上传速率</span>
            <div className="summary-card__val text-rose">{formatSpeed(totals.up)}</div>
          </div>
        </div>

        <div className="glass-card summary-card">
          <div className="summary-card__icon text-sky">
            <ArrowDownOutlined />
          </div>
          <div className="summary-card__content">
            <span className="summary-card__label">总下载速率</span>
            <div className="summary-card__val text-sky">{formatSpeed(totals.down)}</div>
          </div>
        </div>

        <div className="glass-card summary-card">
          <div className="summary-card__icon text-amber">
            <ClockCircleOutlined />
          </div>
          <div className="summary-card__content">
            <span className="summary-card__label">服务器时间</span>
            <div className="summary-card__val summary-card__time">
              {formatUnixTime(snapshot?.server_time)}
            </div>
          </div>
        </div>
      </section>

      {error && <Alert className="glass-monitor__alert" type="error" showIcon message={error} />}

      {!snapshot && !error && (
        <div className="glass-monitor__state">
          <Spin size="large" />
          <span>正在连接实时数据源...</span>
        </div>
      )}

      {snapshot && snapshot.groups.length === 0 && (
        <div className="glass-monitor__state">
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无节点分组数据" />
        </div>
      )}

      {/* 按设备组展示磨砂玻璃卡片瀑布流 */}
      {snapshot?.groups.map(group => {
        const groupUp = (group.nodes || []).reduce((sum, node) => sum + number(node.metrics?.net_out_speed), 0)
        const groupDown = (group.nodes || []).reduce((sum, node) => sum + number(node.metrics?.net_in_speed), 0)
        const onlineCount = (group.nodes || []).filter(node => node.online).length

        return (
          <section className="glass-monitor__group" key={group.id}>
            <div className="group-header">
              <div className="group-header__left">
                <span className="group-header__tag">ID: {group.id}</span>
                <h2>{group.name}</h2>
                <Badge
                  count={`${onlineCount} / ${group.nodes?.length || 0} 在线`}
                  style={{ backgroundColor: onlineCount > 0 ? '#10b981' : '#6b7280', borderRadius: '12px', padding: '0 10px', fontSize: '12px' }}
                />
              </div>
              <div className="group-header__speed">
                <span className="text-rose"><ArrowUpOutlined /> {formatSpeed(groupUp)}</span>
                <span className="text-sky"><ArrowDownOutlined /> {formatSpeed(groupDown)}</span>
              </div>
            </div>

            {/* 卡片网格 Grid */}
            <div className="glass-grid">
              {(group.nodes || []).map(node => {
                const cpuVal = number(node.metrics?.cpu_percent)
                const memVal = percent(node.metrics?.mem_used, node.metrics?.mem_total)
                const diskVal = percent(node.metrics?.disk_used, node.metrics?.disk_total)
                const flag4 = countryFlag(node.ip4_geo || node.metrics?.ip4_geo || '')
                const flag6 = countryFlag(node.ip6_geo || node.metrics?.ip6_geo || '')

                return (
                  <div className={`glass-card node-card ${node.online ? 'is-online' : 'is-offline'}`} key={node.id}>
                    {/* 卡片头部：状态呼吸灯、节点名、国旗 */}
                    <div className="node-card__header">
                      <div className="node-card__title">
                        <span className={`status-glow ${node.online ? 'is-online' : ''}`} />
                        <span className="node-card__name" title={node.name}>
                          {node.name || `节点 #${node.id}`}
                        </span>
                      </div>
                      <div className="node-card__flags">
                        {flag4 && <span className="flag-emoji" title={node.ip4_geo}>{flag4}</span>}
                        {flag6 && <span className="flag-emoji" title={node.ip6_geo}>{flag6}</span>}
                      </div>
                    </div>

                    {/* 卡片子头部：IP 和 挂载系统信息 */}
                    <div className="node-card__sub">
                      <span className="node-card__ip">
                        {node.ip ? node.ip : <span className="text-muted">IP 暂无/已保护</span>}
                      </span>
                      {node.metrics?.hostname && (
                        <Tooltip title={
                          <div>
                            <strong>{node.metrics.hostname}</strong><br />
                            OS: {node.metrics.platform} {node.metrics.platform_version} ({node.metrics.arch})<br />
                            Agent: {node.metrics.version || 'v1.0.0'}
                          </div>
                        }>
                          <span className="node-card__host">
                            <InfoCircleOutlined /> {node.metrics.hostname}
                          </span>
                        </Tooltip>
                      )}
                    </div>

                    {/* 实时速率 & 运行时间 */}
                    <div className="node-card__stats-row">
                      <Tooltip title={`TCP 连接: ${number(node.metrics?.tcp_conn_count)} | UDP 连接: ${number(node.metrics?.udp_conn_count)}`}>
                        <div className="node-card__speed">
                          <span className="text-rose"><ArrowUpOutlined /> {formatSpeed(node.metrics?.net_out_speed)}</span>
                          <span className="text-sky"><ArrowDownOutlined /> {formatSpeed(node.metrics?.net_in_speed)}</span>
                        </div>
                      </Tooltip>
                      <Tooltip title={
                        <div>
                          开机时间：{formatUnixTime(node.metrics?.boot_time)}<br />
                          上次同步：{formatTime(node.last_update || node.last_heartbeat)}
                        </div>
                      }>
                        <div className="node-card__uptime">
                          <ClockCircleOutlined /> {node.online ? formatUptime(node.metrics?.uptime_seconds) : '离线'}
                        </div>
                      </Tooltip>
                    </div>

                    {/* 资源使用率 (CPU / 内存 / 磁盘) 磨砂精致进度条 */}
                    <div className="node-card__metrics">
                      {/* CPU */}
                      <Tooltip title={
                        <div>
                          型号: {node.metrics?.cpu_model || '未知 CPU'}<br />
                          负载: {number(node.metrics?.load1).toFixed(2)} / {number(node.metrics?.load5).toFixed(2)} / {number(node.metrics?.load15).toFixed(2)}<br />
                          进程数: {number(node.metrics?.process_count)}
                        </div>
                      }>
                        <div className="metric-row">
                          <div className="metric-label">
                            <span><DashboardOutlined /> CPU</span>
                            <strong>{cpuVal.toFixed(1)}%</strong>
                          </div>
                          <Progress percent={cpuVal} showInfo={false} size="small" strokeColor={progressColor(cpuVal)} trailColor="rgba(255,255,255,0.15)" />
                        </div>
                      </Tooltip>

                      {/* 内存 */}
                      <Tooltip title={`Swap: ${formatBytes(node.metrics?.swap_used)} / ${formatBytes(node.metrics?.swap_total)}`}>
                        <div className="metric-row">
                          <div className="metric-label">
                            <span><GlobalOutlined /> 内存</span>
                            <strong>{formatBytes(node.metrics?.mem_used)} / {formatBytes(node.metrics?.mem_total)}</strong>
                          </div>
                          <Progress percent={memVal} showInfo={false} size="small" strokeColor={progressColor(memVal)} trailColor="rgba(255,255,255,0.15)" />
                        </div>
                      </Tooltip>

                      {/* 磁盘 */}
                      <div className="metric-row">
                        <div className="metric-label">
                          <span><HddOutlined /> 磁盘</span>
                          <strong>{formatBytes(node.metrics?.disk_used)} / {formatBytes(node.metrics?.disk_total)}</strong>
                        </div>
                        <Progress percent={diskVal} showInfo={false} size="small" strokeColor={progressColor(diskVal)} trailColor="rgba(255,255,255,0.15)" />
                      </div>
                    </div>

                    {/* 底部流量总和 */}
                    <div className="node-card__footer">
                      <span>累计流量</span>
                      <div>
                        <span className="text-rose">↑ {formatBytes(node.metrics?.net_out_transfer)}</span>
                        <span className="text-sky" style={{ marginLeft: 8 }}>↓ {formatBytes(node.metrics?.net_in_transfer)}</span>
                      </div>
                    </div>
                  </div>
                )
              })}
            </div>
          </section>
        )
      })}

      <style>{`
        /* Glassmorphism Monitor Core Styles */
        .glass-monitor {
          --glass-bg: rgba(255, 255, 255, 0.45);
          --glass-border: rgba(255, 255, 255, 0.6);
          --glass-shadow: 0 8px 32px 0 rgba(31, 38, 135, 0.08);
          --card-bg: rgba(255, 255, 255, 0.65);
          --text-primary: #1e293b;
          --text-secondary: #64748b;
          --text-muted: #94a3b8;
          min-height: 100vh;
          padding: 24px;
          color: var(--text-primary);
          transition: all 0.3s ease;
        }

        /* 暗色主题毛玻璃覆盖 */
        .my-theme-dark .glass-monitor {
          --glass-bg: rgba(15, 23, 42, 0.55);
          --glass-border: rgba(255, 255, 255, 0.1);
          --glass-shadow: 0 8px 32px 0 rgba(0, 0, 0, 0.37);
          --card-bg: rgba(30, 41, 59, 0.65);
          --text-primary: #f8fafc;
          --text-secondary: #94a3b8;
          --text-muted: #64748b;
        }

        .text-rose { color: #f43f5e !important; }
        .text-sky { color: #0284c7 !important; }
        .text-emerald { color: #10b981 !important; }
        .text-amber { color: #f59e0b !important; }
        .text-muted { color: var(--text-muted); }

        /* 通用毛玻璃卡片 */
        .glass-card {
          background: var(--card-bg);
          backdrop-filter: blur(16px);
          -webkit-backdrop-filter: blur(16px);
          border: 1px solid var(--glass-border);
          box-shadow: var(--glass-shadow);
          border-radius: 16px;
          padding: 16px;
          transition: transform 0.2s ease, box-shadow 0.2s ease;
        }

        .glass-card:hover {
          transform: translateY(-2px);
          box-shadow: 0 12px 40px 0 rgba(31, 38, 135, 0.15);
        }

        /* Header 区域 */
        .glass-monitor__header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 24px;
          padding-bottom: 16px;
          border-bottom: 1px solid var(--glass-border);
        }

        .glass-monitor__title-area {
          display: flex;
          align-items: center;
          gap: 16px;
        }

        .glass-monitor__icon-badge {
          width: 48px;
          height: 48px;
          border-radius: 14px;
          background: linear-gradient(135deg, #3b82f6, #6366f1);
          color: #ffffff;
          display: flex;
          align-items: center;
          justify-content: center;
          font-size: 24px;
          box-shadow: 0 4px 14px rgba(99, 102, 241, 0.35);
        }

        .glass-monitor__sub {
          font-size: 11px;
          font-weight: 700;
          letter-spacing: 1.5px;
          color: var(--text-secondary);
        }

        .glass-monitor h1 {
          margin: 0;
          font-size: 24px;
          font-weight: 800;
          color: var(--text-primary);
        }

        .glass-monitor__status-badge {
          display: flex;
          align-items: center;
          gap: 8px;
          padding: 8px 16px;
          border-radius: 20px;
          font-size: 13px;
          font-weight: 600;
          background: var(--glass-bg);
          border: 1px solid var(--glass-border);
          backdrop-filter: blur(12px);
        }

        .glass-monitor__status-badge.is-connected { color: #10b981; }
        .glass-monitor__status-badge.is-connecting { color: #f59e0b; }
        .glass-monitor__status-badge.is-disconnected { color: #ef4444; }

        .pulse-icon {
          animation: pulse-ring 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
        }

        @keyframes pulse-ring {
          0%, 100% { opacity: 1; transform: scale(1); }
          50% { opacity: 0.5; transform: scale(1.1); }
        }

        /* 汇总四元组 */
        .glass-monitor__summary {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
          gap: 16px;
          margin-bottom: 32px;
        }

        .summary-card {
          display: flex;
          align-items: center;
          gap: 16px;
        }

        .summary-card__icon {
          font-size: 28px;
          width: 44px;
          height: 44px;
          border-radius: 12px;
          background: rgba(255, 255, 255, 0.2);
          display: flex;
          align-items: center;
          justify-content: center;
        }

        .summary-card__label {
          font-size: 12px;
          color: var(--text-secondary);
          display: block;
          margin-bottom: 2px;
        }

        .summary-card__val {
          font-size: 20px;
          font-weight: 700;
          line-height: 1.2;
        }

        .summary-card__total {
          font-size: 14px;
          color: var(--text-muted);
          margin-left: 4px;
        }

        .summary-card__time {
          font-size: 14px;
          font-weight: 600;
        }

        /* 设备组 Section */
        .glass-monitor__group {
          margin-bottom: 36px;
        }

        .group-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 16px;
          padding: 0 4px;
        }

        .group-header__left {
          display: flex;
          align-items: center;
          gap: 12px;
        }

        .group-header__tag {
          font-size: 11px;
          font-weight: 700;
          color: var(--text-secondary);
          background: var(--glass-bg);
          padding: 2px 8px;
          border-radius: 6px;
          border: 1px solid var(--glass-border);
        }

        .group-header h2 {
          margin: 0;
          font-size: 18px;
          font-weight: 700;
          color: var(--text-primary);
        }

        .group-header__speed {
          display: flex;
          gap: 16px;
          font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
          font-weight: 700;
          font-size: 13px;
        }

        /* 节点卡片 Grid */
        .glass-grid {
          display: grid;
          grid-template-columns: repeat(auto-fill, minmax(290px, 1fr));
          gap: 16px;
        }

        .node-card {
          display: flex;
          flex-direction: column;
          gap: 12px;
          position: relative;
          overflow: hidden;
        }

        .node-card.is-offline {
          opacity: 0.65;
        }

        .node-card__header {
          display: flex;
          justify-content: space-between;
          align-items: center;
        }

        .node-card__title {
          display: flex;
          align-items: center;
          gap: 8px;
          max-width: 80%;
        }

        .node-card__name {
          font-weight: 700;
          font-size: 15px;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
          color: var(--text-primary);
        }

        /* 呼吸灯 Glow */
        .status-glow {
          width: 10px;
          height: 10px;
          border-radius: 50%;
          background: #ef4444;
          box-shadow: 0 0 8px #ef4444;
          flex-shrink: 0;
          transition: all 0.3s ease;
        }

        .status-glow.is-online {
          background: #10b981;
          box-shadow: 0 0 10px #10b981;
        }

        .node-card__flags {
          font-size: 18px;
          display: flex;
          gap: 4px;
        }

        .node-card__sub {
          display: flex;
          justify-content: space-between;
          align-items: center;
          font-size: 12px;
          color: var(--text-secondary);
        }

        .node-card__ip {
          font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
          font-weight: 600;
        }

        .node-card__host {
          cursor: help;
          max-width: 120px;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }

        .node-card__stats-row {
          display: flex;
          justify-content: space-between;
          align-items: center;
          background: rgba(0, 0, 0, 0.03);
          padding: 8px 12px;
          border-radius: 10px;
          font-size: 12px;
        }

        .my-theme-dark .node-card__stats-row {
          background: rgba(255, 255, 255, 0.04);
        }

        .node-card__speed {
          display: flex;
          gap: 12px;
          font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
          font-weight: 700;
        }

        .node-card__uptime {
          color: var(--text-secondary);
          font-size: 11px;
        }

        /* 指标 Progress 区域 */
        .node-card__metrics {
          display: flex;
          flex-direction: column;
          gap: 8px;
        }

        .metric-row {
          display: flex;
          flex-direction: column;
          gap: 2px;
        }

        .metric-label {
          display: flex;
          justify-content: space-between;
          font-size: 11px;
          color: var(--text-secondary);
        }

        .metric-label strong {
          color: var(--text-primary);
        }

        .node-card__footer {
          display: flex;
          justify-content: space-between;
          align-items: center;
          border-top: 1px dashed var(--glass-border);
          padding-top: 8px;
          font-size: 11px;
          color: var(--text-secondary);
          font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
        }

        .glass-monitor__state {
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          min-height: 200px;
          gap: 12px;
          color: var(--text-secondary);
        }

        @media (max-width: 640px) {
          .glass-monitor { padding: 16px; }
          .glass-monitor__header { flex-direction: column; align-items: flex-start; gap: 12px; }
          .glass-grid { grid-template-columns: 1fr; }
        }
      `}</style>
    </main>
  )
}
