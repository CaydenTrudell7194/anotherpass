import { Button, Card, Form, Input, Spin, Typography, message } from 'antd'
import { LockOutlined, UserAddOutlined, UserOutlined } from '@ant-design/icons'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { useEffect, useRef } from 'react'
import { errorMessage, register } from '../api'
import { useSite } from '../site'

const { Title, Text } = Typography

export default function Register() {
  const { settings, initialized } = useSite()
  const navigate = useNavigate()
  const canvasRef = useRef<HTMLCanvasElement>(null)

  useEffect(() => {
    // 强制设置注册页为黑底科技感，卸载时恢复用户原主题
    const wasDark = localStorage.getItem('app-theme') !== 'light'
    document.documentElement.classList.add('my-theme-dark')
    document.documentElement.classList.remove('my-theme-light')

    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    let animationFrameId: number
    let width = (canvas.width = window.innerWidth)
    let height = (canvas.height = window.innerHeight)

    const handleResize = () => {
      width = canvas.width = window.innerWidth
      height = canvas.height = window.innerHeight
    }
    window.addEventListener('resize', handleResize)

    const particles: Array<{
      x: number
      y: number
      vx: number
      vy: number
      size: number
      alpha: number
      pulseSpeed: number
    }> = []

    for (let i = 0; i < 60; i++) {
      particles.push({
        x: Math.random() * width,
        y: Math.random() * height,
        vx: (Math.random() - 0.5) * 0.5,
        vy: (Math.random() - 0.5) * 0.5,
        size: Math.random() * 2 + 1,
        alpha: Math.random(),
        pulseSpeed: 0.01 + Math.random() * 0.02
      })
    }

    let breathe = 0

    const draw = () => {
      ctx.clearRect(0, 0, width, height)

      // 网格
      ctx.strokeStyle = `rgba(0, 212, 255, ${0.03 + Math.sin(breathe) * 0.01})`
      ctx.lineWidth = 1
      const gridSize = 40
      for (let x = 0; x < width; x += gridSize) {
        ctx.beginPath()
        ctx.moveTo(x, 0)
        ctx.lineTo(x, height)
        ctx.stroke()
      }
      for (let y = 0; y < height; y += gridSize) {
        ctx.beginPath()
        ctx.moveTo(0, y)
        ctx.lineTo(width, y)
        ctx.stroke()
      }

      breathe += 0.015
      const glowOpacity = 0.05 + Math.sin(breathe) * 0.03
      const gradient = ctx.createRadialGradient(
        width / 2,
        height / 2,
        10,
        width / 2,
        height / 2,
        Math.max(width, height) * 0.8
      )
      gradient.addColorStop(0, `rgba(13, 0, 26, 0.9)`)
      gradient.addColorStop(0.5, `rgba(3, 0, 8, 0.95)`)
      gradient.addColorStop(1, `rgba(0, 0, 0, 1)`)
      ctx.fillStyle = gradient
      ctx.fillRect(0, 0, width, height)

      ctx.fillStyle = `rgba(0, 212, 255, ${glowOpacity})`
      ctx.beginPath()
      ctx.arc(width / 2, height / 2, 300, 0, Math.PI * 2)
      ctx.fill()

      particles.forEach(p => {
        p.x += p.vx
        p.y += p.vy
        p.alpha += p.pulseSpeed
        if (p.alpha > 1 || p.alpha < 0.2) p.pulseSpeed = -p.pulseSpeed

        if (p.x < 0 || p.x > width) p.vx = -p.vx
        if (p.y < 0 || p.y > height) p.vy = -p.vy

        ctx.fillStyle = `rgba(0, 212, 255, ${p.alpha * 0.5})`
        ctx.beginPath()
        ctx.arc(p.x, p.y, p.size, 0, Math.PI * 2)
        ctx.fill()

        particles.forEach(other => {
          const dx = p.x - other.x
          const dy = p.y - other.y
          const dist = Math.sqrt(dx * dx + dy * dy)
          if (dist < 100) {
            ctx.strokeStyle = `rgba(0, 212, 255, ${(1 - dist / 100) * 0.08})`
            ctx.lineWidth = 0.5
            ctx.beginPath()
            ctx.moveTo(p.x, p.y)
            ctx.lineTo(other.x, other.y)
            ctx.stroke()
          }
        })
      })

      animationFrameId = requestAnimationFrame(draw)
    }

    draw()

    return () => {
      window.removeEventListener('resize', handleResize)
      cancelAnimationFrame(animationFrameId)
      // 恢复用户原主题
      if (wasDark) {
        document.documentElement.classList.add('my-theme-dark')
        document.documentElement.classList.remove('my-theme-light')
      } else {
        document.documentElement.classList.add('my-theme-light')
        document.documentElement.classList.remove('my-theme-dark')
      }
    }
  }, [])

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
    <div style={{ position: 'relative', minHeight: '100vh', overflow: 'hidden', display: 'flex', justifyContent: 'center', alignItems: 'center', padding: 20 }}>
      <canvas ref={canvasRef} style={{ position: 'absolute', top: 0, left: 0, width: '100%', height: '100%', zIndex: 0, pointerEvents: 'none' }} />
      <Card style={{ width: 420, zIndex: 1, boxShadow: '0 8px 32px rgba(0, 0, 0, 0.5)' }}>
        <Title level={3} style={{ marginBottom: 4, textShadow: '0 0 8px rgba(0, 212, 255, 0.4)' }}>{settings.site_name}</Title>
        <Text type="secondary" style={{ color: 'rgba(0, 212, 255, 0.65)' }}>创建入口直出账户</Text>
        <Form onFinish={submit} size="large" layout="vertical" style={{ marginTop: 24 }}>
          <Form.Item name="username" label={<span style={{ color: 'rgba(0, 212, 255, 0.8)' }}>用户名</span>} rules={[{ required: true }, { max: 64 }]}><Input prefix={<UserOutlined />} /></Form.Item>
          <Form.Item name="display_name" label={<span style={{ color: 'rgba(0, 212, 255, 0.8)' }}>显示名</span>} rules={[{ max: 64 }]}><Input /></Form.Item>
          <Form.Item name="password" label={<span style={{ color: 'rgba(0, 212, 255, 0.8)' }}>密码</span>} rules={[{ required: true, min: 8, message: '密码至少8位' }]}><Input.Password prefix={<LockOutlined />} /></Form.Item>
          <Form.Item name="confirm" label={<span style={{ color: 'rgba(0, 212, 255, 0.8)' }}>确认密码</span>} rules={[{ required: true }]}><Input.Password prefix={<LockOutlined />} /></Form.Item>
          <Button type="primary" htmlType="submit" icon={<UserAddOutlined />} block>注册</Button>
        </Form>
        <div style={{ textAlign: 'center', marginTop: 16 }}>
          <Link to="/login" style={{ color: '#00d4ff', textShadow: '0 0 4px rgba(0, 212, 255, 0.2)' }}>已有账户，返回登录</Link>
        </div>
      </Card>
    </div>
  )
}
