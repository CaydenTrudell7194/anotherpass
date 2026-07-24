import React, { useEffect, useState } from 'react'
import { Table, Button, Modal, Form, Input, InputNumber, Select, Switch, Popconfirm, Space, message, Tooltip, Tag } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, CopyOutlined, KeyOutlined } from '@ant-design/icons'
import { listDeviceGroups, createDeviceGroup, updateDeviceGroup, deleteDeviceGroup, listNodes, deleteNode, getDeviceGroupNodeToken, resetDeviceGroupNodeToken } from '../../api'

interface DeviceGroup {
  id: number
  name: string
  user_group_ids: string
  type: string
  connection_addr: string
  rate: number
  traffic_used: number
  online_devices: number
  notes: string
  sort_order: number
  hide_in_probe: boolean
  created_at: string
}

const TYPE_LABELS: Record<string, string> = {
  entry_force_direct: '入口(强制直出)',
  entry_optional_direct: '入口(可选直出)',
}
const releaseTag = import.meta.env.VITE_RELEASE_TAG || 'latest'

const DeviceGroups: React.FC = () => {
  const [groups, setGroups] = useState<DeviceGroup[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editingGroup, setEditingGroup] = useState<DeviceGroup | null>(null)
  const [form] = Form.useForm()
  const [submitting, setSubmitting] = useState(false)
  const [nodes, setNodes] = useState<any[]>([])
  const [command, setCommand] = useState('')

  useEffect(() => {
    fetchGroups()
  }, [])

  const fetchGroups = async () => {
    setLoading(true)
    try {
      const [res, nodeRes] = await Promise.all([listDeviceGroups(), listNodes()])
      setGroups(res.data || res)
      setNodes(nodeRes.data || [])
    } catch {
      message.error('获取设备组列表失败')
    } finally {
      setLoading(false)
    }
  }

  const buildCommand = (token: string) => {
    const path = releaseTag === 'latest' ? 'latest/download' : `download/${releaseTag}`
    return `curl --proto '=https' --tlsv1.2 -fsSL https://github.com/CaydenTrudell7194/anotherpass/releases/${path}/install-node.sh | sudo bash -s -- --server '${window.location.origin}' --group-token '${token}'`
  }
  const showCommand = async (group: DeviceGroup) => {
    try { setCommand(buildCommand((await getDeviceGroupNodeToken(group.id)).data.token)) }
    catch { message.error('获取安装命令失败') }
  }
  const resetToken = async (group: DeviceGroup) => {
    try {
      const res = await resetDeviceGroupNodeToken(group.id)
      setCommand(buildCommand(res.data.token))
      message.success('Token 已重置，旧节点已断开')
    } catch { message.error('重置失败') }
  }
  const copyCommand = async () => {
    try { await navigator.clipboard.writeText(command); message.success('安装命令已复制') }
    catch { message.warning('浏览器禁止自动复制，请手动复制弹窗中的命令') }
  }
  const removeNode = async (id: number) => {
    try { await deleteNode(id); message.success('节点已删除'); fetchGroups() }
    catch { message.error('删除节点失败') }
  }

  const handleAdd = () => {
    setEditingGroup(null)
    form.resetFields()
    setModalOpen(true)
  }

  const handleEdit = (record: DeviceGroup) => {
    setEditingGroup(record)
    form.setFieldsValue({
      ...record,
      hide_in_probe: !!record.hide_in_probe,
    })
    setModalOpen(true)
  }

  const handleDelete = async (id: number) => {
    try {
      await deleteDeviceGroup(id)
      message.success('删除成功')
      fetchGroups()
    } catch {
      message.error('删除失败')
    }
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)
      if (editingGroup) {
        await updateDeviceGroup(editingGroup.id, values)
        message.success('更新成功')
      } else {
        await createDeviceGroup(values)
        message.success('创建成功')
      }
      setModalOpen(false)
      fetchGroups()
    } catch (err: any) {
      if (err?.errorFields) return
      message.error('操作失败')
    } finally {
      setSubmitting(false)
    }
  }

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const columns = [
    { title: '排序', dataIndex: 'sort_order', key: 'sort_order', width: 60 },
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '用户组ID', dataIndex: 'user_group_ids', key: 'user_group_ids', render: (val: string) => val || '-' },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type: string) => TYPE_LABELS[type] || type,
    },
    { title: '连接地址', dataIndex: 'connection_addr', key: 'connection_addr', render: (val: string) => val || '-' },
    { title: '倍率', dataIndex: 'rate', key: 'rate', render: (val: number) => (val ?? 1) },
    {
      title: '已用流量',
      key: 'traffic',
      render: (_: any, record: DeviceGroup) => (
        <Tooltip title={formatBytes(record.traffic_used || 0)}>
          <span>{formatBytes(record.traffic_used || 0)}</span>
        </Tooltip>
      ),
    },
    { title: '在线节点', dataIndex: 'online_devices', key: 'online_devices' },
    { title: '备注', dataIndex: 'notes', key: 'notes', render: (val: string) => val || '-' },
    {
      title: '操作',
      key: 'action',
      width: 300,
      render: (_: any, record: DeviceGroup) => (
        <Space>
          <Button type="link" icon={<EditOutlined />} onClick={() => handleEdit(record)}>
            编辑
          </Button>
          {TYPE_LABELS[record.type] && <Button type="link" icon={<CopyOutlined />} onClick={() => showCommand(record)}>安装命令</Button>}
          {TYPE_LABELS[record.type] && <Popconfirm title="重置后所有旧节点需重新执行安装命令，确定？" onConfirm={() => resetToken(record)}>
            <Button type="link" icon={<KeyOutlined />}>重置 Token</Button>
          </Popconfirm>}
          <Popconfirm title="确定删除此设备组？" onConfirm={() => handleDelete(record.id)} okText="确定" cancelText="取消">
            <Button type="link" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
          添加设备组
        </Button>
      </div>
      <Table
        rowKey="id"
        columns={columns}
        dataSource={groups}
        loading={loading}
        pagination={{ pageSize: 20, showSizeChanger: true }}
        scroll={{ x: 1100 }}
        expandable={{
          expandedRowRender: group => {
            const rows = nodes.filter(n => n.device_group_id === group.id)
            return <Table size="small" rowKey="id" pagination={false} dataSource={rows} locale={{emptyText:'尚无节点，执行该组安装命令后自动登记'}} columns={[
              {title:'节点',dataIndex:'name'},
              {title:'IP',dataIndex:'ip',render:(v:string)=>v||'-'},
              {title:'状态',dataIndex:'status',render:(v:string)=><Tag color={v==='online'?'green':'red'}>{v==='online'?'在线':'离线'}</Tag>},
              {title:'实例ID',dataIndex:'instance_id',render:(v:string)=>v||'旧版节点'},
              {title:'操作',render:(_:any,n:any)=><Popconfirm title="确定删除该节点记录？" onConfirm={()=>removeNode(n.id)}><Button danger type="link" icon={<DeleteOutlined />}>删除</Button></Popconfirm>}
            ]} />
          }
        }}
      />
      <Modal
        title={editingGroup ? '编辑设备组' : '添加设备组'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitting}
        destroyOnClose
        width={600}
      >
        <Form form={form} layout="vertical" preserve={false}>
          <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入名称' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="user_group_ids" label="用户组ID" tooltip="多个ID用英文逗号分隔">
            <Input placeholder="例如: 1,2,3" />
          </Form.Item>
          <Form.Item name="type" label="类型" rules={[{ required: true, message: '请选择类型' }]}>
            <Select>
              {Object.entries(TYPE_LABELS).map(([value, label]) => (
                <Select.Option key={value} value={value}>
                  {label}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="hide_in_probe" label="隐藏探针" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="notes" label="备注">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item name="rate" label="倍率" initialValue={1}>
            <InputNumber min={0} step={0.01} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="sort_order" label="排序" initialValue={0}>
            <InputNumber precision={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="connection_addr" label="连接地址">
            <Input placeholder="例如: example.com:8080" />
          </Form.Item>
        </Form>
      </Modal>
      <Modal title="设备组节点安装命令" open={!!command} onCancel={()=>setCommand('')} footer={<Button type="primary" icon={<CopyOutlined />} onClick={copyCommand}>复制完整命令</Button>} width={760}>
        <Input.TextArea value={command} readOnly autoSize={{minRows:5}} style={{fontFamily:'monospace'}} />
        <div style={{marginTop:12,color:'#666'}}>该命令长期有效，可在多台服务器重复执行。每台服务器会自动登记为该设备组下的独立节点。</div>
      </Modal>
    </div>
  )
}

export default DeviceGroups
