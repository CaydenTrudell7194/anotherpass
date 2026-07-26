import { useState, useEffect } from 'react'
import {
  Card, Table, Button, Modal, Form, Input, InputNumber, Select, message,
  Space, Tag, Popconfirm, Tooltip, Row, Col, Statistic, Collapse, Divider
} from 'antd'
import {
  PlusOutlined, ImportOutlined, CopyOutlined, ReloadOutlined, DeleteOutlined,
  EditOutlined, PlayCircleOutlined, PauseCircleOutlined, BugOutlined,
  SearchOutlined, BarChartOutlined, FireOutlined, CheckSquareOutlined,
  RightOutlined
} from '@ant-design/icons'
import {
  listForwardRules, createForwardRule, updateForwardRule,
  deleteForwardRule, toggleForwardRule,
  listCategories, deleteCategory, createCategory,
  duplicateForwardRule, diagnoseForwardRule,
  batchToggleForwardRules, batchClearTraffic, listMyDeviceGroups
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
  const [searchText, setSearchText] = useState('')
  const [statsOpen, setStatsOpen] = useState(false)
  const [advOpen, setAdvOpen] = useState(false)
  const [catMgrOpen, setCatMgrOpen] = useState(false)
  const [catForm] = Form.useForm()
  const [form] = Form.useForm()
  const groupMap = new Map(deviceGroups.map(d => [d.id, d]))

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

  const filtered = rules.filter(r => {
    if (activeCategory !== '全部' && r.category !== activeCategory) return false
    if (searchText) {
      const q = searchText.toLowerCase()
      return r.name?.toLowerCase().includes(q) || String(r.listen_port).includes(q) || r.target_addr?.toLowerCase().includes(q)
    }
    return true
  })

  const formatBytes = (b: number) => {
    if (!b) return '0 B'; const u = ['B','KB','MB','GB','TB']; let i=0; let v=b
    while(v>=1024&&i<u.length-1){v/=1024;i++}; return `${v.toFixed(v>=100?0:2)} ${u[i]}`
  }

  const handleToggle = async (id: number) => {
    try { await toggleForwardRule(id); fetchData() }
    catch { message.error('操作失败') }
  }

  const handleDelete = async (id: number) => {
    try { await deleteForwardRule(id); fetchData() }
    catch { message.error('删除失败') }
  }

  const handleDuplicate = async (id: number) => {
    try {
      const res = await duplicateForwardRule(id)
      setEditingRule(res.data)
      form.setFieldsValue({ ...res.data, destinations: `${res.data.target_addr}:${res.data.target_port}`, node_id: res.data.device_group_id })
      setAddModalOpen(true)
      fetchData()
    } catch { message.error('复制失败') }
  }

  const handleDiagnose = async (id: number) => {
    try {
      const res = await diagnoseForwardRule(id)
      if (res.data.reachable) message.success(`${res.data.addr||''} 可达 ✓`)
      else message.error(`不可达: ${res.data.error||'超时'}`)
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
      if (editingRule) { await updateForwardRule(editingRule.id, payload); message.success('修改成功') }
      else { await createForwardRule(payload); message.success('添加成功') }
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

  const selectAll = () => { setSelectedIds(selectedIds.length===filtered.length?[]:filtered.map(r=>r.id)) }
  const batchToggle = async (enabled: boolean) => {
    if (!selectedIds.length) return
    try { await batchToggleForwardRules(selectedIds, enabled); fetchData(); setSelectedIds([]); message.success(enabled?'已启用':'已停用') }
    catch { message.error('操作失败') }
  }
  const batchClear = async () => {
    if (!selectedIds.length) return
    try { await batchClearTraffic(selectedIds); fetchData(); message.success('流量已清空') }
    catch { message.error('操作失败') }
  }
  const batchDelete = async () => {
    if (!selectedIds.length) return
    try { await Promise.all(selectedIds.map(id=>deleteForwardRule(id))); setSelectedIds([]); fetchData(); message.success('已删除') }
    catch { message.error('删除失败') }
  }

  // 统计数据
  const stats = { total: rules.length, traffic: rules.reduce((s,r)=>s+(r.traffic||0),0), online: rules.filter(r=>r.enabled).length }

  const totalGB = formatBytes(stats.traffic)

  const columns = [
    { title: '规则名', dataIndex: 'name', key: 'name', ellipsis: true, width: 150 },
    {
      title: '入口', key: 'entry', width: 160,
      render: (_:any, r:any) => {
        const g = groupMap.get(r.device_group_id)
        return <Tag color={r.enabled?'green':'default'}>{g?.name||`#${r.device_group_id}`} <span style={{fontSize:11}}>×{g?.rate||1}</span></Tag>
      }
    },
    {
      title: '出口', key: 'exit', width: 200,
      render: (_:any, r:any) => {
        const destCount = (r.dest?.length) || 1
        return <Tooltip title={`${r.target_addr}:${r.target_port}`}>{r.target_addr}:{r.target_port}{destCount>1?<span style={{color:'#999'}}> 等{destCount}个</span>:''}</Tooltip>
      }
    },
    {
      title: '已用流量', key: 'traffic', width: 110,
      render: (_:any, r:any) => <span>{formatBytes(r.traffic||0)}</span>
    },
    {
      title: '状态', dataIndex: 'enabled', key: 'enabled', width: 60,
      render: (v:boolean) => <Tag color={v?'green':'default'} style={{fontSize:11}}>{v?'正常':'停用'}</Tag>
    },
    {
      title: '操作', key: 'action', width: 210,
      render: (_:any, r:any) => (
        <Space size={4}>
          <Tooltip title={r.enabled?'暂停':'启动'}>
            <Button size="small" shape="default" icon={r.enabled?<PauseCircleOutlined/>:<PlayCircleOutlined/>}
              onClick={()=>handleToggle(r.id)} style={{width:28,height:28,padding:0}} />
          </Tooltip>
          <Tooltip title="诊断"><Button size="small" icon={<BugOutlined/>} onClick={()=>handleDiagnose(r.id)} style={{width:28,height:28,padding:0}} /></Tooltip>
          <Tooltip title="复制"><Button size="small" icon={<CopyOutlined/>} onClick={()=>handleDuplicate(r.id)} style={{width:28,height:28,padding:0}} /></Tooltip>
          <Tooltip title="编辑"><Button size="small" icon={<EditOutlined/>} onClick={()=>openEdit(r)} style={{width:28,height:28,padding:0}} /></Tooltip>
          <Popconfirm title="确定删除？" onConfirm={()=>handleDelete(r.id)}>
            <Tooltip title="删除"><Button size="small" danger icon={<DeleteOutlined/>} style={{width:28,height:28,padding:0}} /></Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  const rowSelection = {
    selectedRowKeys: selectedIds,
    onChange: (keys: React.Key[]) => setSelectedIds(keys as number[]),
  }

  return (
    <div style={{ display:'flex', gap:12 }}>
      {/* 主内容 */}
      <div style={{ flex:1, minWidth:0 }}>
        {/* 头部统计 */}
        <Card size="small" style={{marginBottom:12}} bodyStyle={{padding:'12px 16px'}}>
          <Row align="middle" justify="space-between">
            <Col>
              <Space size={24}>
                <span style={{fontSize:13}}>流量: <b>{totalGB}</b></span>
                <span style={{fontSize:13}}>规则数: <b>{stats.total}</b></span>
              </Space>
            </Col>
            <Col>
              <Space size={8}>
                <Input prefix={<SearchOutlined/>} placeholder="搜索规则" value={searchText} onChange={e=>setSearchText(e.target.value)} style={{width:180}} allowClear />
                <Button icon={<ReloadOutlined/>} onClick={fetchData}>刷新</Button>
                <Button icon={<BarChartOutlined/>} onClick={()=>setStatsOpen(true)}>统计数据</Button>
              </Space>
            </Col>
          </Row>
        </Card>

        {/* 分类 + 操作栏 */}
        <Card size="small" style={{marginBottom:12}} bodyStyle={{padding:'8px 16px'}}>
          <Row align="middle" justify="space-between" style={{marginBottom:8}}>
            <Col>
              <Space size={4}>
                <Button size="small" onClick={()=>setCatMgrOpen(true)} icon={<RightOutlined style={{fontSize:12}} />}>管理分组</Button>
                {categories.map(c => (
                  <Tag key={c.name} color={activeCategory===c.name?'blue':'default'} style={{cursor:'pointer',margin:0}}
                    onClick={()=>{setActiveCategory(c.name);setSelectedIds([])}}>
                    {c.name} ({rules.filter(r=>r.category===c.name).length})
                  </Tag>
                ))}
              </Space>
            </Col>
            <Col>
              <CheckSquareOutlined style={{cursor:'pointer',fontSize:16}} onClick={selectAll} />
            </Col>
          </Row>
          <Space size={8}>
            <Button type="primary" size="small" icon={<PlusOutlined/>} onClick={openAdd}>添加规则</Button>
            <Button size="small" icon={<ImportOutlined/>} onClick={()=>message.info('请使用右侧批量按钮')}>批量导入</Button>
            <Button size="small" icon={<CopyOutlined/>} onClick={()=>{
              const json = JSON.stringify(rules.map(r=>({name:r.name,listen_port:r.listen_port,dest:[`${r.target_addr}:${r.target_port}`],protocol:r.protocol||'tcp'})),null,2)
              navigator.clipboard.writeText(json); message.success('已导出')
            }}>批量导出</Button>
            {selectedIds.length>0 && <>
              <Button size="small" icon={<PlayCircleOutlined/>} onClick={()=>batchToggle(true)}>启动</Button>
              <Button size="small" icon={<PauseCircleOutlined/>} onClick={()=>batchToggle(false)}>暂停</Button>
              <Button size="small" icon={<FireOutlined/>} onClick={batchClear}>清空流量</Button>
              <Popconfirm title={`删除 ${selectedIds.length} 条?`} onConfirm={batchDelete}>
                <Button size="small" danger icon={<DeleteOutlined/>}>删除选中</Button>
              </Popconfirm>
            </>}
          </Space>
        </Card>

        {/* 规则表格 */}
        <Table rowKey="id" size="small" columns={columns} dataSource={filtered} loading={loading}
          rowSelection={rowSelection} pagination={{pageSize:15,showSizeChanger:true,showTotal:t=>`共 ${t} 条`}}
          scroll={{x:900}} />
      </div>

      {/* 右侧竖排操作栏 */}
      <div style={{width:44,display:'flex',flexDirection:'column',gap:4}}>
        <Tooltip title="管理分组" placement="left"><Button size="small" icon={<RightOutlined/>} onClick={()=>setCatMgrOpen(true)} style={{height:40,padding:0}}>分组</Button></Tooltip>
        <Tooltip title="添加单条" placement="left"><Button size="small" icon={<PlusOutlined/>} onClick={openAdd} style={{height:40,padding:0}}>单条</Button></Tooltip>
        <Tooltip title="批量导入" placement="left"><Button size="small" icon={<ImportOutlined/>} onClick={()=>message.info('功能开发中')} style={{height:40,padding:0}}>批量</Button></Tooltip>
        <Tooltip title="导出" placement="left"><Button size="small" icon={<CopyOutlined/>} onClick={()=>{
          const json = JSON.stringify(rules.map(r=>({name:r.name,listen_port:r.listen_port,dest:[`${r.target_addr}:${r.target_port}`],protocol:r.protocol||'tcp'})),null,2)
          navigator.clipboard.writeText(json); message.success('已导出')
        }} style={{height:40,padding:0}}>导出</Button></Tooltip>
        <Tooltip title="批量切换" placement="left"><Button size="small" icon={<CheckSquareOutlined/>} onClick={()=>selectedIds.length?batchToggle(!filtered.some(r=>r.enabled)):message.info('请先勾选规则')} style={{height:40,padding:0}}>切换</Button></Tooltip>
        <Tooltip title="清空流量" placement="left"><Button size="small" icon={<FireOutlined/>} onClick={batchClear} style={{height:40,padding:0}}>流量</Button></Tooltip>
        <Tooltip title="批量暂停" placement="left"><Button size="small" icon={<PauseCircleOutlined/>} onClick={()=>batchToggle(false)} style={{height:40,padding:0}}>暂停</Button></Tooltip>
        <Tooltip title="批量启动" placement="left"><Button size="small" icon={<PlayCircleOutlined/>} onClick={()=>batchToggle(true)} style={{height:40,padding:0}}>启动</Button></Tooltip>
        <Popconfirm title={`删除 ${selectedIds.length} 条?`} onConfirm={batchDelete}>
          <Tooltip title="批量删除" placement="left"><Button size="small" danger icon={<DeleteOutlined/>} style={{height:40,padding:0}}>删除</Button></Tooltip>
        </Popconfirm>
      </div>

      {/* 添加/编辑弹窗 */}
      <Modal title={editingRule?'编辑规则':'添加规则'} open={addModalOpen} onCancel={()=>{setAddModalOpen(false);setEditingRule(null);setAdvOpen(false)}} footer={null} destroyOnClose width={520}>
        <Form form={form} layout="vertical" onFinish={handleSubmit} initialValues={{protocol:'tcp+udp',rate:1}}>
          <Form.Item name="name" label="名称" rules={[{required:true}]}>
            <Input placeholder="规则名称" />
          </Form.Item>
          <Form.Item name="node_id" label="入口" rules={[{required:true,message:'请选择入口节点'}]}>
            <Select placeholder="选择入口节点">
              {deviceGroups.map(g => (
                <Select.Option key={g.id} value={g.id}>
                  {g.name} <Tag style={{fontSize:10}}>×{g.rate||1}</Tag>
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="listen_port" label="监听端口">
            <InputNumber min={1} max={65535} style={{width:'100%'}} placeholder="留空则随机" />
          </Form.Item>
          <Form.Item name="destinations" label="目标地址" rules={[{required:true,message:'请输入目标地址'}]}
            extra="一行一个，格式 IP:端口">
            <Input.TextArea rows={3} placeholder="1.2.3.4:5678" />
          </Form.Item>
          <Form.Item name="target_addr" hidden><Input /></Form.Item>
          <Form.Item name="target_port" hidden><InputNumber /></Form.Item>
          <Collapse ghost activeKey={advOpen?['1']:[]} onChange={()=>setAdvOpen(!advOpen)}>
            <Collapse.Panel key="1" header="高级选项">
              <Form.Item name="protocol" label="协议">
                <Select><Select.Option value="tcp">TCP</Select.Option><Select.Option value="udp">UDP</Select.Option><Select.Option value="tcp+udp">TCP+UDP</Select.Option></Select>
              </Form.Item>
              <Form.Item name="rate" label="计费倍率"><InputNumber disabled min={0} step={0.01} style={{width:'100%'}} /></Form.Item>
            </Collapse.Panel>
          </Collapse>
          <Form.Item style={{marginTop:16}}>
            <Button type="primary" htmlType="submit" loading={submitting} block>{editingRule?'保存修改':'确 定'}</Button>
          </Form.Item>
        </Form>
      </Modal>

      {/* 统计数据弹窗 */}
      <Modal title="统计数据" open={statsOpen} onCancel={()=>setStatsOpen(false)} footer={null} destroyOnClose>
        <Row gutter={16}>
          <Col span={8}><Card><Statistic title="规则总数" value={stats.total} /></Card></Col>
          <Col span={8}><Card><Statistic title="运行中" value={stats.online} suffix={`/ ${stats.total}`} /></Card></Col>
          <Col span={8}><Card><Statistic title="总流量" value={totalGB} /></Card></Col>
        </Row>
      </Modal>

      {/* 管理分组弹窗 */}
      <Modal title="管理分组" open={catMgrOpen} onCancel={()=>setCatMgrOpen(false)} footer={null} destroyOnClose width={400}>
        {categories.map(c => (
          <Row key={c.name} align="middle" style={{marginBottom:8}}>
            <Col flex="1"><Tag color="blue">{c.name}</Tag></Col>
            <Col>
              {c.name !== '全部' && (
                <Popconfirm title={`删除分组"${c.name}"？规则将移至"全部"`} onConfirm={async()=>{
                  try { await deleteCategory(c.id); fetchData() } catch { message.error('删除失败') }
                }}>
                  <Button size="small" danger icon={<DeleteOutlined/>} />
                </Popconfirm>
              )}
            </Col>
          </Row>
        ))}
        <Divider style={{margin:'12px 0'}} />
        <Form form={catForm} layout="inline" onFinish={async(values)=>{
          try { await createCategory(values); catForm.resetFields(); fetchData() } catch { message.error('创建失败') }
        }}>
          <Form.Item name="name" rules={[{required:true}]} style={{flex:1}}><Input placeholder="新分组名称" /></Form.Item>
          <Form.Item><Button type="primary" htmlType="submit">创建</Button></Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
