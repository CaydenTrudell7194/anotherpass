import React, { useEffect, useState } from 'react'
import { Card, Col, Row, Statistic, Spin, message } from 'antd'
import {
  UserOutlined, SwapOutlined, AppstoreOutlined, ApiOutlined,
  TeamOutlined, ShoppingCartOutlined, CheckCircleOutlined,
  DashboardOutlined, DollarOutlined
} from '@ant-design/icons'
import { adminDashboard } from '../../api'

interface OrderTrend {
  date: string
  amount_cents: number
}

interface TopTrafficUser {
  username: string
  display_name: string
  traffic_used: number
}

interface DashboardData {
  user_count: number
  active_user_count: number
  rule_count: number
  device_group_count: number
  online_node_count: number
  total_orders: number
  approved_orders: number
  total_traffic: number
  total_recharge_cents: number
  order_trends_7d?: OrderTrend[]
  top_traffic_users?: TopTrafficUser[]
}

const Dashboard: React.FC = () => {
  const [data, setData] = useState<DashboardData | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchData()
  }, [])

  const fetchData = async () => {
    setLoading(true)
    try {
      const res = await adminDashboard()
      setData(res.data)
    } catch {
      message.error('获取仪表盘数据失败')
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return <Spin size="large" style={{ display: 'flex', justifyContent: 'center', marginTop: 120 }} />
  }

  if (!data) return null

  const formatBytes = (bytes: number) => {
    if (!bytes) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const formatYuan = (cents: number) => '¥' + (cents / 100).toFixed(2)

  // 渲染 SVG 订单折线图
  const renderTrendChart = (trends: OrderTrend[] = []) => {
    if (trends.length === 0) return <div style={{ height: 200, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#94a3b8' }}>暂无流水数据</div>
    
    const height = 200
    const width = 500
    const padding = { top: 20, right: 30, bottom: 30, left: 50 }
    
    const maxVal = Math.max(...trends.map(t => t.amount_cents / 100), 100)
    
    // 计算坐标点
    const points = trends.map((t, i) => {
      const x = padding.left + (i * (width - padding.left - padding.right)) / (trends.length - 1)
      const valYuan = t.amount_cents / 100
      const y = height - padding.bottom - ((valYuan * (height - padding.top - padding.bottom)) / maxVal)
      return { x, y, val: valYuan, label: t.date }
    })

    const pathD = `M ${points.map(p => `${p.x} ${p.y}`).join(' L ')}`
    
    // 阴影区域路径
    const areaD = `${pathD} L ${points[points.length - 1].x} ${height - padding.bottom} L ${points[0].x} ${height - padding.bottom} Z`

    return (
      <svg viewBox={`0 0 ${width} ${height}`} width="100%" height={height} style={{ overflow: 'visible' }}>
        <defs>
          <linearGradient id="chartGlow" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#00d4ff" stopOpacity="0.4"/>
            <stop offset="100%" stopColor="#00d4ff" stopOpacity="0"/>
          </linearGradient>
          <linearGradient id="lineGlow" x1="0" y1="0" x2="1" y2="0">
            <stop offset="0%" stopColor="#00d4ff"/>
            <stop offset="50%" stopColor="#ff00ff"/>
            <stop offset="100%" stopColor="#00d4ff"/>
          </linearGradient>
        </defs>
        
        {/* 背景网格线 */}
        {[0, 0.25, 0.5, 0.75, 1].map((ratio, idx) => {
          const y = padding.top + ratio * (height - padding.top - padding.bottom)
          const val = (maxVal * (1 - ratio)).toFixed(0)
          return (
            <g key={idx}>
              <line x1={padding.left} y1={y} x2={width - padding.right} y2={y} stroke="rgba(255, 255, 255, 0.08)" strokeDasharray="3 3" />
              <text x={padding.left - 10} y={y + 4} fill="#64748b" fontSize="10" textAnchor="end">¥{val}</text>
            </g>
          )
        })}

        {/* 区域和折线 */}
        <path d={areaD} fill="url(#chartGlow)" />
        <path d={pathD} fill="none" stroke="url(#lineGlow)" strokeWidth="3" strokeLinecap="round" />

        {/* 数据点与 Tooltip 信息 */}
        {points.map((p, idx) => (
          <g key={idx} className="chart-point">
            <circle cx={p.x} cy={p.y} r="5" fill="#ffffff" stroke="#00d4ff" strokeWidth="2" />
            {/* 隐藏的背景大圆，方便 hover */}
            <circle cx={p.x} cy={p.y} r="15" fill="transparent" style={{ cursor: 'pointer' }} />
            
            {/* 提示文本，hover 时显现 */}
            <g className="chart-tooltip">
              <rect x={p.x - 45} y={p.y - 35} width="90" height="24" rx="4" fill="rgba(15, 23, 42, 0.95)" stroke="#ff00ff" strokeWidth="1" />
              <text x={p.x} y={p.y - 20} fill="#00d4ff" fontSize="10" fontWeight="bold" textAnchor="middle">¥{p.val.toFixed(2)}</text>
            </g>

            {/* X 轴日期 */}
            <text x={p.x} y={height - 8} fill="#64748b" fontSize="9" textAnchor="middle">
              {p.label.substring(5)}
            </text>
          </g>
        ))}
        
        <style>{`
          .chart-point .chart-tooltip {
            opacity: 0;
            transition: opacity 0.2s ease;
            pointer-events: none;
          }
          .chart-point:hover .chart-tooltip {
            opacity: 1;
          }
        `}</style>
      </svg>
    )
  }

  // 渲染 SVG 流量 Top10 柱状图
  const renderBarChart = (users: TopTrafficUser[] = []) => {
    if (users.length === 0) return <div style={{ height: 200, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#94a3b8' }}>暂无流量数据</div>
    
    const height = 200
    const width = 500
    const padding = { top: 20, right: 30, bottom: 40, left: 60 }
    
    const maxVal = Math.max(...users.map(u => u.traffic_used), 1024 * 1024)
    const barWidth = ((width - padding.left - padding.right) / users.length) * 0.6
    const barGap = ((width - padding.left - padding.right) / users.length) * 0.4

    return (
      <svg viewBox={`0 0 ${width} ${height}`} width="100%" height={height} style={{ overflow: 'visible' }}>
        <defs>
          <linearGradient id="barGlow" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#ff00ff"/>
            <stop offset="100%" stopColor="#00d4ff"/>
          </linearGradient>
        </defs>

        {/* 背景网格线 */}
        {[0, 0.25, 0.5, 0.75, 1].map((ratio, idx) => {
          const y = padding.top + ratio * (height - padding.top - padding.bottom)
          const val = maxVal * (1 - ratio)
          return (
            <g key={idx}>
              <line x1={padding.left} y1={y} x2={width - padding.right} y2={y} stroke="rgba(255, 255, 255, 0.08)" strokeDasharray="3 3" />
              <text x={padding.left - 10} y={y + 4} fill="#64748b" fontSize="10" textAnchor="end">{formatBytes(val)}</text>
            </g>
          )
        })}

        {/* 绘制柱子 */}
        {users.map((u, i) => {
          const x = padding.left + i * (barWidth + barGap) + barGap / 2
          const barHeight = (u.traffic_used * (height - padding.top - padding.bottom)) / maxVal
          const y = height - padding.bottom - barHeight

          return (
            <g key={i} className="bar-group">
              {/* 渐变流光柱子 */}
              <rect
                x={x}
                y={y}
                width={barWidth}
                height={barHeight}
                fill="url(#barGlow)"
                rx="3"
                className="bar-rect"
              />

              {/* Tooltip 信息 */}
              <g className="bar-tooltip">
                <rect x={x + barWidth/2 - 50} y={y - 35} width="100" height="24" rx="4" fill="rgba(15, 23, 42, 0.95)" stroke="#00d4ff" strokeWidth="1" />
                <text x={x + barWidth/2} y={y - 20} fill="#00d4ff" fontSize="10" fontWeight="bold" textAnchor="middle">{formatBytes(u.traffic_used)}</text>
              </g>

              {/* X 轴名字 */}
              <text
                x={x + barWidth / 2}
                y={height - 20}
                fill="#64748b"
                fontSize="9"
                textAnchor="end"
                transform={`rotate(-25, ${x + barWidth / 2}, ${height - 20})`}
                style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
              >
                {u.display_name || u.username}
              </text>
            </g>
          )
        })}
        
        <style>{`
          .bar-group .bar-tooltip {
            opacity: 0;
            transition: opacity 0.2s ease;
            pointer-events: none;
          }
          .bar-group:hover .bar-tooltip {
            opacity: 1;
          }
          .bar-rect {
            transition: height 0.3s ease, y 0.3s ease;
          }
          .bar-group:hover .bar-rect {
            filter: brightness(1.2);
          }
        `}</style>
      </svg>
    )
  }

  return (
    <div>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic title="用户数" value={data.user_count} prefix={<UserOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic title="活跃用户数" value={data.active_user_count} prefix={<TeamOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic title="转发规则数" value={data.rule_count} prefix={<SwapOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic title="设备组数" value={data.device_group_count} prefix={<AppstoreOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic title="在线节点数" value={data.online_node_count} prefix={<ApiOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic title="总订单数" value={data.total_orders} prefix={<ShoppingCartOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic title="已审核订单" value={data.approved_orders} prefix={<CheckCircleOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic title="总流量" value={formatBytes(data.total_traffic)} prefix={<DashboardOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic title="充值总额" value={formatYuan(data.total_recharge_cents)} prefix={<DollarOutlined />} />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 24 }}>
        <Col xs={24} lg={12}>
          <Card title="7日订单流水趋势 (折线图)" style={{ overflow: 'hidden' }}>
            {renderTrendChart(data.order_trends_7d)}
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="当日用户已用流量 Top 10 (柱状图)" style={{ overflow: 'hidden' }}>
            {renderBarChart(data.top_traffic_users)}
          </Card>
        </Col>
      </Row>
    </div>
  )
}

export default Dashboard
