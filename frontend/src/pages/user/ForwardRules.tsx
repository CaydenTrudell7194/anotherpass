import { useState, useEffect } from 'react'
import {
  Card, Table, Button, Modal, Form, Input, InputNumber, Select, message,
  Space, Tag, Switch, Popconfirm, Tabs, Row, Col, Statistic, Tooltip
} from 'antd'
import {
  PlusOutlined, ImportOutlined, ReloadOutlined, DeleteOutlined,
  EditOutlined, PlayCircleOutlined, PauseCircleOutlined, CopyOutlined
} from '@ant-design/icons'
import {
  listForwardRules, createForwardRule, updateForwardRule,
  deleteForwardRule, toggleForwardRule, batchCreateRules, listMyDeviceGroups
} from '../../api'

export default function ForwardRules() {
  const [rules, setRules] = useState<any[]>([])
  const [deviceGroups, setDeviceGroups] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  const [addModalOpen, setAddModalOpen] = useState(false)
  const [batchModalOpen, setBatchModalOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<any>(null)
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([])
  const [activeGroup, setActiveGroup] = useState<string>('all')
  const [submitting, setSubmitting] = useState(false)
  const [form] = Form.useForm()
  const [batchForm] = Form.useForm()

  const fetchData = async () => {
    setLoading(true)
    try {
      const [rulesRes, dgRes] = await Promise.all([listForwardRules(), listMyDeviceGroups()])
      setRules(Array.isArray(rulesRes.data) ? rulesRes.data : rulesRes.data?.rules || [])
      setDeviceGroups(Array.isArray(dgRes.data) ? dgRes.data : dgRes.data?.device_groups || [])
    } catch {
      message.error('获取数据失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchData() }, [])

  const groupMap = new Map(deviceGroups.map(g => [g.id, g]))
  const groupNames = [...new Set(rules.map(r => r.device_group_id).filter(Boolean))] as number[]

  const filteredRules = activeGroup === 'all'
    ? rules
    : rules.filter(r => String(r.device_group_id) === activeGroup)

  const totalTraffic = rules.reduce((sum: number, r: any) => sum + (r.traffic || 0), 0)

  const formatBytes = (bytes: number) => {
    if (!bytes) return '0 B'
    const units = ['B', 'KB', 'MB', 'GB', 'TB']
    let i = 0
    let v = bytes
    while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
    return `${v.toFixed(2)} ${units[i]}`
  }

  const handleToggle = async (id: number) => {
    try {
      await toggleForwardRule(id)
      message.success('状态切换成功')
      fetchData()
    } catch {
      message.error('操作失败')
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await deleteForwardRule(id)
      message.success('删除成功')
      fetchData()
    } catch {
      message.error('删除失败')
    }
  }

  const handleBatchDelete = async () => {
    if (!selectedRowKeys.length) return
    try {
      await Promise.all(selectedRowKeys.map(id => deleteForwardRule(Number(id))))
      message.success('批量删除成功')
      setSelectedRowKeys([])
      fetchData()
    } catch {
      message.error('批量删除失败')
    }
  }

  const parseDestinations = (dest: string) => {
    const lines = dest.split('\n').map(l => l.trim()).filter(Boolean)
    if (lines.length === 0) return { target_addr: '', target_port: 0 }
    const parts = lines[0].split(':')
    if (parts.length === 2) {
      const port = parseInt(parts[1], 10)
      if (!isNaN(port) && port >= 1 && port <= 65535) {
        return { target_addr: parts[0], target_port: port }
      }
    }
    return { target_addr: '', target_port: 0 }
  }

  const handleAdd = async (values: any) => {
    setSubmitting(true)
    try {
      const { target_addr, target_port } = parseDestinations(values.destinations || values.target_addr + ':' + values.target_port)
      const payload = { ...values, target_addr, target_port }
      delete payload.destinations
      if (editingRule) {
        await updateForwardRule(editingRule.id, payload)
        message.success('修改成功')
      } else {
        await createForwardRule(payload)
        message.success('添加成功')
      }
      setAddModalOpen(false)
      setEditingRule(null)
      form.resetFields()
      fetchData()
    } catch {
      message.error(editingRule ? '修改失败' : '添加失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleBatchImport = async (values: { rules_text: string; device_group_id: number }) => {
    const text = values.rules_text.trim()
    let parsed: any[] = []

    try {
      if (text.startsWith('[')) {
        // NY 新格式 (JSON)
        const json = JSON.parse(text)
        parsed = (Array.isArray(json) ? json : [json]).map((r: any) => {
          let target_addr = r.target_addr || r.目标地址 || ''
          let target_port = r.target_port || r.目标端口 || 0
          if (r.dest && Array.isArray(r.dest) && r.dest.length > 0) {
            const parts = r.dest[0].split(':')
            if (parts.length === 2) {
              target_addr = parts[0]
              target_port = parseInt(parts[1], 10) || 0
            }
          }
          return {
            name: r.name || r.规则名 || '',
            device_group_id: values.device_group_id,
            listen_port: r.listen_port || r.监听端口 || 0,
            target_addr,
            target_port,
            protocol: String(r.protocol || r.协议 || 'tcp').toLowerCase(),
          }
        })
      } else {
        // NY 旧格式: 名称#监听端口#目标地址#目标端口
        const lines = text.split('\n').map(l => l.trim()).filter(Boolean)
        parsed = lines.map(line => {
          const parts = line.split('#')
          const [name, listen_port, target_addr, target_port] = parts
          if (parts.length !== 4) throw new Error('invalid fields')
          return { name, device_group_id: values.device_group_id, listen_port: Number(listen_port), target_addr, target_port: Number(target_port), protocol: 'tcp' }
        })
      }
    } catch {
      message.error('解析失败，请检查数据格式')
      return
    }

    if (parsed.length === 0) {
      message.warning('没有有效的规则')
      return
    }
    const invalidIndex = parsed.findIndex(r => !String(r.name || '').trim() || !String(r.target_addr || '').trim() ||
      !Number.isInteger(r.listen_port) || r.listen_port < 1 || r.listen_port > 65535 ||
      !Number.isInteger(r.target_port) || r.target_port < 1 || r.target_port > 65535 || r.protocol !== 'tcp')
    if (invalidIndex >= 0) {
      message.error(`第 ${invalidIndex + 1} 条规则无效；当前仅支持 TCP，端口范围为 1-65535`)
      return
    }

    setSubmitting(true)
    try {
      await batchCreateRules(parsed)
      message.success(`成功导入 ${parsed.length} 条规则`)
      setBatchModalOpen(false)
      batchForm.resetFields()
      fetchData()
    } catch {
      message.error('批量导入失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleBatchExport = () => {
    // NY 新格式 (JSON) with dest
    const exportData = rules.map(r => ({
      name: r.name,
      listen_port: r.listen_port,
      dest: [`${r.target_addr}:${r.target_port}`],
      protocol: r.protocol || 'tcp',
    }))
    const json = JSON.stringify(exportData, null, 2)
    navigator.clipboard.writeText(json)
    message.success(`已复制 ${exportData.length} 条规则 (JSON 格式)`)
  }

  const handleBatchExportLine = () => {
    // NY 旧格式 (行)
    const lines = rules.map(r => `${r.name}#${r.listen_port}#${r.target_addr}#${r.target_port}`)
    const text = lines.join('\n')
    navigator.clipboard.writeText(text)
    message.success(`已复制 ${lines.length} 条规则 (行格式)`)
  }

  const openEdit = (rule: any) => {
    setEditingRule(rule)
    form.setFieldsValue({ ...rule, destinations: `${rule.target_addr}:${rule.target_port}` })
    setAddModalOpen(true)
  }

  const columns = [
    { title: '规则名', dataIndex: 'name', key: 'name', ellipsis: true },
    {
      title: '入口设备组', dataIndex: 'device_group_id', key: 'device_group_id',
      render: (id: number) => groupMap.get(id)?.name || `#${id}`,
    },
    { title: '目标地址', dataIndex: 'target_addr', key: 'target_addr' },
    { title: '监听端口', dataIndex: 'listen_port', key: 'listen_port', width: 100 },
    {
      title: '流量', key: 'traffic', width: 120,
      render: (_: any, r: any) => (
        <Tooltip title={`已用 ${formatBytes(r.traffic || 0)}`}>
          <span>{formatBytes(r.traffic || 0)}</span>
        </Tooltip>
      ),
    },
    {
      title: '状态', dataIndex: 'enabled', key: 'enabled', width: 80,
      render: (enabled: boolean) => <Tag color={enabled ? 'green' : 'default'}>{enabled ? '启用' : '停用'}</Tag>,
    },
    {
      title: '操作', key: 'action', width: 180,
      render: (_: any, r: any) => (
        <Space>
          <Switch
            checked={r.enabled}
            checkedChildren={<PlayCircleOutlined />}
            unCheckedChildren={<PauseCircleOutlined />}
            onChange={() => handleToggle(r.id)}
          />
          <Tooltip title="编辑">
            <Button type="link" icon={<EditOutlined />} onClick={() => openEdit(r)} />
          </Tooltip>
          <Popconfirm title="确定删除?" onConfirm={() => handleDelete(r.id)}>
            <Button type="link" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  const groupTabs = [
    { key: 'all', label: `全部 (${rules.length})` },
    ...groupNames.map(id => ({
      key: String(id),
      label: `${groupMap.get(id)?.name || `#${id}`} (${rules.filter(r => r.device_group_id === id).length})`,
    })),
    { key: 'users', label: '用户节点选择（开发中）' },
  ]

  return (
    <div>
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={12}>
          <Card size="small">
            <Statistic title="总流量使用" value={formatBytes(totalTraffic)} />
          </Card>
        </Col>
        <Col span={12}>
          <Card size="small">
            <Statistic title="规则总数" value={rules.length} />
          </Card>
        </Col>
      </Row>
      <Card
        title="转发规则"
        extra={
          <Space>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => { setEditingRule(null); form.resetFields(); setAddModalOpen(true) }}>
              添加规则
            </Button>
            <Button icon={<ImportOutlined />} onClick={() => { batchForm.resetFields(); setBatchModalOpen(true) }}>
              批量导入
            </Button>
            <Button icon={<CopyOutlined />} onClick={handleBatchExport}>
              批量导出
            </Button>
            <Button icon={<ReloadOutlined />} onClick={fetchData}>刷新</Button>
            <Popconfirm title={`确定删除选中的 ${selectedRowKeys.length} 条规则?`} onConfirm={handleBatchDelete}>
              <Button danger icon={<DeleteOutlined />} disabled={!selectedRowKeys.length}>删除选中</Button>
            </Popconfirm>
          </Space>
        }
      >
        <Tabs activeKey={activeGroup} onChange={setActiveGroup} items={groupTabs} style={{ marginBottom: 8 }} />
        {activeGroup === 'users' ? (
          <div style={{ padding: 40, textAlign: 'center', color: '#999' }}>用户节点选择功能正在开发中</div>
        ) : (
          <Table
            rowKey="id"
            columns={columns}
            dataSource={filteredRules}
            loading={loading}
            rowSelection={{
              selectedRowKeys,
              onChange: (keys) => setSelectedRowKeys(keys),
            }}
            pagination={{ pageSize: 20, showSizeChanger: true, showTotal: t => `共 ${t} 条` }}
            scroll={{ x: 800 }}
            size="small"
          />
        )}
      </Card>

      <Modal
        title={editingRule ? '编辑规则' : '添加规则'}
        open={addModalOpen}
        onCancel={() => { setAddModalOpen(false); setEditingRule(null) }}
        footer={null}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={handleAdd} initialValues={{ protocol: 'tcp' }}>
          <Form.Item name="name" label="规则名" rules={[{ required: true, message: '请输入规则名' }]}>
            <Input placeholder="规则名称" />
          </Form.Item>
          <Form.Item name="device_group_id" label="入口设备组" rules={[{ required: true, message: '请选择设备组' }]}>
            <Select placeholder="请选择设备组">
              {deviceGroups.map(g => (
                <Select.Option key={g.id} value={g.id}>{g.name}</Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="listen_port" label="监听端口" rules={[{ required: true, message: '请输入监听端口' }]}>
            <InputNumber min={1} max={65535} style={{ width: '100%' }} placeholder="例如: 8080" />
          </Form.Item>
          <Form.Item name="destinations" label="目标地址（每行一个 IP:端口）" rules={[{ required: true, message: '请输入目标地址' }]}
            extra="首行作为主目标，支持多行备用"
          >
            <Input.TextArea rows={3} placeholder={"例如:\n192.168.1.100:80\n10.0.0.1:443"} />
          </Form.Item>
          <Form.Item name="target_addr" hidden><Input /></Form.Item>
          <Form.Item name="target_port" hidden><InputNumber /></Form.Item>
          <Form.Item name="protocol" label="协议">
            <Select>
              <Select.Option value="tcp">TCP</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={submitting} block>
              {editingRule ? '保存修改' : '添加'}
            </Button>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="批量导入规则"
        open={batchModalOpen}
        onCancel={() => setBatchModalOpen(false)}
        footer={null}
        destroyOnClose
      >
        <Form form={batchForm} layout="vertical" onFinish={handleBatchImport}>
          <Form.Item name="device_group_id" label="入口设备组" rules={[{ required: true, message: '请选择设备组' }]}>
            <Select placeholder="请选择设备组">
              {deviceGroups.map(g => (
                <Select.Option key={g.id} value={g.id}>{g.name}</Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="rules_text" label="规则数据" rules={[{ required: true, message: '请输入规则数据' }]}
            extra="支持 NY 新旧两种格式：JSON 数组 或 每行 名称#监听端口#目标地址#目标端口"
          >
            <Input.TextArea rows={8} placeholder={'JSON 格式:\n[{"name":"规则1","listen_port":8080,"dest":["192.168.1.1:80"]}]\n\n行格式:\n规则1#8080#192.168.1.1#80'} />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={submitting} block>导入</Button>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
