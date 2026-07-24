import { useEffect, useState } from 'react'
import { Alert, Button, Form, Input, Modal, Segmented, Space, Table, Tag, Typography, message } from 'antd'
import { CheckOutlined, CloseOutlined, ReloadOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import { adminApproveOrder, adminListOrders, adminRejectOrder, errorMessage } from '../../api'

interface Order {
  id: number
  user_id: number
  plan_name: string
  plan_price_cents: number
  plan_duration_days: number
  plan_rule_limit: number
  status: string
  user_note: string
  admin_note: string
  reviewed_by?: number
  reviewed_at?: string
  created_at: string
}

type StatusFilter = 'pending' | 'approved' | 'rejected' | 'all'
type ReviewAction = 'approve' | 'reject'

const statusMeta: Record<string, { color: string; text: string }> = {
  pending: { color: 'gold', text: '待处理' },
  approved: { color: 'green', text: '已通过' },
  rejected: { color: 'red', text: '已拒绝' },
}

const formatPrice = (cents: number) => `¥${(cents / 100).toFixed(2)}`

export default function Orders() {
  const [orders, setOrders] = useState<Order[]>([])
  const [status, setStatus] = useState<StatusFilter>('pending')
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [reviewing, setReviewing] = useState<{ order: Order; action: ReviewAction } | null>(null)
  const [form] = Form.useForm()

  const fetchOrders = async (nextStatus: StatusFilter = status) => {
    setLoading(true)
    try {
      const res = await adminListOrders(nextStatus === 'all' ? undefined : nextStatus)
      setOrders(res.data || res)
    } catch (err) {
      message.error(errorMessage(err, '获取订单列表失败'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchOrders(status) }, [status])

  const openReview = (order: Order, action: ReviewAction) => {
    setReviewing({ order, action })
    form.resetFields()
  }

  const submitReview = async () => {
    if (!reviewing) return
    try {
      const values = await form.validateFields()
      setSubmitting(true)
      const payload = { admin_note: values.admin_note || '' }
      if (reviewing.action === 'approve') await adminApproveOrder(reviewing.order.id, payload)
      else await adminRejectOrder(reviewing.order.id, payload)
      message.success(reviewing.action === 'approve' ? '订单已通过' : '订单已拒绝')
      setReviewing(null)
      await fetchOrders()
    } catch (err: any) {
      if (err?.errorFields) return
      message.error(errorMessage(err, '订单处理失败'))
    } finally {
      setSubmitting(false)
    }
  }

  const columns = [
    { title: '订单号', dataIndex: 'id', width: 90, render: (id: number) => `#${id}` },
    { title: '用户', dataIndex: 'user_id', width: 90, render: (id: number) => `#${id}` },
    { title: '套餐', dataIndex: 'plan_name', ellipsis: true },
    { title: '金额', dataIndex: 'plan_price_cents', width: 100, render: formatPrice },
    { title: '权益', key: 'benefit', width: 150, render: (_: unknown, order: Order) => `${order.plan_duration_days} 天 / ${order.plan_rule_limit > 0 ? `${order.plan_rule_limit} 条` : '规则不限'}` },
    { title: '状态', dataIndex: 'status', width: 90, render: (value: string) => <Tag color={statusMeta[value]?.color}>{statusMeta[value]?.text || value}</Tag> },
    { title: '用户备注', dataIndex: 'user_note', ellipsis: true, render: (note: string) => note || '-' },
    { title: '审核备注', dataIndex: 'admin_note', ellipsis: true, render: (note: string) => note || '-' },
    { title: '提交时间', dataIndex: 'created_at', width: 150, render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm') },
    { title: '操作', key: 'action', fixed: 'right' as const, width: 150, render: (_: unknown, order: Order) => order.status === 'pending' ? <Space size={0}>
      <Button type="link" icon={<CheckOutlined />} onClick={() => openReview(order, 'approve')}>通过</Button>
      <Button type="link" danger icon={<CloseOutlined />} onClick={() => openReview(order, 'reject')}>拒绝</Button>
    </Space> : <Typography.Text type="secondary">已处理</Typography.Text> },
  ]

  return <div>
    <Alert type="warning" showIcon message="通过订单会立即发放套餐权益，请先核实用户付款信息；订单一经处理不可改为其他状态。" style={{ marginBottom: 16 }} />
    <Space style={{ marginBottom: 16 }} wrap>
      <Segmented
        value={status}
        onChange={value => setStatus(value as StatusFilter)}
        options={[{ label: '待处理', value: 'pending' }, { label: '已通过', value: 'approved' }, { label: '已拒绝', value: 'rejected' }, { label: '全部', value: 'all' }]}
      />
      <Button icon={<ReloadOutlined />} onClick={() => fetchOrders()} loading={loading}>刷新</Button>
    </Space>
    <Table rowKey="id" size="small" columns={columns} dataSource={orders} loading={loading} scroll={{ x: 1250 }} pagination={{ pageSize: 20, showSizeChanger: true }} />
    <Modal
      title={reviewing?.action === 'approve' ? '通过订单' : '拒绝订单'}
      open={!!reviewing}
      onCancel={() => setReviewing(null)}
      onOk={submitReview}
      okText={reviewing?.action === 'approve' ? '确认通过' : '确认拒绝'}
      okButtonProps={{ danger: reviewing?.action === 'reject' }}
      confirmLoading={submitting}
      destroyOnClose
    >
      {reviewing && <Alert
        type={reviewing.action === 'approve' ? 'info' : 'error'}
        showIcon
        message={`订单 #${reviewing.order.id} · ${reviewing.order.plan_name} · ${formatPrice(reviewing.order.plan_price_cents)}`}
        description={`用户 #${reviewing.order.user_id} 备注：${reviewing.order.user_note || '无'}`}
        style={{ marginBottom: 16 }}
      />}
      <Form form={form} layout="vertical" preserve={false}>
        <Form.Item name="admin_note" label="审核备注" rules={[{ max: 500, message: '备注不能超过 500 个字符' }]}>
          <Input.TextArea rows={4} showCount maxLength={500} placeholder="填写核实结果或拒绝原因" />
        </Form.Item>
      </Form>
    </Modal>
  </div>
}
