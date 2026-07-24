import React, { useEffect, useState } from 'react'
import { Table, Button, Modal, Form, Input, InputNumber, Select, DatePicker, Switch, Popconfirm, Space, Progress, Tag, message, Tooltip } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import { listUsers, createUser, updateUser, deleteUser, listUserGroups } from '../../api'

interface User {
  id: number
  username: string
  display_name: string
  user_group_id: number
  status: string
  traffic_used: number
  traffic_limit: number
  rule_limit: number
  expire_at: string | null
  created_at: string
}

interface UserGroup {
  id: number
  name: string
}

const Users: React.FC = () => {
  const [users, setUsers] = useState<User[]>([])
  const [userGroups, setUserGroups] = useState<UserGroup[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editingUser, setEditingUser] = useState<User | null>(null)
  const [form] = Form.useForm()
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    fetchUsers()
    fetchUserGroups()
  }, [])

  const fetchUsers = async () => {
    setLoading(true)
    try {
      const res = await listUsers()
      setUsers(res.data || res)
    } catch {
      message.error('获取用户列表失败')
    } finally {
      setLoading(false)
    }
  }

  const fetchUserGroups = async () => {
    try {
      const res = await listUserGroups()
      setUserGroups(res.data || res)
    } catch {
      message.error('获取用户组列表失败')
    }
  }

  const handleAdd = () => {
    setEditingUser(null)
    form.resetFields()
    setModalOpen(true)
  }

  const handleEdit = (record: User) => {
    setEditingUser(record)
    form.setFieldsValue({
      ...record,
      expire_at: record.expire_at ? dayjs(record.expire_at) : null,
      status: record.status === 'active',
    })
    setModalOpen(true)
  }

  const handleDelete = async (id: number) => {
    try {
      await deleteUser(id)
      message.success('删除成功')
      fetchUsers()
    } catch {
      message.error('删除失败')
    }
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)
      const payload = {
        ...values,
        status: values.status ? 'active' : 'disabled',
        expire_at: values.expire_at ? values.expire_at.toISOString() : '0001-01-01T00:00:00Z',
      }
      if (editingUser) {
        await updateUser(editingUser.id, payload)
        message.success('更新成功')
      } else {
        await createUser(payload)
        message.success('创建成功')
      }
      setModalOpen(false)
      fetchUsers()
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
    { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
    { title: '用户名', dataIndex: 'username', key: 'username' },
    { title: '显示名', dataIndex: 'display_name', key: 'display_name' },
    { title: '用户组ID', dataIndex: 'user_group_id', key: 'user_group_id' },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={status === 'active' ? 'green' : 'default'}>{status === 'active' ? '启用' : '禁用'}</Tag>
      ),
    },
    {
      title: '流量/限制',
      key: 'traffic',
      render: (_: any, record: User) => {
        if (record.traffic_limit <= 0) return <Tooltip title={`已用 ${formatBytes(record.traffic_used)}`}>不限</Tooltip>
        const percent = record.traffic_limit > 0 ? Math.min((record.traffic_used / record.traffic_limit) * 100, 100) : 0
        return (
          <Tooltip title={`${formatBytes(record.traffic_used)} / ${formatBytes(record.traffic_limit)}`}>
            <Progress percent={Math.round(percent)} size="small" />
          </Tooltip>
        )
      },
    },
    {
      title: '过期时间',
      dataIndex: 'expire_at',
      key: 'expire_at',
      render: (val: string | null) => (val ? dayjs(val).format('YYYY-MM-DD HH:mm') : '-'),
    },
    {
      title: '操作',
      key: 'action',
      width: 160,
      render: (_: any, record: User) => (
        <Space>
          <Button type="link" icon={<EditOutlined />} onClick={() => handleEdit(record)}>
            编辑
          </Button>
          <Popconfirm title="确定删除此用户？" onConfirm={() => handleDelete(record.id)} okText="确定" cancelText="取消">
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
          添加用户
        </Button>
      </div>
      <Table
        rowKey="id"
        columns={columns}
        dataSource={users}
        loading={loading}
        pagination={{ pageSize: 20, showSizeChanger: true }}
        scroll={{ x: 900 }}
      />
      <Modal
        title={editingUser ? '编辑用户' : '添加用户'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitting}
        destroyOnClose
        width={560}
      >
        <Form form={form} layout="vertical" preserve={false}>
          <Form.Item name="username" label="用户名" rules={[{ required: true, message: '请输入用户名' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="password" label="密码" rules={editingUser ? [] : [{ required: true, message: '请输入密码' }]}>
            <Input.Password placeholder={editingUser ? '留空则不修改' : ''} />
          </Form.Item>
          <Form.Item name="display_name" label="显示名">
            <Input />
          </Form.Item>
          <Form.Item name="user_group_id" label="用户组">
            <Select allowClear placeholder="选择用户组">
              {userGroups.map((g) => (
                <Select.Option key={g.id} value={g.id}>
                  {g.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="traffic_limit" label="流量限制 (字节)" tooltip="0 表示不限制">
            <InputNumber min={0} precision={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="rule_limit" label="规则数限制" tooltip="0 表示不限制">
            <InputNumber min={0} precision={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="status" label="状态" valuePropName="checked" initialValue={true}>
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>
          <Form.Item name="expire_at" label="过期时间">
            <DatePicker showTime style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default Users
