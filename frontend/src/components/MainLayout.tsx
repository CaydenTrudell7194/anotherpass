import { useState, useEffect } from 'react'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { Layout, Menu, Button, Dropdown, Avatar, Switch } from 'antd'
import type { MenuProps } from 'antd'
import {
  HomeOutlined, UserOutlined, UnorderedListOutlined, DesktopOutlined,
  ApiOutlined, DashboardOutlined, SettingOutlined, TeamOutlined,
  CloudServerOutlined, LogoutOutlined, MoonOutlined, SunOutlined, ShoppingOutlined, FileDoneOutlined,
  GiftOutlined, DollarOutlined
} from '@ant-design/icons'
import { getProfile } from '../api'
import { useSite, useSiteBackground } from '../site'

const { Header, Sider, Content } = Layout

const menuItems = (isAdmin: boolean): MenuProps['items'] => {
  const items: MenuProps['items'] = [
    { key: '/', icon: <HomeOutlined />, label: '主页' },
    { key: '/profile', icon: <UserOutlined />, label: '个人中心' },
    { key: '/forward_rules', icon: <UnorderedListOutlined />, label: '转发规则' },
    { key: '/device_group', icon: <CloudServerOutlined />, label: '单端隧道' },
    { key: '/node_status', icon: <ApiOutlined />, label: '节点状态' },
    { key: '/affiliate', icon: <GiftOutlined />, label: '推广返利' },
    { key: '/plans', icon: <ShoppingOutlined />, label: '套餐与订单' },
  ]
  if (isAdmin) {
    items.push({
      key: 'admin', icon: <SettingOutlined />, label: '管理',
      children: [
        { key: '/admin/dashboard', icon: <DashboardOutlined />, label: '仪表盘' },
        { key: '/admin/settings', icon: <SettingOutlined />, label: '站点设置' },
        { key: '/admin/users', icon: <UserOutlined />, label: '用户管理' },
        { key: '/admin/user_groups', icon: <TeamOutlined />, label: '用户组管理' },
        { key: '/admin/device_groups', icon: <CloudServerOutlined />, label: '设备组管理' },
        { key: '/admin/plans', icon: <ShoppingOutlined />, label: '套餐管理' },
        { key: '/admin/orders', icon: <FileDoneOutlined />, label: '订单审核' },
        { key: '/admin/redeem-codes', icon: <DollarOutlined />, label: '兑换码管理' },
        { key: '/admin/affiliates', icon: <TeamOutlined />, label: '推广管理' },
      ],
    } as MenuProps['items'][number])
  }
  return items
}

export default function MainLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const [user, setUser] = useState<any>(null)
  const [collapsed, setCollapsed] = useState(false)
  const [darkMode, setDarkMode] = useState(false)
  const { settings } = useSite()
  const background = useSiteBackground()

  useEffect(() => {
    getProfile().then(res => setUser(res.data)).catch(() => {})
  }, [])

  const handleLogout = () => {
    localStorage.removeItem('token')
    navigate('/login')
  }

  const selectedKey = location.pathname
  const openKeys = selectedKey.startsWith('/admin') ? ['admin'] : []

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider collapsible collapsed={collapsed} onCollapse={setCollapsed} theme={darkMode ? 'dark' : 'light'}>
        <div style={{ height: 64, display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 'bold', fontSize: collapsed ? 14 : 18 }}>
           {collapsed ? settings.site_name.slice(0, 1) : settings.site_name}
        </div>
        <Menu
          mode="inline"
          selectedKeys={[selectedKey]}
          defaultOpenKeys={openKeys}
          items={menuItems(user?.is_admin)}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Layout>
        <Header style={{ background: darkMode ? '#141414' : '#fff', padding: '0 24px', display: 'flex', justifyContent: 'flex-end', alignItems: 'center', gap: 16 }}>
          <Switch
            checkedChildren={<MoonOutlined />}
            unCheckedChildren={<SunOutlined />}
            checked={darkMode}
            onChange={setDarkMode}
          />
          <Dropdown menu={{
            items: [
              { key: 'profile', icon: <UserOutlined />, label: '个人中心', onClick: () => navigate('/profile') },
              { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: handleLogout }
            ]
          }}>
            <Button type="text" style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
              <Avatar size="small" icon={<UserOutlined />} />
              {user?.display_name || user?.username}
            </Button>
          </Dropdown>
        </Header>
        <Content style={{ padding: 24, ...background }}>
          <div style={settings.theme_policy === 'transparent' ? { background: 'rgba(255,255,255,.88)', borderRadius: 12, padding: 20, backdropFilter: 'blur(8px)' } : {}}><Outlet /></div>
        </Content>
      </Layout>
    </Layout>
  )
}
