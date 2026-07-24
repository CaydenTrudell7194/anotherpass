import React, { useEffect, useState } from 'react'
import { Card, Col, Row, Statistic, Spin, message } from 'antd'
import {
  UserOutlined, SwapOutlined, AppstoreOutlined, ApiOutlined,
  TeamOutlined, ShoppingCartOutlined, CheckCircleOutlined,
  DashboardOutlined, DollarOutlined
} from '@ant-design/icons'
import { adminDashboard } from '../../api'

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
    </div>
  )
}

export default Dashboard
