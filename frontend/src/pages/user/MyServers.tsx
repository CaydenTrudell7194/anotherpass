import { useEffect, useState } from 'react'
import { Table, Button, Modal, Form, Input, Space, message, Popconfirm, Tag } from 'antd'
import { PlusOutlined, CopyOutlined, DeleteOutlined, CloudServerOutlined } from '@ant-design/icons'
import { listMyServers, createMyServer, deleteMyServer, getMyServerSetup } from '../../api'

interface Server {
  id: number
  name: string
  ip: string
  status: string
  created_at: string
}

export default function MyServers() {
  const [servers, setServers] = useState<Server[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [form] = Form.useForm()
  const [submitting, setSubmitting] = useState(false)
  const [command, setCommand] = useState('')
  const [setupModalOpen, setSetupModalOpen] = useState(false)

  useEffect(() => {
    fetchServers()
  }, [])

  const fetchServers = async () => {
    setLoading(true)
    try {
      const res = await listMyServers()
      setServers(res.data || [])
    } catch {
      message.error('获取服务器列表失败')
    } finally {
      setLoading(false)
    }
  }

  const handleAdd = () => {
    form.resetFields()
    setModalOpen(true)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)
      const res = await createMyServer(values)
      setModalOpen(false)
      message.success('创建成功')
      fetchServers()
      const setupRes = await getMyServerSetup(res.data.id)
      setCommand(setupRes.data.command || setupRes.data)
      setSetupModalOpen(true)
    } catch (err: any) {
      if (err?.errorFields) return
      message.error('创建失败')
    } finally {
      setSubmitting(false)
    }
  }

  const showSetup = async (id: number) => {
    try {
      const res = await getMyServerSetup(id)
      setCommand(res.data.command || res.data)
      setSetupModalOpen(true)
    } catch {
      message.error('获取对接命令失败')
    }
  }

  const copyCommand = async () => {
    try {
      await navigator.clipboard.writeText(command)
      message.success('命令已复制')
    } catch {
      message.warning('浏览器禁止自动复制，请手动复制')
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await deleteMyServer(id)
      message.success('删除成功')
      fetchServers()
    } catch {
      message.error('删除失败')
    }
  }

  const statusMap: Record<string, { color: string; text: string }> = {
    active: { color: 'green', text: '在线' },
    inactive: { color: 'default', text: '离线' },
    pending: { color: 'orange', text: '待激活' },
  }

  const columns = [
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: 'IP', dataIndex: 'ip', key: 'ip', render: (v: string) => v || '-' },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (v: string) => {
        const s = statusMap[v] || { color: 'default', text: v || '未知' }
        return <Tag color={s.color}>{s.text}</Tag>
      },
    },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at', render: (v: string) => v ? new Date(v).toLocaleString('zh-CN', { hour12: false }) : '-' },
    {
      title: '操作',
      key: 'action',
      width: 240,
      render: (_: any, record: Server) => (
        <Space>
          <Button type="link" icon={<CopyOutlined />} onClick={() => showSetup(record.id)}>
            复制对接命令
          </Button>
          <Popconfirm title="确定删除此服务器？" onConfirm={() => handleDelete(record.id)} okText="确定" cancelText="取消">
            <Button type="link" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
          添加服务器
        </Button>
      </div>
      <Table
        rowKey="id"
        columns={columns}
        dataSource={servers}
        loading={loading}
        pagination={{ pageSize: 20, showSizeChanger: true }}
        scroll={{ x: 800 }}
        size="small"
      />
      <Modal
        title="添加服务器"
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitting}
        destroyOnClose
        width={480}
      >
        <Form form={form} layout="vertical" preserve={false}>
          <Form.Item name="name" label="服务器名称" rules={[{ required: true, message: '请输入服务器名称' }]}>
            <Input placeholder="例如: 日本出口" />
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        title="服务器对接命令"
        open={setupModalOpen}
        onCancel={() => setSetupModalOpen(false)}
        footer={<Button type="primary" icon={<CopyOutlined />} onClick={copyCommand}>复制完整命令</Button>}
        width={760}
      >
        <Input.TextArea value={command} readOnly autoSize={{ minRows: 5 }} style={{ fontFamily: 'monospace' }} />
        <div style={{ marginTop: 12, color: '#666' }}>在目标服务器上执行此命令完成对接。</div>
      </Modal>
    </div>
  )
}
