import { useState, useEffect } from 'react'
import { Card, Form, Input, Button, Descriptions, Tag, message, Spin, Table } from 'antd'
import { UserOutlined, LockOutlined, KeyOutlined, SaveOutlined, WalletOutlined } from '@ant-design/icons'
import { getProfile, changePassword, listBalanceLedger } from '../../api'

export default function Profile() {
  const [user, setUser] = useState<any>(null)
  const [loading, setLoading] = useState(true)
  const [changing, setChanging] = useState(false)
  const [form] = Form.useForm()
  const [ledger, setLedger] = useState<any[]>([])

  useEffect(() => {
    getProfile()
      .then(res => setUser(res.data))
      .catch(() => message.error('获取用户信息失败'))
      .finally(() => setLoading(false))
    listBalanceLedger().then(res=>setLedger(res.data||[])).catch(()=>{})
  }, [])

  const handleChangePassword = async (values: { old_password: string; new_password: string; confirm: string }) => {
    if (values.new_password !== values.confirm) {
      message.error('两次输入的新密码不一致')
      return
    }
    setChanging(true)
    try {
      await changePassword(values.old_password, values.new_password)
      message.success('密码修改成功')
      form.resetFields()
    } catch {
      message.error('密码修改失败，请检查旧密码是否正确')
    } finally {
      setChanging(false)
    }
  }

  if (loading) return <Spin size="large" style={{ display: 'block', margin: '80px auto' }} />

  return (
    <div style={{ maxWidth: 720, margin: '0 auto' }}>
      <Card title={<><UserOutlined /> 个人信息</>} style={{ marginBottom: 16 }}>
        <Descriptions bordered column={{ xs: 1, sm: 2 }} size="small">
          <Descriptions.Item label="用户名">{user?.username}</Descriptions.Item>
          <Descriptions.Item label="显示名">{user?.display_name || '-'}</Descriptions.Item>
          <Descriptions.Item label="邮箱">{user?.email || '-'}</Descriptions.Item>
          <Descriptions.Item label="用户组">{user?.group_name || '-'}</Descriptions.Item>
          <Descriptions.Item label="角色">
            <Tag color={user?.is_admin ? 'red' : 'blue'}>{user?.is_admin ? '管理员' : '普通用户'}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="余额">¥{((user?.balance_cents||0)/100).toFixed(2)}</Descriptions.Item>
          <Descriptions.Item label="过期时间">
            {user?.expire_at ? new Date(user.expire_at).toLocaleDateString() : '永久'}
          </Descriptions.Item>
          <Descriptions.Item label="创建时间">
            {user?.created_at ? new Date(user.created_at).toLocaleString() : '-'}
          </Descriptions.Item>
        </Descriptions>
      </Card>
      <Card title={<><WalletOutlined /> 余额流水</>} style={{marginBottom:16}}>
        <Table size="small" rowKey="id" pagination={{pageSize:8}} dataSource={ledger} columns={[
          {title:'时间',dataIndex:'created_at',render:(v:string)=>new Date(v).toLocaleString()},
          {title:'类型',dataIndex:'kind'},
          {title:'变动',dataIndex:'delta_cents',render:(v:number)=><Tag color={v>=0?'green':'red'}>{v>=0?'+':''}¥{(v/100).toFixed(2)}</Tag>},
          {title:'余额',dataIndex:'balance_after_cents',render:(v:number)=>`¥${(v/100).toFixed(2)}`},
          {title:'备注',dataIndex:'note'}
        ]}/>
      </Card>

      <Card title={<><LockOutlined /> 修改密码</>}>
        <Form form={form} layout="vertical" onFinish={handleChangePassword} style={{ maxWidth: 400 }}>
          <Form.Item name="old_password" label="旧密码" rules={[{ required: true, message: '请输入旧密码' }]}>
            <Input.Password prefix={<KeyOutlined />} placeholder="请输入旧密码" />
          </Form.Item>
          <Form.Item name="new_password" label="新密码" rules={[{ required: true, min: 8, message: '新密码至少8位' }]}>
            <Input.Password prefix={<LockOutlined />} placeholder="请输入新密码" />
          </Form.Item>
          <Form.Item name="confirm" label="确认新密码" dependencies={['new_password']}
            rules={[
              { required: true, message: '请确认新密码' },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('new_password') === value) return Promise.resolve()
                  return Promise.reject(new Error('两次输入的密码不一致'))
                },
              }),
            ]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder="请再次输入新密码" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={changing} icon={<SaveOutlined />}>
              修改密码
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  )
}
