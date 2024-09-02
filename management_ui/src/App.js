import React, { useState, useEffect } from 'react';
import { Layout, Table, Button, Select, message, Modal, Form, Input, InputNumber, Switch, Row, Col } from 'antd';
import { SettingOutlined, PlusOutlined } from '@ant-design/icons';
import { IntlProvider, FormattedMessage, useIntl } from 'react-intl';
import { Route, Routes, useNavigate } from 'react-router-dom';
import enUS from './locales/en-US.json';
import zhCN from './locales/zh-CN.json';
import 'antd/dist/reset.css';
import './App.css';
import Settings from './Settings';

const { Header, Content, Footer } = Layout;
const { Option } = Select;

const messages = {
  'en-US': enUS,
  'zh-CN': zhCN,
};

function App() {
  const [locale, setLocale] = useState(localStorage.getItem('locale') || 'en-US');
  const [consumers, setConsumers] = useState([]);
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [form] = Form.useForm();
  const navigate = useNavigate();

  const hideSettingsButton = window.location.pathname === '/settings';

  useEffect(() => {
    fetchConsumers();
  }, []);

  const fetchConsumers = async () => {
    try {
      const response = await fetch('http://localhost:1981/consumers');
      const data = await response.json();
      if (Array.isArray(data)) {
        setConsumers(data);
      } else {
        message.error('Invalid data format');
      }
    } catch (error) {
      message.error('Failed to fetch consumers');
    }
  };

  const handleAddConsumer = async (newConsumer) => {
    try {
      const response = await fetch('http://localhost:1981/consumers', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(newConsumer),
      });
      if (response.ok) {
        message.success('Consumer added successfully');
        fetchConsumers(); // Refresh the consumer list
      } else {
        message.error('Failed to add consumer');
      }
    } catch (error) {
      message.error('Failed to add consumer');
    }
  };

  const handleEditConsumer = async (updatedConsumer) => {
    try {
      const response = await fetch(`http://localhost:1981/consumers/${updatedConsumer.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(updatedConsumer),
      });
      if (response.ok) {
        message.success('Consumer updated successfully');
        fetchConsumers(); // Refresh the consumer list
      } else {
        message.error('Failed to update consumer');
      }
    } catch (error) {
      message.error('Failed to update consumer');
    }
  };

  const handleDeleteConsumer = async (consumerId) => {
    try {
      const response = await fetch(`http://localhost:1981/consumers/${consumerId}`, {
        method: 'DELETE',
      });
      if (response.ok) {
        message.success('Consumer deleted successfully');
        fetchConsumers(); // Refresh the consumer list
      } else {
        message.error('Failed to delete consumer');
      }
    } catch (error) {
      message.error('Failed to delete consumer');
    }
  };

  const columns = [
    {
      title: <FormattedMessage id="table.name" />,
      dataIndex: 'name',
      key: 'name',
      render: (text) => <FormattedMessage id={`consumer.${text}`} defaultMessage={text} />,
    },
    {
      title: <FormattedMessage id="table.status" />,
      dataIndex: 'status',
      key: 'status',
      render: (text) => <FormattedMessage id={`status.${text}`} defaultMessage={text} />,
    },
    {
      title: <FormattedMessage id="table.queueName" />,
      dataIndex: 'queue_name',
      key: 'queue_name',
    },
    {
      title: <FormattedMessage id="table.exchangeName" />,
      dataIndex: 'exchange_name',
      key: 'exchange_name',
    },
    {
      title: <FormattedMessage id="table.routingKey" />,
      dataIndex: 'routing_key',
      key: 'routing_key',
    },
    {
      title: <FormattedMessage id="table.deathQueueName" />,
      dataIndex: 'death_queue_name',
      key: 'death_queue_name',
    },
    {
      title: <FormattedMessage id="table.deathQueueBindExchange" />,
      dataIndex: 'death_queue_bind_exchange',
      key: 'death_queue_bind_exchange',
    },
    {
      title: <FormattedMessage id="table.deathQueueBindRoutingKey" />,
      dataIndex: 'death_queue_bind_routing_key',
      key: 'death_queue_bind_routing_key',
    },
    {
      title: <FormattedMessage id="table.deathQueueTTL" />,
      dataIndex: 'death_queue_ttl',
      key: 'death_queue_ttl',
    },
    {
      title: <FormattedMessage id="table.callback" />,
      dataIndex: 'callback',
      key: 'callback',
    },
    {
      title: <FormattedMessage id="table.retryMode" />,
      dataIndex: 'retry_mode',
      key: 'retry_mode',
    },
    {
      title: <FormattedMessage id="table.queueCount" />,
      dataIndex: 'queue_count',
      key: 'queue_count',
    },
    {
      title: <FormattedMessage id="table.actions" />,
      key: 'actions',
      render: (_, record) => (
        <>
          <Button type="link" onClick={() => handleEdit(record)}>
            <FormattedMessage id="button.edit" defaultMessage="Edit" />
          </Button>
          <Button type="link" onClick={() => handleDeleteConsumer(record.id)}>
            <FormattedMessage id="button.delete" defaultMessage="Delete" />
          </Button>
        </>
      ),
    },
  ];

  const handleEdit = (consumer) => {
    // Implement edit functionality
    console.log('Edit consumer:', consumer);
  };

  const showModal = () => {
    setIsModalVisible(true);
  };

  const handleCancel = () => {
    setIsModalVisible(false);
    form.resetFields();
  };

  const onFinish = (values) => {
    // Convert boolean to string for status
    values.status = values.status ? 'running' : 'stopped';
    handleAddConsumer(values);
    setIsModalVisible(false);
    form.resetFields();
  };

  const handleLocaleChange = (value) => {
    setLocale(value);
    localStorage.setItem('locale', value);
  };

  return (
    <IntlProvider messages={messages[locale]} locale={locale}>
      <Layout className="layout">
        <Header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div className="logo">
            <FormattedMessage id="app.title" />
          </div>
          <div>
            <Select defaultValue={locale} style={{ width: 120, marginRight: 16 }} onChange={handleLocaleChange}>
              <Option value="en-US">English</Option>
              <Option value="zh-CN">中文</Option>
            </Select>
            {!hideSettingsButton && (
              <Button icon={<SettingOutlined />} onClick={() => window.open('/settings', '_blank')}>
                <FormattedMessage id="app.settings" />
              </Button>
            )}
          </div>
        </Header>
        <Content style={{ padding: '0 50px' }}>
          <Routes>
            <Route path="/" element={
              <div className="site-layout-content">
                <div style={{ marginBottom: 16, textAlign: 'right' }}>
                  <Button type="primary" icon={<PlusOutlined />} onClick={showModal}>
                    <FormattedMessage id="app.addConsumer" />
                  </Button>
                </div>
                <Table columns={columns} dataSource={consumers} rowKey="id" />
              </div>
            } />
            <Route path="/settings" element={<Settings />} />
          </Routes>
        </Content>
        <Footer style={{ textAlign: 'center' }}>RabbitMQ Consumers Hub ©2024</Footer>
      </Layout>

      <Modal
        title={<FormattedMessage id="modal.addConsumer" />}
        open={isModalVisible}
        onCancel={handleCancel}
        footer={null}
        width={800}
        styles={{ body: { maxHeight: '70vh', overflow: 'auto' } }}
      >
        <ConsumerForm form={form} onFinish={onFinish} />
      </Modal>
    </IntlProvider>
  );
}

