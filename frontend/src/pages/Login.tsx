import { useState, useEffect, useRef } from 'react'
import { Card, Form, Input, Button, Typography, message } from 'antd'
import { UserOutlined, LockOutlined, SendOutlined } from '@ant-design/icons'
import { errorMessage, login } from '../api'
import { useNavigate } from 'react-router-dom'
import { Link } from 'react-router-dom'
import { useSite } from '../site'

const { Title, Text } = Typography

export default function Login() {
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()
  const { settings } = useSite()
  const canvasRef = useRef<HTMLCanvasElement>(null)

  useEffect(() => {
    // 强制设置登录页为黑底科技感，卸载时恢复用户原主题
    const wasDark = localStorage.getItem('app-theme') !== 'light'
    document.documentElement.classList.add('my-theme-dark')
    document.documentElement.classList.remove('my-theme-light')

    // 粒子背景与网格呼吸灯动效
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

    // 粒子配置
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

      // 1. 绘制网格背景
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

      // 2. 绘制呼吸灯光效
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

      // 3. 绘制带有漫反射科技色彩的光晕
      ctx.fillStyle = `rgba(0, 212, 255, ${glowOpacity})`
      ctx.beginPath()
      ctx.arc(width / 2, height / 2, 300, 0, Math.PI * 2)
      ctx.fill()

      // 4. 绘制飘动粒子
      particles.forEach(p => {
        p.x += p.vx
        p.y += p.vy
        p.alpha += p.pulseSpeed
        if (p.alpha > 1 || p.alpha < 0.2) p.pulseSpeed = -p.pulseSpeed

        // 边界碰撞
        if (p.x < 0 || p.x > width) p.vx = -p.vx
        if (p.y < 0 || p.y > height) p.vy = -p.vy

        ctx.fillStyle = `rgba(0, 212, 255, ${p.alpha * 0.5})`
        ctx.beginPath()
        ctx.arc(p.x, p.y, p.size, 0, Math.PI * 2)
        ctx.fill()

        // 粒子连线 (只连比较近的)
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
    <div style={{ position: 'relative', height: '100vh', overflow: 'hidden', display: 'flex', justifyContent: 'center', alignItems: 'center', padding: 20 }}>
      {/* 科技背景 Canvas */}
      <canvas ref={canvasRef} style={{ position: 'absolute', top: 0, left: 0, width: '100%', height: '100%', zIndex: 0, pointerEvents: 'none' }} />
      
      <Card style={{ width: 400, zIndex: 1, boxShadow: '0 8px 32px rgba(0, 0, 0, 0.5)' }}>
        <div style={{ textAlign: 'center', marginBottom: 24 }}>
          <Title level={3} style={{ margin: 0, textShadow: '0 0 8px rgba(0, 212, 255, 0.4)' }}>{settings.site_name}</Title>
          <Text type="secondary" style={{ color: 'rgba(0, 212, 255, 0.65)' }}>{settings.site_subtitle}</Text>
        </div>
        <Form onFinish={handleSubmit} size="large">
          <Form.Item name="username" rules={[{ required: true, message: '请输入用户名' }]}>
            <Input prefix={<UserOutlined />} placeholder="用户名" />
          </Form.Item>
          <Form.Item name="password" rules={[{ required: true, message: '请输入密码' }]}>
            <Input.Password prefix={<LockOutlined />} placeholder="密码" />
          </Form.Item>
          {settings.allow_register && (
            <div style={{ textAlign: 'right', marginBottom: 24 }}>
              <Link to="/register" style={{ color: '#00d4ff', textShadow: '0 0 4px rgba(0, 212, 255, 0.2)' }}>注册新账户</Link>
            </div>
          )}
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
