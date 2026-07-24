import { useEffect, useState } from 'react'
import { Alert, Button, Card, Col, Form, Input, InputNumber, Radio, Row, Space, Switch, Typography, message } from 'antd'
import { SaveOutlined, SettingOutlined } from '@ant-design/icons'
import { errorMessage, getAdminSettings, listUserGroups, updateAdminSettings } from '../../api'
import { useSite } from '../../site'

const { Paragraph, Text } = Typography

export default function Settings() {
  const [form] = Form.useForm()
  const [groups, setGroups] = useState<any[]>([])
  const [saving, setSaving] = useState(false)
  const { refresh } = useSite()
  useEffect(() => {
    Promise.all([getAdminSettings(), listUserGroups()]).then(([settings, userGroups]) => {
      form.setFieldsValue(settings.data)
      setGroups(userGroups.data || [])
    }).catch(err => message.error(errorMessage(err, '读取站点设置失败')))
  }, [])
  const save = async (values: any) => {
    setSaving(true)
    try {
      await updateAdminSettings(values)
      await refresh()
      message.success('站点设置已保存')
    } catch (err) {
      message.error(errorMessage(err, '保存失败'))
    } finally { setSaving(false) }
  }
  return (
    <div style={{ maxWidth: 1050, margin: '0 auto' }}>
      <Alert type="info" showIcon style={{ marginBottom: 16 }} message="入口直出模式" description="本面板管理入口直出、套餐和人工订单；暂不包含支付、商城、邀请返利或出口故障转移。多台节点加入同一设备组时共用该组规则。" />
      <Form form={form} layout="vertical" onFinish={save}>
        <Card title={<><SettingOutlined /> 基本站点</>} style={{ marginBottom: 16 }}>
          <Row gutter={16}>
            <Col xs={24} md={12}><Form.Item name="site_name" label="站点名称" rules={[{ required: true }, { max: 64 }]}><Input /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item name="site_subtitle" label="站点副标题" rules={[{ required: true }, { max: 128 }]}><Input /></Form.Item></Col>
          </Row>
          <Form.Item name="site_notice" label="站点公告" rules={[{ max: 4096 }]}><Input.TextArea rows={5} placeholder="显示在用户主页，按纯文本安全展示" /></Form.Item>
        </Card>
        <Card title="注册与默认配额" style={{ marginBottom: 16 }}>
          <Form.Item name="allow_register" label="允许公开注册" valuePropName="checked"><Switch /></Form.Item>
          <Row gutter={16}>
            <Col xs={24} md={6}><Form.Item name="register_user_group_id" label="默认用户组" rules={[{ required: true }]}>
              <Radio.Group>{groups.map(g => <Radio key={g.id} value={g.id}>{g.name}</Radio>)}</Radio.Group>
            </Form.Item></Col>
            <Col xs={24} md={6}><Form.Item name="register_rule_limit" label="规则数量限制"><InputNumber min={0} precision={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item name="register_expire_days" label="默认有效期（天）"><InputNumber min={1} max={3650} precision={0} style={{ width: '100%' }} /></Form.Item></Col>
          </Row>
        </Card>
        <Card title="主题与背景" style={{ marginBottom: 16 }}>
          <Form.Item name="theme_policy" label="主题策略"><Radio.Group><Radio value="classic">经典主题</Radio><Radio value="transparent">透明背景主题</Radio></Radio.Group></Form.Item>
          <Form.Item name="background_url" label="横屏背景图 URL"><Input placeholder="https://..." /></Form.Item>
          <Form.Item name="mobile_background_url" label="竖屏背景图 URL"><Input placeholder="https://..." /></Form.Item>
        </Card>
        <Card title="节点状态" style={{ marginBottom: 16 }}>
          <Paragraph type="secondary">影响节点在线状态展示，不改变节点客户端的规则同步周期。</Paragraph>
          <Space wrap size="large">
            <Form.Item name="offline_node_seconds" label="无心跳判定离线（秒）"><InputNumber min={20} max={3600} precision={0} /></Form.Item>
            <Form.Item name="offline_node_retention_hours" label="离线节点保留展示（小时）"><InputNumber min={1} max={8760} precision={0} /></Form.Item>
          </Space>
          <div><Text type="secondary">设备组倍率、允许用户组、连接地址、排序和备注请在“设备组管理”中设置。</Text></div>
        </Card>
        <Card title="Telegram 通知" style={{ marginBottom: 16 }}>
          <Alert type="info" showIcon style={{marginBottom:16}} message="Bot Token 通过服务器环境变量 TELEGRAM_BOT_TOKEN 配置，不会保存到数据库或发送到浏览器。" />
          <Form.Item name="telegram_enabled" label="启用 Telegram 通知" valuePropName="checked"><Switch /></Form.Item>
          <Form.Item name="telegram_chat_id" label="通知 Chat ID" rules={[{max:64}]}><Input placeholder="例如：-1001234567890" /></Form.Item>
          <Form.Item shouldUpdate noStyle>{({getFieldValue}) => <Text type={getFieldValue('telegram_bot_configured')?'success':'warning'}>{getFieldValue('telegram_bot_configured')?'服务器已配置 Bot Token':'服务器尚未配置 Bot Token'}</Text>}</Form.Item>
        </Card>
        <Button type="primary" htmlType="submit" icon={<SaveOutlined />} loading={saving} size="large">保存全部设置</Button>
      </Form>
    </div>
  )
}
