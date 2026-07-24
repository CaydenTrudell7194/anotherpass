import { useEffect, useMemo, useRef, useState } from 'react'
import { Alert, Empty, Progress, Spin, Table, Tooltip } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  ArrowDownOutlined,
  ArrowUpOutlined,
  CheckCircleFilled,
  ClockCircleOutlined,
  CloudServerOutlined,
  DisconnectOutlined,
  InfoCircleOutlined,
  WifiOutlined,
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
}

type Node = {
  id: number
  device_group_id: number
  name: string
  ip: string
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

const progressColor = (value: number) => value >= 90 ? '#c71f37' : value >= 75 ? '#d97706' : '#1976a3'

function MetricProgress({ value, detail }: { value: number; detail?: string }) {
  const normalized = Math.min(100, Math.max(0, number(value)))
  return (
    <div className="node-status__progress">
      <div><strong>{normalized.toFixed(1)}%</strong>{detail && <span>{detail}</span>}</div>
      <Progress percent={normalized} showInfo={false} size="small" strokeColor={progressColor(normalized)} trailColor="#e7e7e3" />
    </div>
  )
}

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

  const columns = useMemo<ColumnsType<Node>>(() => [
    {
      title: '状态 / IP',
      key: 'identity',
      width: 190,
      fixed: 'left',
      render: (_, node) => (
        <div className="node-status__identity">
          <div className="node-status__node-name">
            <span className={`node-status__dot ${node.online ? 'is-online' : ''}`} />
            <strong>{node.name || `节点 #${node.id}`}</strong>
          </div>
          <span className="node-status__mono">{node.ip || '-'}</span>
        </div>
      ),
    },
    {
      title: '实时速率',
      key: 'speed',
      width: 150,
      render: (_, node) => (
        <Tooltip title={`TCP ${number(node.metrics?.tcp_conn_count)} · UDP ${number(node.metrics?.udp_conn_count)}`}>
          <div className="node-status__traffic">
            <span className="is-up"><ArrowUpOutlined /> {formatSpeed(node.metrics?.net_out_speed)}</span>
            <span className="is-down"><ArrowDownOutlined /> {formatSpeed(node.metrics?.net_in_speed)}</span>
          </div>
        </Tooltip>
      ),
    },
    {
      title: '运行时间',
      key: 'uptime',
      width: 125,
      render: (_, node) => (
        <Tooltip title={<div>启动：{formatUnixTime(node.metrics?.boot_time)}<br />同步：{formatTime(node.last_update || node.last_heartbeat)}</div>}>
          <span className="node-status__uptime"><ClockCircleOutlined /> {node.online ? formatUptime(node.metrics?.uptime_seconds) : '离线'}</span>
        </Tooltip>
      ),
    },
    {
      title: '累计流量',
      key: 'transfer',
      width: 145,
      render: (_, node) => (
        <div className="node-status__traffic">
          <span className="is-up"><ArrowUpOutlined /> {formatBytes(node.metrics?.net_out_transfer)}</span>
          <span className="is-down"><ArrowDownOutlined /> {formatBytes(node.metrics?.net_in_transfer)}</span>
        </div>
      ),
    },
    {
      title: 'CPU',
      key: 'cpu',
      width: 165,
      render: (_, node) => (
        <Tooltip title={<div>{node.metrics?.cpu_model || '未知处理器'}<br />负载 {number(node.metrics?.load1).toFixed(2)} / {number(node.metrics?.load5).toFixed(2)} / {number(node.metrics?.load15).toFixed(2)}<br />进程 {number(node.metrics?.process_count)}</div>}>
          <div><MetricProgress value={node.metrics?.cpu_percent} /><span className="node-status__hint">负载 {number(node.metrics?.load1).toFixed(2)} · {number(node.metrics?.process_count)} 进程</span></div>
        </Tooltip>
      ),
    },
    {
      title: '内存',
      key: 'memory',
      width: 180,
      render: (_, node) => {
        const value = percent(node.metrics?.mem_used, node.metrics?.mem_total)
        return (
          <Tooltip title={`交换空间 ${formatBytes(node.metrics?.swap_used)} / ${formatBytes(node.metrics?.swap_total)}`}>
            <MetricProgress value={value} detail={`${formatBytes(node.metrics?.mem_used)} / ${formatBytes(node.metrics?.mem_total)}`} />
          </Tooltip>
        )
      },
    },
    {
      title: '磁盘',
      key: 'disk',
      width: 180,
      render: (_, node) => <MetricProgress value={percent(node.metrics?.disk_used, node.metrics?.disk_total)} detail={`${formatBytes(node.metrics?.disk_used)} / ${formatBytes(node.metrics?.disk_total)}`} />,
    },
    {
      title: '系统',
      key: 'system',
      width: 220,
      render: (_, node) => (
        <div className="node-status__system">
          <strong>{node.metrics?.hostname || '-'}</strong>
          <span>{[node.metrics?.platform, node.metrics?.platform_version, node.metrics?.arch].filter(Boolean).join(' · ') || '-'}</span>
          <span>Agent {node.metrics?.version || '-'}</span>
        </div>
      ),
    },
  ], [])

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
    <main className="node-status">
      <header className="node-status__header">
        <div>
          <span className="node-status__kicker">NETWORK OPERATIONS</span>
          <h1><CloudServerOutlined /> 节点监控</h1>
        </div>
        <div className={`node-status__connection is-${connection}`}>
          {connection === 'connected' ? <WifiOutlined /> : <DisconnectOutlined />}
          <span>{connection === 'connected' ? '已连接 · 1秒更新' : connection === 'connecting' ? '正在连接' : '连接中断 · 3秒后重试'}</span>
        </div>
      </header>

      <section className="node-status__summary" aria-label="节点概览">
        <div><span>在线节点</span><strong className="is-online">{totals.online}<small> / {totals.total}</small></strong></div>
        <div><span><ArrowUpOutlined /> 总上传</span><strong>{formatSpeed(totals.up)}</strong></div>
        <div><span><ArrowDownOutlined /> 总下载</span><strong>{formatSpeed(totals.down)}</strong></div>
        <div><span>服务器时间</span><strong className="node-status__summary-time">{formatUnixTime(snapshot?.server_time)}</strong></div>
      </section>

      {error && <Alert className="node-status__alert" type="error" showIcon message={error} />}

      {!snapshot && !error && (
        <div className="node-status__state"><Spin size="large" /><span>正在接入监控数据</span></div>
      )}

      {snapshot && snapshot.groups.length === 0 && (
        <div className="node-status__state"><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无节点分组" /></div>
      )}

      {snapshot?.groups.map(group => {
        const groupUp = (group.nodes || []).reduce((sum, node) => sum + number(node.metrics?.net_out_speed), 0)
        const groupDown = (group.nodes || []).reduce((sum, node) => sum + number(node.metrics?.net_in_speed), 0)
        const online = (group.nodes || []).filter(node => node.online).length
        return (
          <section className="node-status__group" key={group.id}>
            <div className="node-status__group-heading">
              <div>
                <span className="node-status__group-index">GROUP {String(group.id).padStart(2, '0')}</span>
                <h2>{group.name}</h2>
                <span className="node-status__group-count"><CheckCircleFilled /> {online}/{group.nodes?.length || 0} 在线</span>
              </div>
              <div className="node-status__group-speed">
                <span><ArrowUpOutlined /> {formatSpeed(groupUp)}</span>
                <span><ArrowDownOutlined /> {formatSpeed(groupDown)}</span>
              </div>
            </div>
            <Table<Node>
              rowKey="id"
              columns={columns}
              dataSource={group.nodes || []}
              pagination={false}
              size="small"
              scroll={{ x: 1355 }}
              locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="该分组暂无节点" /> }}
            />
          </section>
        )
      })}

      <style>{`
        .node-status { --ink:#171717; --paper:#f7f7f3; --line:#d6d6d0; --red:#c71f37; --blue:#1976a3; color:var(--ink); background:var(--paper); min-height:100%; padding:28px clamp(16px,3vw,42px) 48px; font-family:Inter,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif; }
        .node-status__header { display:flex; align-items:flex-end; justify-content:space-between; gap:24px; padding-bottom:18px; border-bottom:3px solid var(--ink); }
        .node-status__kicker,.node-status__group-index { display:block; margin-bottom:5px; color:#666; font-size:10px; font-weight:800; letter-spacing:1.8px; }
        .node-status h1 { margin:0; font-family:Georgia,"Times New Roman",serif; font-size:clamp(28px,4vw,43px); line-height:1; letter-spacing:0; }
        .node-status h1 .anticon { margin-right:10px; color:var(--red); font-size:.75em; }
        .node-status__connection { display:flex; align-items:center; gap:8px; padding-bottom:3px; font-size:12px; font-weight:700; white-space:nowrap; }
        .node-status__connection .anticon { font-size:15px; }
        .node-status__connection.is-connected { color:#27704b; }
        .node-status__connection.is-connecting { color:#a36208; }
        .node-status__connection.is-disconnected { color:var(--red); }
        .node-status__summary { display:grid; grid-template-columns:repeat(4,minmax(0,1fr)); border-bottom:1px solid var(--ink); }
        .node-status__summary>div { min-width:0; padding:16px 18px; border-right:1px solid var(--line); }
        .node-status__summary>div:first-child { padding-left:0; }
        .node-status__summary>div:last-child { border-right:0; }
        .node-status__summary span { display:block; margin-bottom:5px; color:#666; font-size:11px; font-weight:700; }
        .node-status__summary strong { display:block; overflow:hidden; font-family:Georgia,"Times New Roman",serif; font-size:24px; line-height:1.15; text-overflow:ellipsis; white-space:nowrap; }
        .node-status__summary strong.is-online { color:#27704b; font-size:30px; }
        .node-status__summary strong small { color:#777; font-size:16px; font-weight:400; }
        .node-status__summary .node-status__summary-time { font-family:inherit; font-size:14px; line-height:28px; }
        .node-status__alert { margin:18px 0 0; border-radius:0; }
        .node-status__state { display:flex; min-height:260px; align-items:center; justify-content:center; flex-direction:column; gap:14px; color:#777; font-size:13px; }
        .node-status__group { padding-top:30px; }
        .node-status__group+.node-status__group { margin-top:12px; border-top:1px solid var(--ink); }
        .node-status__group-heading { display:flex; min-height:52px; align-items:flex-end; justify-content:space-between; gap:20px; margin-bottom:10px; }
        .node-status__group-heading>div:first-child { display:flex; align-items:baseline; gap:12px; flex-wrap:wrap; }
        .node-status__group-index { width:100%; margin:0 0 -7px; }
        .node-status h2 { margin:0; font-family:Georgia,"Times New Roman",serif; font-size:23px; line-height:1.2; letter-spacing:0; }
        .node-status__group-count { color:#27704b; font-size:11px; font-weight:700; }
        .node-status__group-speed { display:flex; gap:18px; padding-bottom:3px; font:700 12px ui-monospace,SFMono-Regular,Consolas,monospace; white-space:nowrap; }
        .node-status__group-speed span:first-child,.is-up { color:var(--red); }
        .node-status__group-speed span:last-child,.is-down { color:var(--blue); }
        .node-status .ant-table-wrapper { border-top:2px solid var(--ink); }
        .node-status .ant-table { background:transparent; border-radius:0; font-size:12px; }
        .node-status .ant-table-container,.node-status .ant-table-content,.node-status .ant-table-cell { border-radius:0!important; }
        .node-status .ant-table-thead>tr>th { padding:8px 10px!important; background:#ebebe6!important; border-bottom:1px solid var(--ink)!important; color:#555; font-size:10px; font-weight:800; letter-spacing:.5px; text-transform:uppercase; }
        .node-status .ant-table-tbody>tr>td { padding:9px 10px!important; background:var(--paper); border-bottom:1px solid var(--line); vertical-align:middle; }
        .node-status .ant-table-tbody>tr:hover>td { background:#f0f0eb!important; }
        .node-status .ant-table-cell-fix-left { background:var(--paper)!important; }
        .node-status__identity,.node-status__traffic,.node-status__system { display:flex; flex-direction:column; gap:3px; min-width:0; }
        .node-status__node-name { display:flex; align-items:center; gap:7px; min-width:0; }
        .node-status__node-name strong { overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
        .node-status__dot { width:7px; height:7px; flex:none; border-radius:50%; background:#aaa; box-shadow:0 0 0 2px #ddd; }
        .node-status__dot.is-online { background:#258554; box-shadow:0 0 0 2px #cce4d7; }
        .node-status__mono,.node-status__traffic { font:11px ui-monospace,SFMono-Regular,Consolas,monospace; }
        .node-status__mono,.node-status__system span,.node-status__hint { color:#747474; }
        .node-status__traffic span { white-space:nowrap; }
        .node-status__uptime { color:#444; white-space:nowrap; }
        .node-status__uptime .anticon { margin-right:4px; color:#777; }
        .node-status__progress { width:100%; }
        .node-status__progress>div { display:flex; justify-content:space-between; gap:5px; margin-bottom:1px; font-size:10px; white-space:nowrap; }
        .node-status__progress strong { font-size:11px; }
        .node-status__progress span { color:#747474; overflow:hidden; text-overflow:ellipsis; }
        .node-status__progress .ant-progress { margin:0; line-height:1; }
        .node-status__progress .ant-progress-inner { border-radius:0; }
        .node-status__progress .ant-progress-bg { border-radius:0!important; }
        .node-status__hint { display:block; overflow:hidden; margin-top:2px; font-size:9px; text-overflow:ellipsis; white-space:nowrap; }
        .node-status__system strong,.node-status__system span { overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
        .node-status__system span { font-size:10px; }
        @media (max-width:700px) {
          .node-status { padding:20px 12px 36px; }
          .node-status__header { align-items:flex-start; flex-direction:column; gap:13px; }
          .node-status__connection { padding:0; }
          .node-status__summary { grid-template-columns:repeat(2,minmax(0,1fr)); }
          .node-status__summary>div { padding:12px 10px; border-bottom:1px solid var(--line); }
          .node-status__summary>div:nth-child(odd) { padding-left:0; }
          .node-status__summary>div:nth-child(even) { border-right:0; }
          .node-status__summary>div:nth-child(n+3) { border-bottom:0; }
          .node-status__summary strong { font-size:19px; }
          .node-status__summary strong.is-online { font-size:25px; }
          .node-status__group { padding-top:24px; }
          .node-status__group-heading { align-items:flex-start; flex-direction:column; gap:8px; }
          .node-status__group-speed { width:100%; justify-content:space-between; }
        }
      `}</style>
    </main>
  )
}