const ConsumerForm = ({ form, onFinish }) => {
  const intl = useIntl();

  return (
    <Form form={form} onFinish={onFinish} layout="vertical">
      <Row gutter={16}>
        <Col span={12}>
          <Form.Item
            name="name"
            label={<FormattedMessage id="table.name" />}
            rules={[{ required: true, message: intl.formatMessage({ id: 'validation.name' }) }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="status"
            label={<FormattedMessage id="table.status" />}
            valuePropName="checked"
          >
            <Switch 
              defaultChecked 
              checkedChildren={<FormattedMessage id="status.running" />} 
              unCheckedChildren={<FormattedMessage id="status.stopped" />} 
            />
          </Form.Item>
          <Form.Item
            name="queue_name"
            label={<FormattedMessage id="table.queueName" />}
            rules={[{ required: true, message: intl.formatMessage({ id: 'validation.queueName' }) }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="exchange_name"
            label={<FormattedMessage id="table.exchangeName" />}
            rules={[{ required: true, message: intl.formatMessage({ id: 'validation.exchangeName' }) }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="routing_key"
            label={<FormattedMessage id="table.routingKey" />}
            rules={[{ required: true, message: intl.formatMessage({ id: 'validation.routingKey' }) }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="callback"
            label={<FormattedMessage id="table.callback" />}
            rules={[{ required: true, message: intl.formatMessage({ id: 'validation.callback' }) }]}
          >
            <Input />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            name="death_queue_name"
            label={<FormattedMessage id="table.deathQueueName" />}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="death_queue_bind_exchange"
            label={<FormattedMessage id="table.deathQueueBindExchange" />}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="death_queue_bind_routing_key"
            label={<FormattedMessage id="table.deathQueueBindRoutingKey" />}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="death_queue_ttl"
            label={<FormattedMessage id="table.deathQueueTTL" />}
          >
            <Row gutter={16}>
              <Col span={8}>
                <InputNumber min={0} addonAfter={<FormattedMessage id="placeholder.hours" />} />
              </Col>
              <Col span={8}>
                <InputNumber min={0} max={59} addonAfter={<FormattedMessage id="placeholder.minutes" />} />
              </Col>
              <Col span={8}>
                <InputNumber min={0} max={59} addonAfter={<FormattedMessage id="placeholder.seconds" />} />
              </Col>
            </Row>
          </Form.Item>
          <Form.Item
            name="retry_mode"
            label={<FormattedMessage id="table.retryMode" />}
          >
            <Row gutter={16}>
              <Col span={8}>
                <InputNumber min={0} addonAfter={<FormattedMessage id="placeholder.hours" />} />
              </Col>
              <Col span={8}>
                <InputNumber min={0} max={59} addonAfter={<FormattedMessage id="placeholder.minutes" />} />
              </Col>
              <Col span={8}>
                <InputNumber min={0} max={59} addonAfter={<FormattedMessage id="placeholder.seconds" />} />
              </Col>
            </Row>
          </Form.Item>
          <Form.Item
            name="queue_count"
            label={<FormattedMessage id="table.queueCount" />}
          >
            <InputNumber min={1} defaultValue={1} />
          </Form.Item>
        </Col>
      </Row>
      <Form.Item style={{ textAlign: 'center' }}>
        <Button type="primary" htmlType="submit">
          <FormattedMessage id="form.submit" />
        </Button>
      </Form.Item>
    </Form>
  );
};

export default App;