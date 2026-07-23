import { useState, useEffect } from 'react'
import { Card, Table, Tag, message, Spin, Tooltip } from 'antd'
import { DesktopOutlined, WifiOutlined } from '@ant-design/icons'
import { listMyDeviceGroups } from '../../api'

export default function DeviceGroups() {
  const [groups, setGroups] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    listMyDeviceGroups()
      .then(res => setGroups(Array.isArray(res.data) ? res.data : res.data?.device_groups || []))
      .catch(() => message.error('获取设备组列表失败'))
      .finally(() => setLoading(false))
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
    {
      title: '排序', key: 'index', width: 60,
      render: (_: any, __: any, i: number) => i + 1,
    },
    { title: '名称', dataIndex: 'name', key: 'name' },
    {
      title: '类型', dataIndex: 'type', key: 'type',
      render: (type: string) => type ? <Tag>{type.toUpperCase()}</Tag> : '-',
    },
    { title: '连接地址', dataIndex: 'connection_addr', key: 'connection_addr', render: (v: string) => v || '-' },
    {
      title: '已用流量', dataIndex: 'traffic_used', key: 'traffic_used',
      render: (v: number) => formatBytes(v),
    },
    {
      title: '在线设备', dataIndex: 'online_devices', key: 'online_devices',
      render: (v: number) => (
        <span>
          <WifiOutlined style={{ color: v ? '#52c41a' : '#ff4d4f', marginRight: 4 }} />
          {v || 0}
        </span>
      ),
    },
    { title: '备注', dataIndex: 'notes', key: 'notes', render: (v: string) => v || '-', ellipsis: true },
    {
      title: '操作', key: 'action', width: 120,
      render: (_: any, r: any) => (
        <Tooltip title={r.connection_addr ? `连接地址: ${r.connection_addr}` : '无连接信息'}>
          <Tag color={r.online_devices ? 'success' : 'default'}>{r.online_devices ? '可连接' : '离线'}</Tag>
        </Tooltip>
      ),
    },
  ]

  if (loading) return <Spin size="large" style={{ display: 'block', margin: '80px auto' }} />

  return (
    <Card title={<><DesktopOutlined /> 单端隧道</>}>
      <Table
        rowKey="id"
        columns={columns}
        dataSource={groups}
        pagination={{ pageSize: 20, showSizeChanger: true, showTotal: t => `共 ${t} 条` }}
        scroll={{ x: 800 }}
        size="small"
      />
    </Card>
  )
}
