import { useEffect, useState } from 'react'
import { Button, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Switch, Table, Tag, message } from 'antd'
import { DeleteOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import { adminCreatePlan, adminDeletePlan, adminListPlans, adminUpdatePlan, errorMessage, listUserGroups } from '../../api'

interface Plan {
  id: number
  name: string
  description: string
  price_cents: number
  duration_days: number
  rule_limit: number
  user_group_id?: number
  enabled: boolean
  created_at: string
}

interface UserGroup { id: number; name: string }

const formatPrice = (cents: number) => `¥${(cents / 100).toFixed(2)}`

export default function Plans() {
  const [plans, setPlans] = useState<Plan[]>([])
  const [groups, setGroups] = useState<UserGroup[]>([])
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Plan | null>(null)
  const [form] = Form.useForm()

  const fetchData = async () => {
    setLoading(true)
    try {
      const [plansRes, groupsRes] = await Promise.all([adminListPlans(), listUserGroups()])
      setPlans(plansRes.data || plansRes)
      setGroups(groupsRes.data || groupsRes)
    } catch (err) {
      message.error(errorMessage(err, '获取套餐数据失败'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchData() }, [])

  const openCreate = () => {
    setEditing(null)
    form.setFieldsValue({ enabled: true, duration_days: 30, rule_limit: 0 })
    setModalOpen(true)
  }

  const openEdit = (plan: Plan) => {
    setEditing(plan)
    form.setFieldsValue({ ...plan, price_yuan: plan.price_cents / 100 })
    setModalOpen(true)
  }

  const submit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)
      const { price_yuan, ...fields } = values
      const payload = { ...fields, price_cents: Math.round(price_yuan * 100), user_group_id: fields.user_group_id || null }
      if (editing) {
        await adminUpdatePlan(editing.id, payload)
        message.success('套餐已更新')
      } else {
        await adminCreatePlan(payload)
        message.success('套餐已创建')
      }
      setModalOpen(false)
      await fetchData()
    } catch (err: any) {
      if (err?.errorFields) return
      message.error(errorMessage(err, '保存套餐失败'))
    } finally {
      setSubmitting(false)
    }
  }

  const remove = async (id: number) => {
    try {
      await adminDeletePlan(id)
      message.success('套餐已删除')
      await fetchData()
    } catch (err) {
      message.error(errorMessage(err, '删除套餐失败'))
    }
  }

  const groupName = (id?: number) => id ? groups.find(group => group.id === id)?.name || `#${id}` : '保持当前用户组'
  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: '套餐名称', dataIndex: 'name', ellipsis: true },
    { title: '价格', dataIndex: 'price_cents', width: 100, render: formatPrice },
    { title: '有效期', dataIndex: 'duration_days', width: 90, render: (days: number) => `${days} 天` },
    { title: '规则上限', dataIndex: 'rule_limit', width: 100, render: (limit: number) => limit > 0 ? `${limit} 条` : '不限' },
    { title: '目标用户组', dataIndex: 'user_group_id', width: 150, render: groupName },
    { title: '状态', dataIndex: 'enabled', width: 80, render: (enabled: boolean) => <Tag color={enabled ? 'green' : 'default'}>{enabled ? '上架' : '下架'}</Tag> },
    { title: '创建时间', dataIndex: 'created_at', width: 150, render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm') },
    { title: '操作', key: 'action', fixed: 'right' as const, width: 150, render: (_: unknown, plan: Plan) => <Space size={0}>
      <Button type="link" icon={<EditOutlined />} onClick={() => openEdit(plan)}>编辑</Button>
      <Popconfirm title="确定删除此套餐？" description="已有订单引用的套餐无法删除。" onConfirm={() => remove(plan.id)} okText="删除" cancelText="取消">
        <Button type="link" danger icon={<DeleteOutlined />}>删除</Button>
      </Popconfirm>
    </Space> },
  ]

  return <div>
    <Button type="primary" icon={<PlusOutlined />} onClick={openCreate} style={{ marginBottom: 16 }}>新建套餐</Button>
    <Table rowKey="id" size="small" columns={columns} dataSource={plans} loading={loading} scroll={{ x: 1050 }} pagination={{ pageSize: 20, showSizeChanger: true }} />
    <Modal
      title={editing ? '编辑套餐' : '新建套餐'}
      open={modalOpen}
      onCancel={() => setModalOpen(false)}
      onOk={submit}
      confirmLoading={submitting}
      okText="保存"
      cancelText="取消"
      destroyOnClose
      width={600}
    >
      <Form form={form} layout="vertical" preserve={false}>
        <Form.Item name="name" label="套餐名称" rules={[{ required: true, message: '请输入套餐名称' }, { max: 100 }]}><Input /></Form.Item>
        <Form.Item name="description" label="套餐说明" rules={[{ max: 500 }]}><Input.TextArea rows={3} showCount maxLength={500} /></Form.Item>
        <Space size={12} align="start" style={{ display: 'flex' }}>
          <Form.Item name="price_yuan" label="价格（元）" rules={[{ required: true, message: '请输入价格' }]} style={{ flex: 1 }}><InputNumber min={0} precision={2} style={{ width: '100%' }} /></Form.Item>
          <Form.Item name="duration_days" label="有效期（天）" rules={[{ required: true, message: '请输入有效期' }]} style={{ flex: 1 }}><InputNumber min={1} precision={0} style={{ width: '100%' }} /></Form.Item>
          <Form.Item name="rule_limit" label="规则上限" tooltip="0 表示不限" rules={[{ required: true, message: '请输入规则上限' }]} style={{ flex: 1 }}><InputNumber min={0} precision={0} style={{ width: '100%' }} /></Form.Item>
        </Space>
        <Form.Item name="user_group_id" label="审核通过后用户组" tooltip="不选择则保持用户当前所属组">
          <Select allowClear placeholder="保持当前用户组" options={groups.map(group => ({ value: group.id, label: group.name }))} />
        </Form.Item>
        <Form.Item name="enabled" label="上架状态" valuePropName="checked"><Switch checkedChildren="上架" unCheckedChildren="下架" /></Form.Item>
      </Form>
    </Modal>
  </div>
}
