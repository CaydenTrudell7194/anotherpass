import { useEffect, useState } from 'react'
import { Alert, Button, Form, Input, Modal, Popconfirm, Select, Space, Table, Tag, Tooltip, message } from 'antd'
import { CopyOutlined, DeleteOutlined, KeyOutlined, PlusOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import { deleteNode, errorMessage, getNodeSetup, listDeviceGroups, listNodes, registerNode, rotateNodeToken } from '../../api'

interface NodeItem {
  id: number; name: string; device_group_id: number; device_group_name: string; device_group_type: string
  ip: string; status: string; last_heartbeat: string | null; traffic_up: number; traffic_down: number; created_at: string
}

const directTypes = new Set(['entry_force_direct', 'entry_optional_direct'])
const releaseTag = import.meta.env.VITE_RELEASE_TAG || 'latest'

export default function Nodes() {
  const [nodes, setNodes] = useState<NodeItem[]>([])
  const [groups, setGroups] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [form] = Form.useForm()

  const fetchData = async () => {
    setLoading(true)
    try {
      const [nodesRes, groupsRes] = await Promise.all([listNodes(), listDeviceGroups()])
      setNodes(nodesRes.data || [])
      setGroups((groupsRes.data || []).filter((g: any) => directTypes.has(g.type)))
    } catch (err) { message.error(errorMessage(err, '获取节点数据失败')) }
    finally { setLoading(false) }
  }
  useEffect(() => { fetchData() }, [])
  useEffect(() => {
    const timer = window.setInterval(fetchData, 10000)
    return () => window.clearInterval(timer)
  }, [])

  const submit = async (values: any) => {
    setSubmitting(true)
    try {
      const res = await registerNode(values)
      setModalOpen(false)
      form.resetFields()
      await fetchData()
      await copySetupCommand(res.data.node_id)
      message.success('节点已注册，完整安装命令已复制')
    } catch (err) { message.error(errorMessage(err, '注册节点失败')) }
    finally { setSubmitting(false) }
  }

  const commandFor = (code: string) => {
    const server = window.location.origin
    const releasePath = releaseTag === 'latest' ? 'latest/download' : `download/${releaseTag}`
    return `curl --proto '=https' --tlsv1.2 -fsSL https://github.com/CaydenTrudell7194/anotherpass/releases/${releasePath}/install-node.sh | sudo bash -s -- --server '${server}' --enroll '${code}'`
  }
  const copyText = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      const input = document.createElement('textarea')
      input.value = text
      input.style.position = 'fixed'
      input.style.opacity = '0'
      document.body.appendChild(input)
      input.select()
      const copied = document.execCommand('copy')
      input.remove()
      if (!copied) {
        Modal.info({ title: '节点对接命令', width: 720, content: <Input.TextArea value={text} autoSize={{ minRows: 4 }} readOnly /> })
        return false
      }
      return true
    }
  }
  const copySetupCommand = async (id: number) => {
    try {
      const code = (await getNodeSetup(id)).data.enroll_code
      const copied = await copyText(commandFor(code))
      if (copied) message.success('完整对接命令已复制，10 分钟内执行有效')
    } catch (err) { message.error(errorMessage(err, '复制命令失败')) }
  }
  const rotate = async (id: number) => {
    try {
      await rotateNodeToken(id)
      await copySetupCommand(id)
      message.success('Token 已轮换，新的完整安装命令已复制；旧节点会断开')
    } catch (err) { message.error(errorMessage(err, '轮换失败')) }
  }
  const remove = async (id: number) => {
    try { await deleteNode(id); message.success('删除成功'); fetchData() }
    catch (err) { message.error(errorMessage(err, '删除失败')) }
  }
  const formatBytes = (bytes: number) => {
    if (!bytes) return '0 B'; const units = ['B','KB','MB','GB','TB']; let i=0,v=bytes
    while(v>=1024&&i<units.length-1){v/=1024;i++} return `${v.toFixed(2)} ${units[i]}`
  }
  const groupCounts = nodes.reduce<Record<number, number>>((acc, n) => { acc[n.device_group_id]=(acc[n.device_group_id]||0)+1; return acc }, {})
  const columns = [
    { title:'ID', dataIndex:'id', width:60 },
    { title:'节点名', dataIndex:'name' },
    { title:'设备组', key:'group', render:(_:any,n:NodeItem)=><span>{n.device_group_name||`#${n.device_group_id}`} <Tag>{groupCounts[n.device_group_id]||1}台</Tag></span> },
    { title:'IP', dataIndex:'ip', render:(v:string)=>v||'-' },
    { title:'状态', dataIndex:'status', render:(v:string)=><Tag color={v==='online'?'green':'red'}>{v==='online'?'在线':'离线'}</Tag> },
    { title:'最后心跳', dataIndex:'last_heartbeat', render:(v:string|null)=>v&&!v.startsWith('0001')?dayjs(v).format('YYYY-MM-DD HH:mm:ss'):'-' },
    { title:'流量', key:'traffic', render:(_:any,n:NodeItem)=><Tooltip title={`上传 ${formatBytes(n.traffic_up)} / 下载 ${formatBytes(n.traffic_down)}`}><span>{formatBytes(n.traffic_up+n.traffic_down)}</span></Tooltip> },
    { title:'操作', key:'action', width:260, render:(_:any,n:NodeItem)=><Space>
      <Button type="link" icon={<CopyOutlined />} onClick={()=>copySetupCommand(n.id)}>复制对接命令</Button>
      <Popconfirm title="轮换后旧节点立即失效，确定？" onConfirm={()=>rotate(n.id)}><Button type="link" icon={<KeyOutlined />}>轮换</Button></Popconfirm>
      <Popconfirm title="确定删除此节点？" onConfirm={()=>remove(n.id)}><Button type="link" danger icon={<DeleteOutlined />}>删除</Button></Popconfirm>
    </Space> }
  ]
  return <div>
    <Alert type="info" showIcon style={{marginBottom:16}} message="同一设备组可添加多台入口服务器" description="同组所有在线节点通过 WebSocket 接收完全相同的转发规则。每台服务器分别复制自己的完整对接命令执行即可。" />
    <Button type="primary" icon={<PlusOutlined />} onClick={()=>setModalOpen(true)} style={{marginBottom:16}}>添加节点服务器</Button>
    <Table rowKey="id" columns={columns} dataSource={nodes} loading={loading} scroll={{x:1100}} pagination={{pageSize:20}} />
    <Modal title="添加入口节点" open={modalOpen} onCancel={()=>setModalOpen(false)} onOk={()=>form.submit()} confirmLoading={submitting} destroyOnClose>
      <Form form={form} layout="vertical" onFinish={submit} preserve={false}>
        <Form.Item name="name" label="节点名称" rules={[{required:true},{max:128}]}><Input placeholder="例如：AWS 香港 1" /></Form.Item>
        <Form.Item name="device_group_id" label="所属设备组" rules={[{required:true}]}><Select placeholder="选择入口直出设备组" options={groups.map(g=>({value:g.id,label:`${g.name}（当前 ${groupCounts[g.id]||0} 台）`}))} /></Form.Item>
      </Form>
    </Modal>
  </div>
}
