import React, { useEffect, useState } from 'react'
import { Card, Row, Col, Statistic, Spin, message, Typography, Input } from 'antd'
import { CopyOutlined, DollarOutlined, PercentageOutlined, GiftOutlined, UserOutlined } from '@ant-design/icons'
import { getAffiliateInfo } from '../../api'

const { Text } = Typography

interface AffiliateInfo {
  code: string
  total_earned_cents: number
  commission_rate: number
  referral_count: number
  invite_link?: string
}

const Affiliate: React.FC = () => {
  const [info, setInfo] = useState<AffiliateInfo | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    getAffiliateInfo()
      .then(res => setInfo(res.data))
      .catch(() => message.error('获取推广信息失败'))
      .finally(() => setLoading(false))
  }, [])

  const inviteUrl = info?.code ? `${window.location.origin}/register?ref=${info.code}` : ''

  const handleCopy = (text: string, label: string) => {
    navigator.clipboard.writeText(text)
    message.success(`已复制${label}`)
  }

  if (loading) return <Spin size="large" style={{ display: 'flex', justifyContent: 'center', marginTop: 120 }} />
  if (!info) return null

  return (
    <div style={{ maxWidth: 800, margin: '0 auto' }}>
      <Row gutter={[16, 16]}>
        <Col span={24}>
      <Card title={<><GiftOutlined /> 推广返利</>}>
            <div style={{ marginBottom: 16 }}>
              <Text strong>我的推广码：</Text>
              <Input.Search
                value={info.code}
                readOnly
                enterButton={<><CopyOutlined /> 复制</>}
                onSearch={() => handleCopy(info.code, '推广码')}
                style={{ width: 300, marginLeft: 12 }}
              />
            </div>
            <div style={{ marginBottom: 16 }}>
              <Text strong>推广链接：</Text>
              <Input.Search
                value={inviteUrl}
                readOnly
                enterButton={<><CopyOutlined /> 复制</>}
                onSearch={() => handleCopy(inviteUrl, '推广链接')}
                style={{ width: '100%', marginLeft: 12 }}
              />
              <div style={{ fontSize: 12, color: '#999', marginTop: 4, marginLeft: 12 }}>
                用户通过此链接注册将自动填写您的推广码
              </div>
            </div>
            <Row gutter={16}>
              <Col span={12}>
                <Card>
                  <Statistic title="累计收益" prefix="¥" value={(info.total_earned_cents / 100).toFixed(2)} suffix={<DollarOutlined />} />
                </Card>
              </Col>
              <Col span={12}>
                <Card>
                  <Statistic title="佣金比例" value={info.commission_rate * 100} suffix="%" prefix={<PercentageOutlined />} />
                </Card>
              </Col>
              <Col span={12}>
                <Card>
                  <Statistic title="推广人数" value={info.referral_count} prefix={<UserOutlined />} />
                </Card>
              </Col>
            </Row>
          </Card>
        </Col>
      </Row>
    </div>
  )
}

export default Affiliate
