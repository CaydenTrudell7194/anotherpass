import React, { useEffect, useState } from 'react'
import { Table, Button, Modal, Form, InputNumber, Space, Card, message, Popconfirm } from 'antd'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons'
import { adminListRedeemCodes, adminCreateRedeemCodes, adminDeleteRedeemCode } from '../../api'

interface RedeemCode {
  id: number
  code: string
  amount_cents: number
  used_count: number
  max_uses: number
  created_at: string
}

const RedeemCodes: React.FC = () => {
  const [data, setData] = useState<RedeemCode[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [form] = Form.useForm()
  const [creating, setCreating] = useState(false)

  const fetchData = async () => {
    setLoading(true)
    try {
      const res = await adminListRedeemCodes()
      setData(res.data)
    } catch {
      message.error('获取兑换码列表失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  const handleCreate = async () => {
    const values = await form.validateFields()
    setCreating(true)
    try {
      await adminCreateRedeemCodes(values)
      message.success('生成成功')
      setModalOpen(false)
      form.resetFields()
      fetchData()
    } catch {
      message.error('生成失败')
    } finally {
      setCreating(false)
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await adminDeleteRedeemCode(id)
      message.success('删除成功')
      fetchData()
    } catch {
      message.error('删除失败')
    }
  }

  const columns = [
    { title: '兑换码', dataIndex: 'code', key: 'code', ellipsis: true },
    {
      title: '金额', dataIndex: 'amount_cents', key: 'amount_cents',
      render: (v: number) => '¥' + (v / 100).toFixed(2),
    },
    {
      title: '使用次数', key: 'usage',
      render: (_: any, r: RedeemCode) => `${r.used_count} / ${r.max_uses}`,
    },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at' },
    {
      title: '操作', key: 'action',
      render: (_: any, r: RedeemCode) => (
        <Popconfirm title="确定删除？" onConfirm={() => handleDelete(r.id)}>
          <Button type="link" danger icon={<DeleteOutlined />}>删除</Button>
        </Popconfirm>
      ),
    },
  ]

  return (
    <div>
      <Card title="兑换码管理" extra={<Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>批量生成</Button>}>
        <Table rowKey="id" dataSource={data} columns={columns} loading={loading} pagination={{ pageSize: 20 }} />
      </Card>

      <Modal
        title="批量生成兑换码"
        open={modalOpen}
        onOk={handleCreate}
        onCancel={() => setModalOpen(false)}
        confirmLoading={creating}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="count" label="生成数量" rules={[{ required: true, message: '请输入数量' }]}>
            <InputNumber min={1} max={100} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="amount_cents" label="金额（分）" rules={[{ required: true, message: '请输入金额' }]}>
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="max_uses" label="最大使用次数" rules={[{ required: true, message: '请输入最大使用次数' }]}>
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default RedeemCodes
