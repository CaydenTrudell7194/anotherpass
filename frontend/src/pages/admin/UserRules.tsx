import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Table, Button, Space, Tag, Switch, Popconfirm, message, Tooltip } from 'antd'
import { ArrowLeftOutlined, ReloadOutlined, PlayCircleOutlined, PauseCircleOutlined } from '@ant-design/icons'
import { listForwardRules as apiListRules, deleteForwardRule, toggleForwardRule } from '../../api'

export default function UserRules() {
  const { userId } = useParams()
  const navigate = useNavigate()
  const [rules, setRules] = useState<any[]>([])
  const [loading, setLoading] = useState(false)

  const fetchRules = async () => {
    setLoading(true)
    try {
      const res = await apiListRules()
      const allRules = Array.isArray(res.data) ? res.data : []
      setRules(allRules.filter((r: any) => String(r.user_id) === userId))
    } catch {
      message.error('获取规则失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchRules() }, [userId])

  const handleToggle = async (id: number) => {
    try { await toggleForwardRule(id); message.success('状态切换成功'); fetchRules() }
    catch { message.error('操作失败') }
  }

  const handleDelete = async (id: number) => {
    try { await deleteForwardRule(id); message.success('删除成功'); fetchRules() }
    catch { message.error('删除失败') }
  }

  const formatBytes = (bytes: number) => {
    if (!bytes) return '0 B'
    const units = ['B', 'KB', 'MB', 'GB', 'TB']
    let i = 0; let v = bytes
    while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
    return `${v.toFixed(2)} ${units[i]}`
  }

  const columns = [
    { title: '规则名', dataIndex: 'name', key: 'name', ellipsis: true },
    { title: '分类', dataIndex: 'category', key: 'category', width: 100, render: (c: string) => <Tag color="blue">{c}</Tag> },
    { title: '监听端口', dataIndex: 'listen_port', key: 'listen_port', width: 100 },
    { title: '目标地址', key: 'dest', width: 200, render: (_: any, r: any) => `${r.target_addr}:${r.target_port}` },
    { title: '流量', key: 'traffic', width: 120, render: (_: any, r: any) => <Tooltip title={`已用 ${formatBytes(r.traffic || 0)}`}><span>{formatBytes(r.traffic || 0)}</span></Tooltip> },
    { title: '状态', dataIndex: 'enabled', key: 'enabled', width: 80, render: (v: boolean) => <Tag color={v ? 'green' : 'default'}>{v ? '启用' : '停用'}</Tag> },
    {
      title: '操作', key: 'action', width: 120,
      render: (_: any, r: any) => (
        <Space>
          <Switch checked={r.enabled} checkedChildren={<PlayCircleOutlined />} unCheckedChildren={<PauseCircleOutlined />} onChange={() => handleToggle(r.id)} />
          <Popconfirm title="确定删除?" onConfirm={() => handleDelete(r.id)}>
            <Button type="link" danger>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/admin/users')} style={{ marginRight: 12 }}>返回用户管理</Button>
        <Button icon={<ReloadOutlined />} onClick={fetchRules}>刷新</Button>
      </div>
      <Card title={`用户 #${userId} 的转发规则 (${rules.length} 条)`}>
        <Table rowKey="id" columns={columns} dataSource={rules} loading={loading} pagination={{ pageSize: 20 }} scroll={{ x: 800 }} size="small" />
      </Card>
    </div>
  )
}
