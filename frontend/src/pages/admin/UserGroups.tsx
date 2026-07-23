import React, { useEffect, useState } from 'react'
import { Table, Button, Modal, Form, Input, Popconfirm, Space, message } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import { listUserGroups, createUserGroup, updateUserGroup, deleteUserGroup } from '../../api'

interface UserGroup {
  id: number
  name: string
  description: string
  created_at: string
}

const UserGroups: React.FC = () => {
  const [groups, setGroups] = useState<UserGroup[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editingGroup, setEditingGroup] = useState<UserGroup | null>(null)
  const [form] = Form.useForm()
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    fetchGroups()
  }, [])

  const fetchGroups = async () => {
    setLoading(true)
    try {
      const res = await listUserGroups()
      setGroups(res.data || res)
    } catch {
      message.error('获取用户组列表失败')
    } finally {
      setLoading(false)
    }
  }

  const handleAdd = () => {
    setEditingGroup(null)
    form.resetFields()
    setModalOpen(true)
  }

  const handleEdit = (record: UserGroup) => {
    setEditingGroup(record)
    form.setFieldsValue(record)
    setModalOpen(true)
  }

  const handleDelete = async (id: number) => {
    try {
      await deleteUserGroup(id)
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
        await updateUserGroup(editingGroup.id, values)
        message.success('更新成功')
      } else {
        await createUserGroup(values)
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

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '描述', dataIndex: 'description', key: 'description', render: (text: string) => text || '-' },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (val: string) => dayjs(val).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作',
      key: 'action',
      width: 160,
      render: (_: any, record: UserGroup) => (
        <Space>
          <Button type="link" icon={<EditOutlined />} onClick={() => handleEdit(record)}>
            编辑
          </Button>
          <Popconfirm title="确定删除此用户组？" onConfirm={() => handleDelete(record.id)} okText="确定" cancelText="取消">
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
          添加用户组
        </Button>
      </div>
      <Table
        rowKey="id"
        columns={columns}
        dataSource={groups}
        loading={loading}
        pagination={{ pageSize: 20, showSizeChanger: true }}
      />
      <Modal
        title={editingGroup ? '编辑用户组' : '添加用户组'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form form={form} layout="vertical" preserve={false}>
          <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入名称' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default UserGroups
