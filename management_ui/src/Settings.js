import React, { useEffect, useCallback, useState } from 'react';
import { Form, Input, Button, message, Spin } from 'antd';
import { FormattedMessage, useIntl } from 'react-intl';

const Settings = () => {
  const [form] = Form.useForm();
  const intl = useIntl();
  const [isTestingConnection, setIsTestingConnection] = useState(false);

  const fetchRabbitMQConfig = useCallback(async () => {
    try {
      const response = await fetch(`${process.env.REACT_APP_API_BASE_URL}/rabbitmq-config`);
      const data = await response.json();
      form.setFieldsValue({
        host: data.HOSTNAME,
        port: data.PORT,
        user: data.USERNAME,
        password: data.PASSWORD,
        // Removed vhost from here
      });
    } catch (error) {
      message.error(intl.formatMessage({ id: 'settings.fetchError' }));
    }
  }, [form, intl]);

  useEffect(() => {
    fetchRabbitMQConfig();
  }, [fetchRabbitMQConfig]);

  const onFinish = async (values) => {
    console.log(JSON.stringify({
      host: values.host,
      port: values.port,
      user: values.user,
      password: values.password,
      // Removed vhost from here
    }))

    try {
      const response = await fetch(`${process.env.REACT_APP_API_BASE_URL}/rabbitmq-config`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          host: values.host,
          port: values.port,
          user: values.user,
          password: values.password,
          // Removed vhost from here
        }),
      });
      if (response.ok) {
        message.success(intl.formatMessage({ id: 'settings.updateSuccess' }));
      } else {
        message.error(intl.formatMessage({ id: 'settings.updateError' }));
      }
    } catch (error) {
      message.error(intl.formatMessage({ id: 'settings.updateError' }));
    }
  };

  const testConnection = async () => {
    const values = form.getFieldsValue();
    setIsTestingConnection(true);
    try {
      const response = await fetch(`${process.env.REACT_APP_API_BASE_URL}/test-rabbitmq-connection`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          host: values.host,
          port: values.port,
          user: values.user,
          password: values.password,
        }),
      });
      if (response.ok) {
        message.success(intl.formatMessage({ id: 'settings.testConnectionSuccess' }));
      } else {
        message.error(intl.formatMessage({ id: 'settings.testConnectionError' }));
      }
    } catch (error) {
      message.error(intl.formatMessage({ id: 'settings.testConnectionError' }));
    } finally {
      setIsTestingConnection(false);
    }
  };

  return (
    <Form form={form} onFinish={onFinish} layout="vertical" style={{ marginTop: '16px' }}>
      <Form.Item
        name="host"
        label={<FormattedMessage id="settings.host" />}
        rules={[{ required: true, message: intl.formatMessage({ id: 'validation.host' }) }]}
      >
        <Input />
      </Form.Item>
      <Form.Item
        name="port"
        label={<FormattedMessage id="settings.port" />}
        rules={[{ required: true, message: intl.formatMessage({ id: 'validation.port' }) }]}
      >
        <Input />
      </Form.Item>
      <Form.Item
        name="user"
        label={<FormattedMessage id="settings.user" />}
        rules={[{ required: true, message: intl.formatMessage({ id: 'validation.user' }) }]}
      >
        <Input />
      </Form.Item>
      <Form.Item
        name="password"
        label={<FormattedMessage id="settings.password" />}
        rules={[{ required: true, message: intl.formatMessage({ id: 'validation.password' }) }]}
      >
        <Input.Password />
      </Form.Item>
      <Form.Item>
        <Button type="primary" htmlType="submit" style={{ marginRight: '8px' }}>
          <FormattedMessage id="settings.save" />
        </Button>
        <Button 
          onClick={testConnection} 
          type="default"
          style={{ backgroundColor: 'red', color: 'white' }}
          icon={isTestingConnection ? <Spin size="small" /> : null}
          disabled={isTestingConnection}
        >
          <FormattedMessage id="settings.testConnection" />
        </Button>
      </Form.Item>
    </Form>
  );
};

export default Settings;