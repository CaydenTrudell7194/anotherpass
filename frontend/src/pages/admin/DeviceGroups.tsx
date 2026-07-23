import React, { useEffect, useState } from 'react'
import { Table, Button, Modal, Form, Input, Select, Switch, Popconfirm, Space, Tag, message, Progress, Tooltip } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons'
import { listDeviceGroups, createDeviceGroup, updateDeviceGroup, deleteDeviceGroup } from '../../api'

interface DeviceGroup {
  id: number
  name: string
  user_group_ids: string
  type: string
  connection_addr: string
  rate: number
  traffic_used: number
  online_devices: number
  notes: string
  sort_order: number
  hide_in_probe: boolean
  created_at: string
}

const TYPE_LABELS: Record<string, string> = {
  entry_force_direct: '入口(强制直出)',
  entry_optional_direct: '入口(可选直出)',
  entry: '入口',
  monitor: '仅监控',
}

const DeviceGroups: React.FC = () => {
  const [groups, setGroups] = useState<DeviceGroup[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editingGroup, setEditingGroup] = useState<DeviceGroup | null>(null)
  const [form] = Form.useForm()
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    fetchGroups()
  }, [])

  const fetchGroups = async () => {
    setLoading(true)
    try {
      const res = await listDeviceGroups()
      setGroups(res.data || res)
    } catch {
      message.error('获取设备组列表失败')
    } finally {
      setLoading(false)
    }
  }

  const handleAdd = () => {
    setEditingGroup(null)
    form.resetFields()
    setModalOpen(true)
  }

  const handleEdit = (record: DeviceGroup) => {
    setEditingGroup(record)
    form.setFieldsValue({
      ...record,
      hide_in_probe: !!record.hide_in_probe,
    })
    setModalOpen(true)
  }

  const handleDelete = async (id: number) => {
    try {
      await deleteDeviceGroup(id)
      message.success('删除成功')
      fetchGroups()
    } catch {
      message.error('删除失败')
    }
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)
      if (editingGroup) {
        await updateDeviceGroup(editingGroup.id, values)
        message.success('更新成功')
      } else {
        await createDeviceGroup(values)
        message.success('创建成功')
      }
      setModalOpen(false)
      fetchGroups()
    } catch (err: any) {
      if (err?.errorFields) return
      message.error('操作失败')
    } finally {
      setSubmitting(false)
    }
  }

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const columns = [
    { title: '排序', dataIndex: 'sort_order', key: 'sort_order', width: 60 },
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '用户组ID', dataIndex: 'user_group_ids', key: 'user_group_ids', render: (val: string) => val || '-' },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type: string) => TYPE_LABELS[type] || type,
    },
    { title: '连接地址', dataIndex: 'connection_addr', key: 'connection_addr', render: (val: string) => val || '-' },
    { title: '倍率', dataIndex: 'rate', key: 'rate', render: (val: number) => (val ?? 1) },
    {
      title: '已用流量',
      key: 'traffic',
      render: (_: any, record: DeviceGroup) => (
        <Tooltip title={formatBytes(record.traffic_used || record.traffic || 0)}>
          <span>{formatBytes(record.traffic_used || record.traffic || 0)}</span>
        </Tooltip>
      ),
    },
    { title: '在线设备', dataIndex: 'online_devices', key: 'online_devices' },
    { title: '备注', dataIndex: 'notes', key: 'notes', render: (val: string) => val || '-' },
    {
      title: '操作',
      key: 'action',
      width: 160,
      render: (_: any, record: DeviceGroup) => (
        <Space>
          <Button type="link" icon={<EditOutlined />} onClick={() => handleEdit(record)}>
            编辑
          </Button>
          <Popconfirm title="确定删除此设备组？" onConfirm={() => handleDelete(record.id)} okText="确定" cancelText="取消">
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
          添加设备组
        </Button>
      </div>
      <Table
        rowKey="id"
        columns={columns}
        dataSource={groups}
        loading={loading}
        pagination={{ pageSize: 20, showSizeChanger: true }}
        scroll={{ x: 1100 }}
      />
      <Modal
        title={editingGroup ? '编辑设备组' : '添加设备组'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitting}
        destroyOnClose
        width={600}
      >
        <Form form={form} layout="vertical" preserve={false}>
          <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入名称' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="user_group_ids" label="用户组ID" tooltip="多个ID用英文逗号分隔">
            <Input placeholder="例如: 1,2,3" />
          </Form.Item>
          <Form.Item name="type" label="类型" rules={[{ required: true, message: '请选择类型' }]}>
            <Select>
              {Object.entries(TYPE_LABELS).map(([value, label]) => (
                <Select.Option key={value} value={value}>
                  {label}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="hide_in_probe" label="隐藏探针" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="notes" label="备注">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item name="rate" label="倍率" initialValue={1}>
            <Input type="number" min={0} step={0.01} />
          </Form.Item>
          <Form.Item name="sort_order" label="排序" initialValue={0}>
            <Input type="number" />
          </Form.Item>
          <Form.Item name="connection_addr" label="连接地址">
            <Input placeholder="例如: example.com:8080" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default DeviceGroups
