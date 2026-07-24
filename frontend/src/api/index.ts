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

export const listNodes = () => api.get('/admin/nodes')
export const registerNode = (data: any) => api.post('/admin/nodes/register', data)
export const deleteNode = (id: number) => api.delete(`/admin/nodes/${id}`)

export const nodeHeartbeat = (token: string, ip: string) =>
  api.post('/node/heartbeat', { token, ip })
