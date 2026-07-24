import { useState } from 'react'
import { Card, Form, Input, Button, Typography, message } from 'antd'
import { UserOutlined, LockOutlined, SendOutlined } from '@ant-design/icons'
import { errorMessage, login } from '../api'
import { useNavigate } from 'react-router-dom'
import { Link } from 'react-router-dom'
import { useSite, useSiteBackground } from '../site'

const { Title, Text } = Typography

export default function Login() {
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()
  const { settings } = useSite()
  const background = useSiteBackground()

  const handleSubmit = async (values: { username: string; password: string }) => {
    setLoading(true)
    try {
      const res = await login(values.username, values.password)
      const token = res.data.token || res.data.access_token
      if (typeof token !== 'string' || !token) throw new Error('invalid token')
      localStorage.setItem('token', token)
      message.success('登录成功')
      navigate('/')
    } catch (err) {
      message.error(errorMessage(err, '登录失败，请检查网络或服务状态'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ height: '100vh', display: 'flex', justifyContent: 'center', alignItems: 'center', padding: 20, backgroundColor: '#f0f2f5', ...background }}>
      <Card style={{ width: 400, boxShadow: '0 2px 8px rgba(0,0,0,0.09)' }}>
        <div style={{ textAlign: 'center', marginBottom: 24 }}>
          <Title level={3} style={{ margin: 0 }}>{settings.site_name}</Title>
          <Text type="secondary">{settings.site_subtitle}</Text>
        </div>
        <Form onFinish={handleSubmit} size="large">
          <Form.Item name="username" rules={[{ required: true, message: '请输入用户名' }]}>
            <Input prefix={<UserOutlined />} placeholder="用户名" />
          </Form.Item>
          {settings.allow_register && <div style={{ textAlign: 'center' }}><Link to="/register">注册新账户</Link></div>}
          <Form.Item name="password" rules={[{ required: true, message: '请输入密码' }]}>
            <Input.Password prefix={<LockOutlined />} placeholder="密码" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" block loading={loading} icon={<SendOutlined />}>
              登 录
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  )
}
