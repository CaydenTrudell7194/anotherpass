import { useState, useEffect, useRef } from 'react'
import { Card, Table, Tag, Badge, message, Spin, Tooltip } from 'antd'
import { ApiOutlined, ReloadOutlined, CheckCircleOutlined, CloseCircleOutlined } from '@ant-design/icons'
import api from '../../api'

export default function NodeStatus() {
  const [nodes, setNodes] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const fetchNodes = async () => {
    try {
      const res = await api.get('/nodes')
      const data = Array.isArray(res.data) ? res.data : res.data?.nodes || []
      setNodes(data)
    } catch {
      message.error('获取节点状态失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchNodes()
    intervalRef.current = setInterval(fetchNodes, 10000)
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  }, [])

  const formatBytes = (bytes: number) => {
    if (!bytes) return '0 B'
    const units = ['B', 'KB', 'MB', 'GB', 'TB']
    let i = 0
    let v = bytes
    while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
    return `${v.toFixed(2)} ${units[i]}`
  }

  const columns = [
    { title: '节点名', dataIndex: 'name', key: 'name' },
    { title: 'IP', dataIndex: 'ip', key: 'ip', render: (v: string) => v || '-' },
    { title: '设备组ID', dataIndex: 'device_group_id', key: 'device_group_id', render: (v: number) => `#${v}` },
    {
      title: '状态', dataIndex: 'status', key: 'status',
      render: (status: string) => (
        <Tag icon={status === 'online' ? <CheckCircleOutlined /> : <CloseCircleOutlined />} color={status === 'online' ? 'success' : 'error'}>
          {status === 'online' ? '在线' : '离线'}
        </Tag>
      ),
    },
    {
      title: '最后心跳', dataIndex: 'last_heartbeat', key: 'last_heartbeat',
      render: (v: string) => (v && v.startsWith('0001') === false) ? new Date(v).toLocaleString() : '-',
    },
    {
      title: '流量', key: 'traffic', width: 150,
      render: (_: any, r: any) => (
        <Tooltip title={`上传: ${formatBytes(r.traffic_up || 0)} / 下载: ${formatBytes(r.traffic_down || 0)}`}>
          <span>↑ {formatBytes(r.traffic_up || 0)} ↓ {formatBytes(r.traffic_down || 0)}</span>
        </Tooltip>
      ),
    },
  ]

  if (loading) return <Spin size="large" style={{ display: 'block', margin: '80px auto' }} />

  const onlineCount = nodes.filter(n => n.status === 'online').length

  return (
    <Card
      title={<><ApiOutlined /> 节点状态 <Badge count={onlineCount} style={{ backgroundColor: '#52c41a' }} showZero /></>}
      extra={<span><ReloadOutlined /> 每10秒自动刷新</span>}
    >
      <Table
        rowKey="id"
        columns={columns}
        dataSource={nodes}
        pagination={{ pageSize: 20, showSizeChanger: true, showTotal: t => `共 ${t} 条` }}
        scroll={{ x: 800 }}
        size="small"
      />
    </Card>
  )
}
