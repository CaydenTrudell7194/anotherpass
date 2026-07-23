import { Routes, Route, Navigate } from 'react-router-dom'
import Login from './pages/Login'
import MainLayout from './components/MainLayout'
import Home from './pages/user/Home'
import Profile from './pages/user/Profile'
import ForwardRules from './pages/user/ForwardRules'
import DeviceGroups from './pages/user/DeviceGroups'
import NodeStatus from './pages/user/NodeStatus'
import AdminDashboard from './pages/admin/Dashboard'
import AdminUsers from './pages/admin/Users'
import AdminUserGroups from './pages/admin/UserGroups'
import AdminDeviceGroups from './pages/admin/DeviceGroups'
import AdminNodes from './pages/admin/Nodes'

const isLoggedIn = () => !!localStorage.getItem('token')

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  if (!isLoggedIn()) return <Navigate to="/login" replace />
  return <>{children}</>
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/" element={<ProtectedRoute><MainLayout /></ProtectedRoute>}>
        <Route index element={<Home />} />
        <Route path="profile" element={<Profile />} />
        <Route path="forward_rules" element={<ForwardRules />} />
        <Route path="device_group" element={<DeviceGroups />} />
        <Route path="node_status" element={<NodeStatus />} />
        <Route path="admin/dashboard" element={<AdminDashboard />} />
        <Route path="admin/users" element={<AdminUsers />} />
        <Route path="admin/user_groups" element={<AdminUserGroups />} />
        <Route path="admin/device_groups" element={<AdminDeviceGroups />} />
        <Route path="admin/nodes" element={<AdminNodes />} />
      </Route>
    </Routes>
  )
}
