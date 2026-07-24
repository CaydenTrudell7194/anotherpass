import React, { useEffect, useState } from 'react'
import { Table, Card, message } from 'antd'
import { adminListAffiliates } from '../../api'

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

  useEffect(() => {
    setLoading(true)
    adminListAffiliates()
      .then(res => setData(res.data))
      .catch(() => message.error('获取推广列表失败'))
      .finally(() => setLoading(false))
  }, [])

  const columns = [
    { title: '用户 ID', dataIndex: 'user_id', key: 'user_id' },
    { title: '推广码', dataIndex: 'code', key: 'code' },
    {
      title: '佣金比例', dataIndex: 'commission_rate', key: 'commission_rate',
      render: (v: number) => (v * 100).toFixed(1) + '%',
    },
    {
      title: '累计收益', dataIndex: 'total_earned_cents', key: 'total_earned_cents',
      render: (v: number) => '¥' + (v / 100).toFixed(2),
    },
  ]

  return (
    <Card title="推广管理">
      <Table rowKey="id" dataSource={data} columns={columns} loading={loading} pagination={{ pageSize: 20 }} />
    </Card>
  )
}

export default Affiliates
