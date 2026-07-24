import axios from 'axios'

const api = axios.create({ baseURL: '/api', timeout: 15000 })

export const errorMessage = (err: any, fallback: string) =>
  err?.response?.data?.error || (err?.code === 'ECONNABORTED' ? '请求超时' : fallback)

api.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

api.interceptors.response.use(
  res => res,
  err => {
    if (err.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  }
)

export default api

export const login = (username: string, password: string) =>
  api.post('/login', { username, password })
export const register = (data: { username: string; password: string; display_name?: string }) => api.post('/register', data)
export const getSiteSettings = () => api.get('/site')

export const getProfile = () => api.get('/profile')
export const changePassword = (oldPwd: string, newPwd: string) =>
  api.put('/password', { old_password: oldPwd, new_password: newPwd })

export const listMyDeviceGroups = () => api.get('/device_groups')
export const listForwardRules = () => api.get('/forward_rules')
export const createForwardRule = (data: any) => api.post('/forward_rules', data)
export const updateForwardRule = (id: number, data: any) => api.put(`/forward_rules/${id}`, data)
export const deleteForwardRule = (id: number) => api.delete(`/forward_rules/${id}`)
export const toggleForwardRule = (id: number) => api.put(`/forward_rules/${id}/toggle`)
export const batchCreateRules = (rules: any[]) => api.post('/forward_rules/batch', rules)

export const adminDashboard = () => api.get('/admin/dashboard')
export const getAdminSettings = () => api.get('/admin/settings')
export const updateAdminSettings = (data: any) => api.put('/admin/settings', data)
export const listUsers = () => api.get('/admin/users')
export const createUser = (data: any) => api.post('/admin/users', data)
export const updateUser = (id: number, data: any) => api.put(`/admin/users/${id}`, data)
export const deleteUser = (id: number) => api.delete(`/admin/users/${id}`)

export const listUserGroups = () => api.get('/admin/user_groups')
export const createUserGroup = (data: any) => api.post('/admin/user_groups', data)
export const updateUserGroup = (id: number, data: any) => api.put(`/admin/user_groups/${id}`, data)
export const deleteUserGroup = (id: number) => api.delete(`/admin/user_groups/${id}`)

export const listDeviceGroups = () => api.get('/admin/device_groups')
export const createDeviceGroup = (data: any) => api.post('/admin/device_groups', data)
export const updateDeviceGroup = (id: number, data: any) => api.put(`/admin/device_groups/${id}`, data)
export const deleteDeviceGroup = (id: number) => api.delete(`/admin/device_groups/${id}`)
export const getDeviceGroupNodeToken = (id: number) => api.get(`/admin/device_groups/${id}/node-token`)
export const resetDeviceGroupNodeToken = (id: number) => api.post(`/admin/device_groups/${id}/reset-node-token`)

export const listNodes = () => api.get('/admin/nodes')
export const registerNode = (data: any) => api.post('/admin/nodes/register', data)
export const getNodeSetup = (id: number) => api.post(`/admin/nodes/${id}/setup`)
export const rotateNodeToken = (id: number) => api.post(`/admin/nodes/${id}/rotate-token`)
export const deleteNode = (id: number) => api.delete(`/admin/nodes/${id}`)

export const listMyServers = () => api.get('/user-nodes')
export const createMyServer = (data: any) => api.post('/user-nodes', data)
export const deleteMyServer = (id: number) => api.delete(`/user-nodes/${id}`)
export const getMyServerSetup = (id: number) => api.get(`/user-nodes/${id}/setup`)

export const listPlans = () => api.get('/plans')
export const listOrders = () => api.get('/orders')
export const createOrder = (data: any) => api.post('/orders', data)
export const adminListPlans = () => api.get('/admin/plans')
export const adminCreatePlan = (data: any) => api.post('/admin/plans', data)
export const adminUpdatePlan = (id: number, data: any) => api.put(`/admin/plans/${id}`, data)
export const adminDeletePlan = (id: number) => api.delete(`/admin/plans/${id}`)
export const adminListOrders = (status?: string) => api.get('/admin/orders', { params: status ? { status } : {} })
export const adminApproveOrder = (id: number, data: any) => api.post(`/admin/orders/${id}/approve`, data)
export const adminRejectOrder = (id: number, data: any) => api.post(`/admin/orders/${id}/reject`, data)
export const purchasePlanWithBalance = (planId: number, key: string) => api.post('/orders/balance', { plan_id: planId }, { headers: { 'Idempotency-Key': key } })
export const listBalanceLedger = () => api.get('/balance/ledger')
export const listRechargeProviders = () => api.get('/recharge/providers')
export const createRecharge = (data: any, key: string) => api.post('/recharge', data, { headers: { 'Idempotency-Key': key } })
export const listRechargeOrders = () => api.get('/recharge/orders')
export const adminSetBalance = (userId: number, targetBalanceCents: number, reason: string, key: string) => api.post(`/admin/users/${userId}/balance-adjustments`, { target_balance_cents: targetBalanceCents, reason }, { headers: { 'Idempotency-Key': key } })
export const adminListBalanceLedger = (userId: number) => api.get(`/admin/users/${userId}/balance-ledger`)

export const nodeHeartbeat = (token: string, ip: string) =>
  api.post('/node/heartbeat', { token, ip })
