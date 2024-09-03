import React, { useState, useEffect } from 'react';
import { Layout, Table, Button, Select, message, Modal, Form, Input, InputNumber, Switch, Row, Col, Typography } from 'antd';
import { SettingOutlined, PlusOutlined } from '@ant-design/icons';
import { IntlProvider, FormattedMessage, useIntl } from 'react-intl';
import { Route, Routes } from 'react-router-dom';
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

  const handleLocaleChange = (value) => {
    setLocale(value);
    localStorage.setItem('locale', value);
  };

  return (
    <IntlProvider messages={messages[locale]} locale={locale}>
      <AppContent locale={locale} handleLocaleChange={handleLocaleChange} />
    </IntlProvider>
  );
}

function AppContent({ locale, handleLocaleChange }) {
  const [consumers, setConsumers] = useState([]);
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [editingConsumer, setEditingConsumer] = useState(null);
  const [isAdding, setIsAdding] = useState(false);
  const [form] = Form.useForm();
  const intl = useIntl();

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
        message.error(intl.formatMessage({ id: 'error.invalidDataFormat' }));
      }
    } catch (error) {
      message.error(intl.formatMessage({ id: 'error.failedToFetchConsumers' }));
    }
  };

  const handleAddConsumer = async (newConsumer) => {
    try {
      // Function to recursively sanitize an object
      const sanitizeObject = (obj) => {
        if (typeof obj !== 'object' || obj === null) {
          return obj === undefined ? '' : obj;
        }
        return Object.fromEntries(
          Object.entries(obj).map(([key, value]) => [key, sanitizeObject(value)])
        );
      };

      // Sanitize the entire newConsumer object
      const sanitizedConsumer = sanitizeObject(newConsumer);

      const response = await fetch('http://localhost:1981/consumers', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(sanitizedConsumer),
      });
      if (response.ok) {
        message.success(intl.formatMessage({ id: 'success.consumerAdded' }));
        fetchConsumers();
        return true;
      } else {
        message.error(intl.formatMessage({ id: 'error.failedToAddConsumer' }));
        return false;
      }
    } catch (error) {
      message.error(intl.formatMessage({ id: 'error.failedToAddConsumer' }));
      return false;
    }
  };

  const handleDeleteConsumer = async (consumerId) => {
    try {
      const response = await fetch(`http://localhost:1981/consumers/${consumerId}`, {
        method: 'DELETE',
      });
      if (response.ok) {
        message.success(intl.formatMessage({ id: 'success.consumerDeleted' }));
        fetchConsumers();
      } else {
        message.error(intl.formatMessage({ id: 'error.failedToDeleteConsumer' }));
      }
    } catch (error) {
      message.error(intl.formatMessage({ id: 'error.failedToDeleteConsumer' }));
    }
  };

  const columns = [
    {
      title: <FormattedMessage id="table.name" />,
      dataIndex: 'name',
      key: 'name',
      fixed: 'left',
      render: (text) => <FormattedMessage id={`consumer.${text}`} defaultMessage={text} />,
    },
    {
      title: <FormattedMessage id="table.status" />,
      dataIndex: 'status',
      key: 'status',
      fixed: 'left',
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
      dataIndex: ['death_queue', 'x_death_queue_name'],
      key: 'death_queue_name',
    },
    {
      title: <FormattedMessage id="table.deathQueueBindExchange" />,
      dataIndex: ['death_queue', 'bind_exchange'],
      key: 'death_queue_bind_exchange',
    },
    {
      title: <FormattedMessage id="table.deathQueueBindRoutingKey" />,
      dataIndex: ['death_queue', 'bind_routing_key'],
      key: 'death_queue_bind_routing_key',
    },
    {
      title: <FormattedMessage id="table.deathQueueTTL" />,
      dataIndex: ['death_queue', 'x_message_ttl'],
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
      fixed: 'right',
      render: (_, record) => (
        <>
          <Button type="link" onClick={() => showModal(record)}>
            <FormattedMessage id="button.edit" defaultMessage="Edit" />
          </Button>
          <Button type="link" onClick={() => handleDeleteConsumer(record.id)}>
            <FormattedMessage id="button.delete" defaultMessage="Delete" />
          </Button>
        </>
      ),
    },
  ];

  const showModal = (consumer = null) => {
    if (consumer) {
      setEditingConsumer(consumer);
      setIsAdding(false);
      form.setFieldsValue({
        ...consumer,
        status: consumer.status === 'running',
      });
    } else {
      setEditingConsumer(null);
      setIsAdding(true);
      form.resetFields();
    }
    setIsModalVisible(true);
  };

  const handleCancel = () => {
    setIsModalVisible(false);
    setEditingConsumer(null);
    setIsAdding(false);
    form.resetFields();
  };

  const onFinish = async (values) => {
    values.status = values.status ? 'running' : 'stopped';
    let success = false;

    if (editingConsumer) {
      success = await handleEditConsumer(editingConsumer.id, values);
    } else {
      success = await handleAddConsumer(values);
    }

    if (success) {
      setIsModalVisible(false);
      setEditingConsumer(null);
      setIsAdding(false);
      form.resetFields();
    }
    // If not successful, the modal will remain open
  };

  const handleEditConsumer = async (id, updatedConsumer) => {
    try {
      console.log(updatedConsumer)
      // Replace undefined values with empty strings
      const sanitizedConsumer = Object.fromEntries(
        Object.entries(updatedConsumer).map(([key, value]) => [key, value === undefined ? '' : value])
      );

      console.log(JSON.stringify(sanitizedConsumer));
      const response = await fetch(`http://localhost:1981/consumers/${id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(sanitizedConsumer),
      });
      if (response.ok) {
        message.success(intl.formatMessage({ id: 'success.consumerUpdated' }));
        fetchConsumers();
        return true;
      } else {
        message.error(intl.formatMessage({ id: 'error.failedToUpdateConsumer' }));
        return false;
      }
    } catch (error) {
      message.error(intl.formatMessage({ id: 'error.failedToUpdateConsumer' }));
      return false;
    }
  };

  return (
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
                <Button type="primary" icon={<PlusOutlined />} onClick={() => showModal()}>
                  <FormattedMessage id="app.addConsumer" />
                </Button>
              </div>
              <Table columns={columns} dataSource={consumers} rowKey="id" scroll={{ x: 'max-content' }} />
            </div>
          } />
          <Route path="/settings" element={<Settings />} />
        </Routes>
      </Content>
      <Footer style={{ textAlign: 'center' }}>RabbitMQ Consumers Hub ©2024</Footer>

      <Modal
        key={isAdding ? 'add' : 'edit'}
        title={
          <Typography.Title level={4}>
            <FormattedMessage id={isAdding ? "modal.addConsumer" : "modal.editConsumer"} />
          </Typography.Title>
        }
        open={isModalVisible}
        onCancel={handleCancel}
        footer={null}
        width={800}
        styles={{ body: { maxHeight: '70vh', overflow: 'auto' } }}
      >
        <ConsumerForm form={form} onFinish={onFinish} editingConsumer={editingConsumer} />
      </Modal>
    </Layout>
  );
}

const ConsumerForm = ({ form, onFinish, editingConsumer }) => {
  const intl = useIntl();

  const formatTimeValue = (hours, minutes, seconds) => {
    let formatted = '';
    if (hours > 0) formatted += `${hours}h`;
    if (minutes > 0) formatted += `${minutes}m`;
    if (seconds > 0) formatted += `${seconds}s`;
    return formatted || '0s';
  };

  const parseTimeValue = (value) => {
    const hours = value.match(/(\d+)h/);
    const minutes = value.match(/(\d+)m/);
    const seconds = value.match(/(\d+)s/);
    return {
      hours: hours ? parseInt(hours[1]) : 0,
      minutes: minutes ? parseInt(minutes[1]) : 0,
      seconds: seconds ? parseInt(seconds[1]) : 0,
    };
  };

  useEffect(() => {
    if (editingConsumer) {
      const deathQueueTTL = parseTimeValue(editingConsumer.death_queue?.x_message_ttl || '');
      const retryMode = parseTimeValue(editingConsumer.retry_mode || '');
      
      form.setFieldsValue({
        ...editingConsumer,
        'death_queue.x_death_queue_name': editingConsumer.death_queue?.x_death_queue_name,
        'death_queue.bind_exchange': editingConsumer.death_queue?.bind_exchange,
        'death_queue.bind_routing_key': editingConsumer.death_queue?.bind_routing_key,
        death_queue_ttl_hours: deathQueueTTL.hours,
        death_queue_ttl_minutes: deathQueueTTL.minutes,
        death_queue_ttl_seconds: deathQueueTTL.seconds,
        retry_mode_hours: retryMode.hours,
        retry_mode_minutes: retryMode.minutes,
        retry_mode_seconds: retryMode.seconds,
      });
    } else {
      // Set default values for new consumers
      form.setFieldsValue({
        status: true,
        queue_count: 1  // Set default queue_count to 1
      });
    }
  }, [editingConsumer, form]);

  const onFormFinish = (values) => {
    const formattedValues = {
      ...values,
      death_queue: {
        ...values.death_queue,
        x_message_ttl: formatTimeValue(
          values.death_queue_ttl_hours,
          values.death_queue_ttl_minutes,
          values.death_queue_ttl_seconds
        ),
      },
      retry_mode: formatTimeValue(
        values.retry_mode_hours,
        values.retry_mode_minutes,
        values.retry_mode_seconds
      ),
    };

    // Remove individual time fields
    ['death_queue_ttl', 'retry_mode'].forEach(field => {
      delete formattedValues[`${field}_hours`];
      delete formattedValues[`${field}_minutes`];
      delete formattedValues[`${field}_seconds`];
    });
    
    onFinish(formattedValues);
  };

  const TimeInputs = ({ fieldName }) => (
    <Row gutter={16}>
      <Col span={8}>
        <Form.Item name={`${fieldName}_hours`} noStyle>
          <InputNumber min={0} addonAfter={<FormattedMessage id="placeholder.hours" />} />
        </Form.Item>
      </Col>
      <Col span={8}>
        <Form.Item name={`${fieldName}_minutes`} noStyle>
          <InputNumber min={0} max={59} addonAfter={<FormattedMessage id="placeholder.minutes" />} />
        </Form.Item>
      </Col>
      <Col span={8}>
        <Form.Item name={`${fieldName}_seconds`} noStyle>
          <InputNumber min={0} max={59} addonAfter={<FormattedMessage id="placeholder.seconds" />} />
        </Form.Item>
      </Col>
    </Row>
  );

  return (
    <Form form={form} onFinish={onFormFinish} layout="vertical" initialValues={{ status: true, queue_count: 1 }}>
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
            name={['death_queue', 'x_death_queue_name']}
            label={<FormattedMessage id="table.deathQueueName" />}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name={['death_queue', 'bind_exchange']}
            label={<FormattedMessage id="table.deathQueueBindExchange" />}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name={['death_queue', 'bind_routing_key']}
            label={<FormattedMessage id="table.deathQueueBindRoutingKey" />}
          >
            <Input />
          </Form.Item>
          <Form.Item
            label={<FormattedMessage id="table.deathQueueTTL" />}
          >
            <TimeInputs fieldName="death_queue_ttl" />
          </Form.Item>
          <Form.Item
            label={<FormattedMessage id="table.retryMode" />}
          >
            <TimeInputs fieldName="retry_mode" />
          </Form.Item>
          <Form.Item
            name="queue_count"
            label={<FormattedMessage id="table.queueCount" />}
          >
            <InputNumber min={1} />
          </Form.Item>
        </Col>
      </Row>
      <Form.Item style={{ textAlign: 'center' }}>
        <Button type="primary" htmlType="submit">
          <FormattedMessage id={editingConsumer ? "form.update" : "form.submit"} />
        </Button>
      </Form.Item>
    </Form>
  );
};

export default App;