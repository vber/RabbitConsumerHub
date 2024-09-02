import React, { useState, useEffect } from 'react';
import { Layout, Table, Button, Select, message } from 'antd';
import { SettingOutlined, PlusOutlined } from '@ant-design/icons';
import { IntlProvider, FormattedMessage } from 'react-intl';
import enUS from './locales/en-US.json';
import zhCN from './locales/zh-CN.json';
import 'antd/dist/reset.css';
import './App.css';

const { Header, Content, Footer } = Layout;
const { Option } = Select;

const messages = {
  'en-US': enUS,
  'zh-CN': zhCN,
};

function App() {
  const [locale, setLocale] = useState('en-US');
  const [consumers, setConsumers] = useState([]);

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

  return (
    <IntlProvider messages={messages[locale]} locale={locale}>
      <Layout className="layout">
        <Header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div className="logo">
            <FormattedMessage id="app.title" />
          </div>
          <div>
            <Select defaultValue={locale} style={{ width: 120, marginRight: 16 }} onChange={setLocale}>
              <Option value="en-US">English</Option>
              <Option value="zh-CN">中文</Option>
            </Select>
            <Button icon={<SettingOutlined />}>
              <FormattedMessage id="app.settings" />
            </Button>
          </div>
        </Header>
        <Content style={{ padding: '0 50px' }}>
          <div className="site-layout-content">
            <div style={{ marginBottom: 16, textAlign: 'right' }}>
              <Button type="primary" icon={<PlusOutlined />} onClick={handleAddConsumer}>
                <FormattedMessage id="app.addConsumer" />
              </Button>
            </div>
            <Table columns={columns} dataSource={consumers} rowKey="id" />
          </div>
        </Content>
        <Footer style={{ textAlign: 'center' }}>RabbitMQ Consumers Hub ©2024</Footer>
      </Layout>
    </IntlProvider>
  );
}

export default App;