import { useState, useEffect } from 'react'
import {
  Card, Button, Modal, Form, Input, InputNumber, Select, message,
  Space, Switch, Popconfirm, Row, Col, Checkbox, Tabs, Tooltip, Progress, Tag
} from 'antd'
import {
  PlusOutlined, ReloadOutlined, DeleteOutlined, EditOutlined,
  PlayCircleOutlined, PauseCircleOutlined, CopyOutlined, BugOutlined,
  UpOutlined, DownOutlined
} from '@ant-design/icons'
import {
  listForwardRules, createForwardRule, updateForwardRule,
  deleteForwardRule, toggleForwardRule, batchCreateRules,
  listCategories, createCategory, updateCategory, deleteCategory,
  moveRulesToCategory, duplicateForwardRule, diagnoseForwardRule,
  batchToggleForwardRules, listMyDeviceGroups
} from '../../api'

export default function ForwardRules() {
  const [rules, setRules] = useState<any[]>([])
  const [categories, setCategories] = useState<any[]>([])
  const [deviceGroups, setDeviceGroups] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  const [addModalOpen, setAddModalOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<any>(null)
  const [selectedIds, setSelectedIds] = useState<number[]>([])
  const [activeCategory, setActiveCategory] = useState('全部')
  const [submitting, setSubmitting] = useState(false)
  const [form] = Form.useForm()

  const fetchData = async () => {
    setLoading(true)
    try {
      const [r, c, d] = await Promise.all([listForwardRules(), listCategories(), listMyDeviceGroups()])
      setRules(Array.isArray(r.data) ? r.data : [])
      setCategories(Array.isArray(c.data) ? c.data : [])
      setDeviceGroups(Array.isArray(d.data) ? d.data : [])
    } catch { message.error('获取数据失败') }
    finally { setLoading(false) }
  }

  useEffect(() => { fetchData() }, [])

  const filteredRules = activeCategory === '全部'
    ? rules : rules.filter(r => r.category === activeCategory)

  const formatBytes = (b: number) => {
    if (!b) return '0'; const u = ['B','KB','MB','GB','TB']; let i=0; let v=b
    while(v>=1024&&i<u.length-1){v/=1024;i++}; return `${v.toFixed(v>=100?0:2)}${u[i]}`
  }

  const handleToggle = async (id: number) => {
    try { await toggleForwardRule(id); fetchData() }
    catch { message.error('操作失败') }
  }

  const handleDelete = async (id: number) => {
    try { await deleteForwardRule(id); message.success('删除成功'); fetchData() }
    catch { message.error('删除失败') }
  }

  const handleDuplicate = async (id: number) => {
    try {
      const res = await duplicateForwardRule(id)
      message.success('已复制')
      setEditingRule(res.data)
      form.setFieldsValue({ ...res.data, destinations: `${res.data.target_addr}:${res.data.target_port}` })
      setAddModalOpen(true)
      fetchData()
    } catch { message.error('复制失败') }
  }

  const handleDiagnose = async (id: number) => {
    try {
      const res = await diagnoseForwardRule(id)
      message[res.data.reachable ? 'success' : 'error'](
        res.data.reachable ? '目标可达 ✓' : `目标不可达: ${res.data.error}`
      )
    } catch { message.error('诊断失败') }
  }

  const parseDest = (dest: string) => {
    const lines = dest.split('\n').map(l => l.trim()).filter(Boolean)
    if (!lines.length) return { target_addr: '', target_port: 0 }
    const p = lines[0].split(':')
    if (p.length === 2) { const port = parseInt(p[1]); if (!isNaN(port)&&port>=1&&port<=65535) return { target_addr: p[0], target_port: port } }
    return { target_addr: '', target_port: 0 }
  }

  const handleSubmit = async (values: any) => {
    setSubmitting(true)
    try {
      const { target_addr, target_port } = parseDest(values.destinations || '')
      const payload = { ...values, target_addr, target_port, device_group_id: values.node_id || 0, category: editingRule?.category || activeCategory }
      delete payload.destinations; delete payload.node_id
      if (editingRule) {
        await updateForwardRule(editingRule.id, payload)
        message.success('修改成功')
      } else {
        await createForwardRule(payload)
        message.success('添加成功')
      }
      setAddModalOpen(false); setEditingRule(null); form.resetFields(); fetchData()
    } catch { message.error(editingRule?'修改失败':'添加失败') }
    finally { setSubmitting(false) }
  }

  const openAdd = () => { setEditingRule(null); form.resetFields(); setAddModalOpen(true) }

  const openEdit = (rule: any) => {
    setEditingRule(rule)
    form.setFieldsValue({ ...rule, destinations: `${rule.target_addr}:${rule.target_port}`, node_id: rule.device_group_id })
    setAddModalOpen(true)
  }

  const selectAll = () => {
    if (selectedIds.length === filteredRules.length) setSelectedIds([])
    else setSelectedIds(filteredRules.map(r => r.id))
  }

  const batchToggle = async (enabled: boolean) => {
    if (!selectedIds.length) return
    try { await batchToggleForwardRules(selectedIds, enabled); message.success(enabled?'已启用':'已停用'); setSelectedIds([]); fetchData() }
    catch { message.error('操作失败') }
  }

  const batchDelete = async () => {
    if (!selectedIds.length) return
    try { await Promise.all(selectedIds.map(id=>deleteForwardRule(id))); message.success('已删除'); setSelectedIds([]); fetchData() }
    catch { message.error('删除失败') }
  }

  const rateColor = (rate: number) => rate > 1 ? '#f59e0b' : rate === 1 ? '#10b981' : '#3b82f6'

  return (
    <div>
      {/* 操作栏 */}
      <Row gutter={12} style={{ marginBottom: 16 }} align="middle">
        <Col>
          <Button type="primary" icon={<PlusOutlined />} onClick={openAdd}>新建规则</Button>
        </Col>
        <Col>
          <Button icon={<ReloadOutlined />} onClick={fetchData}>刷新</Button>
        </Col>
        {selectedIds.length > 0 && <>
          <Col><Button icon={<PlayCircleOutlined />} onClick={()=>batchToggle(true)}>批量启用</Button></Col>
          <Col><Button icon={<PauseCircleOutlined />} onClick={()=>batchToggle(false)}>批量停用</Button></Col>
          <Col>
            <Popconfirm title={`确定删除 ${selectedIds.length} 条规则？`} onConfirm={batchDelete}>
              <Button danger icon={<DeleteOutlined />}>批量删除</Button>
            </Popconfirm>
          </Col>
        </>}
      </Row>

      {/* 分类 Tabs + 全选 */}
      <Card size="small" style={{ marginBottom: 16 }}>
        <Tabs
          activeKey={activeCategory}
          onChange={(k) => { setActiveCategory(k); setSelectedIds([]) }}
          tabBarExtraContent={
            <Checkbox checked={selectedIds.length === filteredRules.length && filteredRules.length > 0} onChange={selectAll}>
              全选 ({filteredRules.length})
            </Checkbox>
          }
          items={[
            { key: '全部', label: `全部 (${rules.length})` },
            ...categories.map(c => ({ key: c.name, label: `${c.name} (${rules.filter(r=>r.category===c.name).length})` })),
          ]}
        />
      </Card>

      {/* 规则列表 */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        {loading && <div style={{textAlign:'center',padding:40}}>加载中...</div>}
        {!loading && filteredRules.length === 0 && <div style={{textAlign:'center',padding:40,color:'#666'}}>暂无规则</div>}
        {filteredRules.map(rule => {
          const group = deviceGroups.find(d => d.id === rule.device_group_id)
          const groupName = group?.name || `节点#${rule.device_group_id}`
          const connAddr = group?.connection_addr || groupName
          return (
            <Card
              key={rule.id}
              size="small"
              style={{ opacity: rule.enabled ? 1 : 0.5, borderLeft: `4px solid ${rule.enabled ? '#10b981' : '#6b7280'}` }}
            >
              <Row align="middle" gutter={12}>
                {/* 勾选 */}
                <Col>
                  <Checkbox checked={selectedIds.includes(rule.id)} onChange={e => {
                    if (e.target.checked) setSelectedIds([...selectedIds, rule.id])
                    else setSelectedIds(selectedIds.filter(id => id !== rule.id))
                  }} />
                </Col>

                {/* 名称 + 节点 */}
                <Col flex="1">
                  <div style={{ fontWeight: 600 }}>{rule.name}</div>
                  <div style={{ fontSize: 12, color: '#999' }}>{connAddr}:{rule.listen_port} → {rule.target_addr}:{rule.target_port}</div>
                </Col>

                {/* 倍率 */}
                {group && <Col>
                  <Tag color={rateColor(group.rate||1)}>×{group.rate||1}</Tag>
                </Col>}

                {/* 流量 */}
                <Col style={{ textAlign:'right', minWidth:100 }}>
                  <Progress
                    percent={rule.traffic_limit > 0 ? Math.min(rule.traffic||0 / rule.traffic_limit * 100, 100) : 0}
                    size="small"
                    format={() => formatBytes(rule.traffic||0)}
                    style={{ margin: 0, width:100 }}
                  />
                </Col>

                {/* 操作按钮 */}
                <Col>
                  <Space size={4}>
                    {/* 启停 */}
                    <Tooltip title={rule.enabled ? '停用' : '启用'}>
                      <Button
                        size="small"
                        shape="default"
                        icon={rule.enabled ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
                        onClick={() => handleToggle(rule.id)}
                        style={{ width:32, height:32, padding:0 }}
                      />
                    </Tooltip>
                    {/* 编辑 */}
                    <Tooltip title="编辑">
                      <Button size="small" icon={<EditOutlined />} onClick={() => openEdit(rule)} style={{ width:32, height:32, padding:0 }} />
                    </Tooltip>
                    {/* 诊断 */}
                    <Tooltip title="连通性诊断">
                      <Button size="small" icon={<BugOutlined />} onClick={() => handleDiagnose(rule.id)} style={{ width:32, height:32, padding:0 }} />
                    </Tooltip>
                    {/* 复制 */}
                    <Tooltip title="复制">
                      <Button size="small" icon={<CopyOutlined />} onClick={() => handleDuplicate(rule.id)} style={{ width:32, height:32, padding:0 }} />
                    </Tooltip>
                    {/* 删除 */}
                    <Popconfirm title="确定删除？" onConfirm={() => handleDelete(rule.id)}>
                      <Tooltip title="删除">
                        <Button size="small" danger icon={<DeleteOutlined />} style={{ width:32, height:32, padding:0 }} />
                      </Tooltip>
                    </Popconfirm>
                  </Space>
                </Col>
              </Row>
            </Card>
          )
        })}
      </div>

      {/* 新增/编辑弹窗 */}
      <Modal
        title={editingRule ? '编辑规则' : '新建规则'}
        open={addModalOpen}
        onCancel={() => { setAddModalOpen(false); setEditingRule(null) }}
        footer={null}
        destroyOnClose
        width={560}
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit} initialValues={{ protocol:'tcp+udp', rate:1 }}>
          <Form.Item name="name" label="规则名" rules={[{ required:true }]}>
            <Input placeholder="规则名称" />
          </Form.Item>
          <Form.Item name="node_id" label="选择节点" rules={[{ required:true, message:'请选择节点' }]}>
            <Select placeholder="选择节点">
              {deviceGroups.map(g => (
                <Select.Option key={g.id} value={g.id}>
                  {g.name} {g.rate != null && g.rate !== 1 ? <Tag color="#f59e0b">×{g.rate}</Tag> : <Tag color="#10b981">×1</Tag>}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="listen_port" label="监听端口" rules={[{ required:true }]}>
            <InputNumber min={1} max={65535} style={{width:'100%'}} placeholder="例如 8080" />
          </Form.Item>
          <Form.Item name="destinations" label="目标地址" rules={[{ required:true, message:'请输入目标地址' }]}
            extra="格式 IP:端口，每行一个"
          >
            <Input.TextArea rows={2} placeholder="192.168.1.100:80" />
          </Form.Item>
          <Form.Item name="target_addr" hidden><Input /></Form.Item>
          <Form.Item name="target_port" hidden><InputNumber /></Form.Item>
          <Form.Item name="protocol" label="协议">
            <Select>
              <Select.Option value="tcp">TCP</Select.Option>
              <Select.Option value="udp">UDP</Select.Option>
              <Select.Option value="tcp+udp">TCP+UDP</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="rate" label="计费倍率">
            <Select disabled>
              <Select.Option value={1}>1 (默认)</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={submitting} block>
              {editingRule ? '保存修改' : '添加'}
            </Button>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}