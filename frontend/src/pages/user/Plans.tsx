import { useEffect, useState } from 'react'
import { Alert, Button, Card, Col, Empty, Form, Input, InputNumber, Modal, Radio, Row, Space, Switch, Table, Tag, Typography, message, Statistic } from 'antd'
import { CheckCircleOutlined, WalletOutlined, SendOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import { errorMessage, listOrders, listPlans, getProfile, purchasePlanWithBalance, listRechargeProviders, createRecharge, redeemCode } from '../../api'

interface Plan {
  id: number
  name: string
  description: string
  price_cents: number
  duration_days: number
  rule_limit: number
  traffic_limit: number
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
  payment_method: string
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
  const [submitting, setSubmitting] = useState(false)
  const [form] = Form.useForm()
  const [profile, setProfile] = useState<any>({balance_cents:0})
  const [balancePlan, setBalancePlan] = useState<Plan | null>(null)
  const [purchaseKey, setPurchaseKey] = useState('')
  const [rechargeOpen, setRechargeOpen] = useState(false)
  const [providers, setProviders] = useState<Record<string,boolean>>({})
  const [rechargeForm] = Form.useForm()
  const [rechargeKey, setRechargeKey] = useState('')
  const [redeemCodeVal, setRedeemCodeVal] = useState('')

  const submitRedeem = async () => {
    if (!redeemCodeVal.trim()) return
    setSubmitting(true)
    try {
      const res = await redeemCode(redeemCodeVal.trim())
      message.success('兑换成功！余额已增加')
      setRedeemCodeVal('')
      fetchData()
    } catch (err) {
      message.error(errorMessage(err, '兑换失败'))
    } finally {
      setSubmitting(false)
    }
  }

  const fetchData = async () => {
    setLoading(true)
    try {
      const [plansRes, ordersRes, profileRes, providersRes] = await Promise.all([listPlans(), listOrders(), getProfile(), listRechargeProviders()])
      setPlans((plansRes.data || plansRes).filter((plan: Plan) => plan.enabled))
      setOrders(ordersRes.data || ordersRes)
      setProfile(profileRes.data || {})
      setProviders(providersRes.data || {})
    } catch (err) {
      message.error(errorMessage(err, '获取套餐和订单失败'))
    } finally {
      setLoading(false)
    }
  }

  const buyWithBalance = async () => {
    if (!balancePlan) return
    try {
      setSubmitting(true)
      await purchasePlanWithBalance(balancePlan.id, purchaseKey)
      message.success('余额购买成功，套餐已立即生效')
      setBalancePlan(null)
      await fetchData()
    } catch (err) { message.error(errorMessage(err, '余额购买失败')) }
    finally { setSubmitting(false) }
  }

  const recharge = async () => {
    try {
      const values = await rechargeForm.validateFields()
      setSubmitting(true)
      const res = await createRecharge({provider:values.provider,amount_cents:Math.round(values.amount_yuan*100)}, rechargeKey)
      window.location.href = res.data.pay_url
    } catch (err:any) { if (!err?.errorFields) message.error(errorMessage(err,'创建充值订单失败')) }
    finally { setSubmitting(false) }
  }

  useEffect(() => { fetchData() }, [])

  const formatTraffic = (bytes: number) => bytes > 0 ? `${(bytes / (1024**3)).toFixed(0)} GB` : '不限'

  const columns = [
    { title: '订单号', dataIndex: 'id', width: 90, render: (id: number) => `#${id}` },
    { title: '套餐', dataIndex: 'plan_name', ellipsis: true },
    { title: '金额', dataIndex: 'plan_price_cents', width: 100, render: formatPrice },
    { title: '时长', dataIndex: 'plan_duration_days', width: 90, render: (days: number) => `${days} 天` },
    { title: '规则上限', dataIndex: 'plan_rule_limit', width: 100, render: (limit: number) => limit > 0 ? `${limit} 条` : '不限' },
    { title: '状态', dataIndex: 'status', width: 90, render: (status: string) => <Tag color={statusMeta[status]?.color}>{statusMeta[status]?.text || status}</Tag> },
    { title: '方式', dataIndex: 'payment_method', width: 90, render:(v:string)=>v==='balance'?'余额':'人工' },
    { title: '提交备注', dataIndex: 'user_note', ellipsis: true, render: (note: string) => note || '-' },
    { title: '处理备注', dataIndex: 'admin_note', ellipsis: true, render: (note: string) => note || '-' },
    { title: '创建时间', dataIndex: 'created_at', width: 150, render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm') },
  ]

  return <div>
    <Card style={{marginBottom:16}}><Row align="middle" justify="space-between"><Col><Statistic title="账户余额" value={(profile.balance_cents||0)/100} precision={2} prefix="¥" /></Col><Col><Button type="primary" icon={<WalletOutlined />} disabled={!Object.values(providers).some(Boolean)} onClick={()=>{setRechargeKey(crypto.randomUUID());setRechargeOpen(true)}}>充值余额</Button></Col></Row></Card>
    <Alert type="info" showIcon style={{ marginBottom: 16 }} message="余额购买立即生效" description="余额不足时可通过已配置的支付渠道充值。" />
    <Typography.Title level={5} style={{ marginTop: 0 }}>可用套餐</Typography.Title>
    {plans.length === 0 && !loading ? <Empty description="暂无可购买套餐" /> : <Row gutter={[12, 12]}>
      {plans.map(plan => <Col xs={24} sm={12} xl={8} key={plan.id}>
        <Card size="small" title={plan.name} extra={<Typography.Text strong>{formatPrice(plan.price_cents)}</Typography.Text>}>
          <Typography.Paragraph type="secondary" ellipsis={{ rows: 2 }} style={{ minHeight: 44, marginBottom: 12 }}>
            {plan.description || '暂无套餐说明'}
          </Typography.Paragraph>
          <Space size={12} wrap style={{ marginBottom: 14 }}>
            <span><CheckCircleOutlined /> {plan.duration_days}天</span>
            <span><CheckCircleOutlined /> 规则{plan.rule_limit > 0 ? `${plan.rule_limit}条` : '不限'}</span>
            <span><CheckCircleOutlined /> 流量{formatTraffic(plan.traffic_limit)}</span>
          </Space>
          <Button type="primary" block icon={<WalletOutlined />} onClick={() => {setBalancePlan(plan);setPurchaseKey(crypto.randomUUID())}}>余额购买</Button>
        </Card>
      </Col>)}
    </Row>}

    <Typography.Title level={5} style={{ marginTop: 24 }}>我的订单</Typography.Title>
    <Table rowKey="id" size="small" columns={columns} dataSource={orders} loading={loading} scroll={{ x: 1100 }} pagination={{ pageSize: 10, showSizeChanger: true }} />

    <Modal title={`余额购买 - ${balancePlan?.name||''}`} open={!!balancePlan} onCancel={()=>setBalancePlan(null)} onOk={buyWithBalance} okText="确认购买" confirmLoading={submitting} okButtonProps={{disabled:!!balancePlan&&profile.balance_cents<balancePlan.price_cents}}>
      {balancePlan && <><Alert type={profile.balance_cents>=balancePlan.price_cents?'info':'error'} showIcon message={`价格 ${formatPrice(balancePlan.price_cents)}，当前余额 ${formatPrice(profile.balance_cents||0)}`} description={profile.balance_cents>=balancePlan.price_cents?`购买后余额 ${formatPrice(profile.balance_cents-balancePlan.price_cents)}`:'余额不足，请先充值'} />
      {profile.balance_cents>=balancePlan.price_cents && <div style={{marginTop:16}}><Switch defaultChecked /> <Typography.Text style={{marginLeft:8}}>到期自动续费（余额充足时自动扣款续费当前套餐）</Typography.Text></div>}</>}
    </Modal>
    <Modal title="充值余额" open={rechargeOpen} onCancel={()=>setRechargeOpen(false)} onOk={recharge} confirmLoading={submitting}>
      <Form form={rechargeForm} layout="vertical" preserve={false}>
        <Form.Item name="amount_yuan" label="充值金额（元）" rules={[{required:true}]}><InputNumber min={1} max={10000000000} precision={2} style={{width:'100%'}} /></Form.Item>
        <Form.Item name="provider" label="支付渠道" rules={[{required:true}]}><Radio.Group>{providers.epay&&<Radio value="epay">易支付</Radio>}{providers.codepay&&<Radio value="codepay">码支付</Radio>}</Radio.Group></Form.Item>
      </Form>
    </Modal>
    <Card size="small" title="兑换码" style={{marginTop:16}}>
      <Space.Compact style={{width:'100%'}}>
        <Input placeholder="请输入兑换码" value={redeemCodeVal} onChange={e => setRedeemCodeVal(e.target.value)} onPressEnter={submitRedeem} />
        <Button type="primary" icon={<SendOutlined />} onClick={submitRedeem} loading={submitting}>兑换</Button>
      </Space.Compact>
    </Card>
  </div>
}
