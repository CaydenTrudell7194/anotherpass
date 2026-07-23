import React, { useEffect, useState } from 'react'
import { Table, Button, Modal, Popconfirm, Space, Tag, message, Descriptions, Tooltip } from 'antd'
import { PlusOutlined, DeleteOutlined, CopyOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import { listNodes, registerNode, deleteNode } from '../../api'

interface Node {
  id: number
  name: string
  device_group_id: number
  device_group_name?: string
  ip: string
  status: string
  last_heartbeat: string | null
  traffic_up: number
  traffic_down: number
  created_at: string
}

const Nodes: React.FC = () => {
  const [nodes, setNodes] = useState<Node[]>([])
  const [loading, setLoading] = useState(false)
  const [tokenModalOpen, setTokenModalOpen] = useState(false)
  const [token, setToken] = useState('')
  const [registering, setRegistering] = useState(false)

  useEffect(() => {
    fetchNodes()
  }, [])

  const fetchNodes = async () => {
    setLoading(true)
    try {
      const res = await listNodes()
      setNodes(res.data || res)
    } catch {
      message.error('获取节点列表失败')
    } finally {
      setLoading(false)
    }
  }

  const handleRegister = async () => {
    setRegistering(true)
    try {
      const name = prompt('请输入节点名称:')
      if (!name) { setRegistering(false); return }
      const deviceGroupId = prompt('请输入设备组ID:')
      if (!deviceGroupId) { setRegistering(false); return }
      const res = await registerNode({ name, device_group_id: parseInt(deviceGroupId) })
      setToken(res.data?.token || res.token || '')
      setTokenModalOpen(true)
    } catch {
      message.error('注册节点失败')
    } finally {
      setRegistering(false)
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await deleteNode(id)
      message.success('删除成功')
      fetchNodes()
    } catch {
      message.error('删除失败')
    }
  }

  const handleCopyToken = () => {
    navigator.clipboard.writeText(token)
    message.success('已复制到剪贴板')
  }

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
    { title: '节点名', dataIndex: 'name', key: 'name' },
    {
      title: '设备组',
      dataIndex: 'device_group_name',
      key: 'device_group_name',
      render: (text: string) => text || '-',
    },
    { title: 'IP', dataIndex: 'ip', key: 'ip', render: (val: string) => val || '-' },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={status === 'online' ? 'green' : 'red'}>{status === 'online' ? '在线' : '离线'}</Tag>
      ),
    },
    {
      title: '最后心跳',
      dataIndex: 'last_heartbeat',
      key: 'last_heartbeat',
      render: (val: string | null) => (val ? dayjs(val).format('YYYY-MM-DD HH:mm:ss') : '-'),
    },
    {
      title: '流量(上传/下载)',
      key: 'traffic',
      render: (_: any, record: Node) => (
        <Tooltip title={`上传: ${formatBytes(record.traffic_up)} / 下载: ${formatBytes(record.traffic_down)}`}>
          <span>{formatBytes(record.traffic_up)} / {formatBytes(record.traffic_down)}</span>
        </Tooltip>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_: any, record: Node) => (
        <Popconfirm title="确定删除此节点？" onConfirm={() => handleDelete(record.id)} okText="确定" cancelText="取消">
          <Button type="link" danger icon={<DeleteOutlined />}>
            删除
          </Button>
        </Popconfirm>
      ),
    },
  ]

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleRegister} loading={registering}>
          注册节点
        </Button>
      </div>
      <Table
        rowKey="id"
        columns={columns}
        dataSource={nodes}
        loading={loading}
        pagination={{ pageSize: 20, showSizeChanger: true }}
        scroll={{ x: 1000 }}
      />
      <Modal
        title="节点注册成功"
        open={tokenModalOpen}
        onCancel={() => setTokenModalOpen(false)}
        footer={
          <Button type="primary" icon={<CopyOutlined />} onClick={handleCopyToken}>
            复制 Token
          </Button>
        }
        destroyOnClose
      >
        <Descriptions column={1} bordered size="small">
          <Descriptions.Item label="节点 Token">
            <code style={{ wordBreak: 'break-all', fontSize: 13 }}>{token}</code>
          </Descriptions.Item>
        </Descriptions>
        <div style={{ marginTop: 12, color: '#888', fontSize: 13 }}>
          请妥善保管此 Token，关闭后将无法再次查看。
        </div>
      </Modal>
    </div>
  )
}

export default Nodes
