import { useState, useEffect } from 'react'
import { Card, Row, Col, Statistic, Descriptions, Tag, Spin, message } from 'antd'
import {
  UserOutlined, UnorderedListOutlined, ApiOutlined,
  TeamOutlined, ThunderboltOutlined, ClockCircleOutlined
} from '@ant-design/icons'
import { getProfile, listForwardRules, listMyDeviceGroups } from '../../api'

export default function Home() {
  const [user, setUser] = useState<any>(null)
  const [rules, setRules] = useState<any[]>([])
  const [deviceGroups, setDeviceGroups] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  const fetchData = async () => {
    setLoading(true)
    try {
      const [profileRes, rulesRes, dgRes] = await Promise.all([
        getProfile(), listForwardRules(), listMyDeviceGroups()
      ])
      setUser(profileRes.data)
      setRules(Array.isArray(rulesRes.data) ? rulesRes.data : rulesRes.data?.rules || [])
      setDeviceGroups(Array.isArray(dgRes.data) ? dgRes.data : dgRes.data?.device_groups || [])
    } catch {
      message.error('获取数据失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchData() }, [])

  if (loading) return <Spin size="large" style={{ display: 'block', margin: '80px auto' }} />

  const onlineNodes = deviceGroups.reduce((sum: number, g: any) => sum + (g.online_devices || 0), 0)
  const totalTraffic = rules.reduce((sum: number, r: any) => sum + (r.traffic || 0), 0)

  const formatBytes = (bytes: number) => {
    if (!bytes) return '0 B'
    const units = ['B', 'KB', 'MB', 'GB', 'TB']
    let i = 0
    let v = bytes
    while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
    return `${v.toFixed(2)} ${units[i]}`
  }

  return (
    <div>
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={16}>
          <Card title={<><UserOutlined /> 用户信息</>}>
            <Descriptions column={{ xs: 1, sm: 2 }} bordered size="small">
              <Descriptions.Item label="用户名">{user?.username}</Descriptions.Item>
              <Descriptions.Item label="显示名">{user?.display_name || '-'}</Descriptions.Item>
              <Descriptions.Item label="用户组">{user?.group_name || '-'}</Descriptions.Item>
              <Descriptions.Item label="角色">
                <Tag color={user?.is_admin ? 'red' : 'blue'}>{user?.is_admin ? '管理员' : '普通用户'}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="已用流量">{formatBytes(totalTraffic)}</Descriptions.Item>
              <Descriptions.Item label="过期时间">
                {user?.expired_at ? new Date(user.expired_at).toLocaleDateString() : '永久'}
              </Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Row gutter={[16, 16]}>
            <Col span={24}>
              <Card>
                <Statistic title="转发规则数" value={rules.length} prefix={<UnorderedListOutlined />} />
              </Card>
            </Col>
            <Col span={24}>
              <Card>
                <Statistic title="在线节点数" value={onlineNodes} prefix={<ApiOutlined />} suffix={`/ ${deviceGroups.length}`} />
              </Card>
            </Col>
          </Row>
        </Col>
      </Row>
    </div>
  )
}
