import { Button, Card, Form, Input, Spin, Typography, message } from 'antd'
import { LockOutlined, UserAddOutlined, UserOutlined } from '@ant-design/icons'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { errorMessage, register } from '../api'
import { useSite, useSiteBackground } from '../site'

const { Title, Text } = Typography

export default function Register() {
  const { settings, initialized } = useSite()
  const background = useSiteBackground()
  const navigate = useNavigate()
  if (!initialized) return <Spin size="large" fullscreen />
  if (!settings.allow_register) return <Navigate to="/login" replace />
  const submit = async (values: any) => {
    if (values.password !== values.confirm) return message.error('两次输入的密码不一致')
    try {
      await register({ username: values.username, password: values.password, display_name: values.display_name })
      message.success('注册成功，请登录')
      navigate('/login')
    } catch (err) {
      message.error(errorMessage(err, '注册失败'))
    }
  }
  return (
    <div style={{ minHeight: '100vh', display: 'flex', justifyContent: 'center', alignItems: 'center', padding: 20, ...background }}>
      <Card style={{ width: 420, boxShadow: '0 12px 36px rgba(0,0,0,.16)' }}>
        <Title level={3} style={{ marginBottom: 4 }}>{settings.site_name}</Title>
        <Text type="secondary">创建入口直出账户</Text>
        <Form onFinish={submit} size="large" layout="vertical" style={{ marginTop: 24 }}>
          <Form.Item name="username" label="用户名" rules={[{ required: true }, { max: 64 }]}><Input prefix={<UserOutlined />} /></Form.Item>
          <Form.Item name="display_name" label="显示名" rules={[{ max: 64 }]}><Input /></Form.Item>
          <Form.Item name="password" label="密码" rules={[{ required: true, min: 8, message: '密码至少8位' }]}><Input.Password prefix={<LockOutlined />} /></Form.Item>
          <Form.Item name="confirm" label="确认密码" rules={[{ required: true }]}><Input.Password prefix={<LockOutlined />} /></Form.Item>
          <Button type="primary" htmlType="submit" icon={<UserAddOutlined />} block>注册</Button>
        </Form>
        <div style={{ textAlign: 'center', marginTop: 16 }}><Link to="/login">已有账户，返回登录</Link></div>
      </Card>
    </div>
  )
}
