import { Routes, Route, Navigate } from 'react-router-dom'
import Login from './pages/Login'
import MainLayout from './components/MainLayout'
import Home from './pages/user/Home'
import Profile from './pages/user/Profile'
import ForwardRules from './pages/user/ForwardRules'
import DeviceGroups from './pages/user/DeviceGroups'
import MyServers from './pages/user/MyServers'
import NodeStatus from './pages/user/NodeStatus'
import AdminDashboard from './pages/admin/Dashboard'
import AdminUsers from './pages/admin/Users'
import AdminUserGroups from './pages/admin/UserGroups'
import AdminDeviceGroups from './pages/admin/DeviceGroups'
import Register from './pages/Register'
import AdminSettings from './pages/admin/Settings'
import Plans from './pages/user/Plans'
import AdminPlans from './pages/admin/Plans'
import AdminOrders from './pages/admin/Orders'
import Affiliate from './pages/user/Affiliate'
import RedeemCodes from './pages/admin/RedeemCodes'
import AdminAffiliates from './pages/admin/Affiliates'

const isLoggedIn = () => !!localStorage.getItem('token')

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  if (!isLoggedIn()) return <Navigate to="/login" replace />
  return <>{children}</>
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/register" element={<Register />} />
      <Route path="/" element={<ProtectedRoute><MainLayout /></ProtectedRoute>}>
        <Route index element={<Home />} />
        <Route path="profile" element={<Profile />} />
        <Route path="forward_rules" element={<ForwardRules />} />
        <Route path="device_group" element={<DeviceGroups />} />
        <Route path="my_servers" element={<MyServers />} />
        <Route path="node_status" element={<NodeStatus />} />
        <Route path="plans" element={<Plans />} />
        <Route path="affiliate" element={<Affiliate />} />
        <Route path="admin/dashboard" element={<AdminDashboard />} />
        <Route path="admin/settings" element={<AdminSettings />} />
        <Route path="admin/users" element={<AdminUsers />} />
        <Route path="admin/user_groups" element={<AdminUserGroups />} />
        <Route path="admin/device_groups" element={<AdminDeviceGroups />} />
        <Route path="admin/plans" element={<AdminPlans />} />
        <Route path="admin/orders" element={<AdminOrders />} />
        <Route path="admin/redeem-codes" element={<RedeemCodes />} />
        <Route path="admin/affiliates" element={<AdminAffiliates />} />
      </Route>
    </Routes>
  )
}
