import { useEffect, useState } from 'react'
import { Alert, Button, Card, Col, Empty, Form, Input, Modal, Row, Space, Table, Tag, Typography, message } from 'antd'
import { CheckCircleOutlined, ShoppingCartOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import { createOrder, errorMessage, listOrders, listPlans } from '../../api'

interface Plan {
  id: number
  name: string
  description: string
  price_cents: number
  duration_days: number
  rule_limit: number
  enabled: boolean
}

interface Order {
  id: number
  plan_name: string
  plan_price_cents: number
  plan_duration_days: number
  plan_rule_limit: number
  status: string
  user_note: string
  admin_note: string
  reviewed_at: string | null
  created_at: string
}

const statusMeta: Record<string, { color: string; text: string }> = {
  pending: { color: 'gold', text: '待处理' },
  approved: { color: 'green', text: '已通过' },
  rejected: { color: 'red', text: '已拒绝' },
}

const formatPrice = (cents: number) => `¥${(cents / 100).toFixed(2)}`

export default function Plans() {
  const [plans, setPlans] = useState<Plan[]>([])
  const [orders, setOrders] = useState<Order[]>([])
  const [loading, setLoading] = useState(false)
  const [selectedPlan, setSelectedPlan] = useState<Plan | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [form] = Form.useForm()

  const fetchData = async () => {
    setLoading(true)
    try {
      const [plansRes, ordersRes] = await Promise.all([listPlans(), listOrders()])
      setPlans((plansRes.data || plansRes).filter((plan: Plan) => plan.enabled))
      setOrders(ordersRes.data || ordersRes)
    } catch (err) {
      message.error(errorMessage(err, '获取套餐和订单失败'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchData() }, [])

  const submitOrder = async () => {
    if (!selectedPlan) return
    try {
      const values = await form.validateFields()
      setSubmitting(true)
      await createOrder({ plan_id: selectedPlan.id, user_note: values.user_note || '' })
      message.success('订单已提交，请等待管理员核实')
      setSelectedPlan(null)
      form.resetFields()
      await fetchData()
    } catch (err: any) {
      if (err?.errorFields) return
      message.error(errorMessage(err, '订单提交失败'))
    } finally {
      setSubmitting(false)
    }
  }

  const columns = [
    { title: '订单号', dataIndex: 'id', width: 90, render: (id: number) => `#${id}` },
    { title: '套餐', dataIndex: 'plan_name', ellipsis: true },
    { title: '金额', dataIndex: 'plan_price_cents', width: 100, render: formatPrice },
    { title: '时长', dataIndex: 'plan_duration_days', width: 90, render: (days: number) => `${days} 天` },
    { title: '规则上限', dataIndex: 'plan_rule_limit', width: 100, render: (limit: number) => limit > 0 ? `${limit} 条` : '不限' },
    { title: '状态', dataIndex: 'status', width: 90, render: (status: string) => <Tag color={statusMeta[status]?.color}>{statusMeta[status]?.text || status}</Tag> },
    { title: '提交备注', dataIndex: 'user_note', ellipsis: true, render: (note: string) => note || '-' },
    { title: '处理备注', dataIndex: 'admin_note', ellipsis: true, render: (note: string) => note || '-' },
    { title: '创建时间', dataIndex: 'created_at', width: 150, render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm') },
  ]

  return <div>
    <Alert
      type="info"
      showIcon
      style={{ marginBottom: 16 }}
      message="套餐订单采用人工审核"
      description="选择套餐并提交申请，套餐将在管理员审核通过后生效。支付功能将在后续版本接入。"
    />
    <Typography.Title level={5} style={{ marginTop: 0 }}>可用套餐</Typography.Title>
    {plans.length === 0 && !loading ? <Empty description="暂无可购买套餐" /> : <Row gutter={[12, 12]}>
      {plans.map(plan => <Col xs={24} sm={12} xl={8} key={plan.id}>
        <Card size="small" title={plan.name} extra={<Typography.Text strong>{formatPrice(plan.price_cents)}</Typography.Text>}>
          <Typography.Paragraph type="secondary" ellipsis={{ rows: 2 }} style={{ minHeight: 44, marginBottom: 12 }}>
            {plan.description || '暂无套餐说明'}
          </Typography.Paragraph>
          <Space size={16} wrap style={{ marginBottom: 14 }}>
            <span><CheckCircleOutlined /> {plan.duration_days} 天</span>
            <span><CheckCircleOutlined /> {plan.rule_limit > 0 ? `${plan.rule_limit} 条规则` : '规则不限'}</span>
          </Space>
          <Button type="primary" block icon={<ShoppingCartOutlined />} onClick={() => { setSelectedPlan(plan); form.resetFields() }}>
            申请套餐
          </Button>
        </Card>
      </Col>)}
    </Row>}

    <Typography.Title level={5} style={{ marginTop: 24 }}>我的订单</Typography.Title>
    <Table rowKey="id" size="small" columns={columns} dataSource={orders} loading={loading} scroll={{ x: 1100 }} pagination={{ pageSize: 10, showSizeChanger: true }} />

    <Modal
      title={`提交订单${selectedPlan ? ` - ${selectedPlan.name}` : ''}`}
      open={!!selectedPlan}
      onCancel={() => setSelectedPlan(null)}
      onOk={submitOrder}
      okText="确认提交"
      cancelText="取消"
      confirmLoading={submitting}
      destroyOnClose
    >
      {selectedPlan && <Alert type="info" showIcon message={`套餐标价 ${formatPrice(selectedPlan.price_cents)}，当前仅记录订单，不进行在线支付`} style={{ marginBottom: 16 }} />}
      <Form form={form} layout="vertical" preserve={false}>
        <Form.Item name="user_note" label="订单备注" rules={[{ max: 500, message: '备注不能超过 500 个字符' }]}>
          <Input.TextArea rows={4} showCount maxLength={500} placeholder="填写需要管理员了解的信息" />
        </Form.Item>
      </Form>
    </Modal>
  </div>
}
