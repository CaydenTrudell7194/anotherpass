import React, { useEffect, useState } from 'react'
import { Table, Card, Button, InputNumber, Modal, Space, message, Tooltip } from 'antd'
import { EditOutlined } from '@ant-design/icons'
import { adminListAffiliates, adminUpdateAffiliate } from '../../api'

interface AffiliateRecord {
  id: number
  user_id: number
  code: string
  commission_rate: number
  total_earned_cents: number
}

const Affiliates: React.FC = () => {
  const [data, setData] = useState<AffiliateRecord[]>([])
  const [loading, setLoading] = useState(false)
  const [editing, setEditing] = useState<AffiliateRecord | null>(null)
  const [rate, setRate] = useState(0)
  const [saving, setSaving] = useState(false)

  useEffect(() => { fetchData() }, [])

  const fetchData = () => {
    setLoading(true)
    adminListAffiliates()
      .then(res => setData(res.data))
      .catch(() => message.error('获取推广列表失败'))
      .finally(() => setLoading(false))
  }

  const saveRate = async () => {
    if (!editing) return
    setSaving(true)
    try {
      await adminUpdateAffiliate(editing.id, { commission_rate: rate / 100 })
      message.success('佣金比例已更新')
      setEditing(null)
      fetchData()
    } catch {
      message.error('更新失败')
    } finally {
      setSaving(false)
    }
  }

  const columns = [
    { title: '用户 ID', dataIndex: 'user_id', key: 'user_id' },
    { title: '推广码', dataIndex: 'code', key: 'code' },
    {
      title: '佣金比例', dataIndex: 'commission_rate', key: 'commission_rate',
      render: (v: number, r: AffiliateRecord) => (
        <Space>
          <span>{(v * 100).toFixed(1)}%</span>
          <Tooltip title="设置佣金比例">
            <Button type="link" size="small" icon={<EditOutlined />} onClick={() => { setEditing(r); setRate(v * 100) }} />
          </Tooltip>
        </Space>
      ),
    },
    {
      title: '累计收益', dataIndex: 'total_earned_cents', key: 'total_earned_cents',
      render: (v: number) => '¥' + (v / 100).toFixed(2),
    },
  ]

  return (
    <Card title="推广管理">
      <Table rowKey="id" dataSource={data} columns={columns} loading={loading} pagination={{ pageSize: 20 }} />
      <Modal title="设置佣金比例" open={!!editing} onOk={saveRate} onCancel={() => setEditing(null)} okText="保存" confirmLoading={saving}>
        <InputNumber min={0} max={100} precision={1} value={rate} onChange={v => setRate(v ?? 0)} style={{ width: '100%' }} addonAfter="%" />
      </Modal>
    </Card>
  )
}

export default Affiliates
