import React, { useEffect, useCallback } from 'react';
import { Form, Input, Button, message } from 'antd';
import { FormattedMessage, useIntl } from 'react-intl';

const Settings = () => {
  const [form] = Form.useForm();
  const intl = useIntl();

  const fetchRabbitMQConfig = useCallback(async () => {
    try {
      const response = await fetch('http://localhost:1981/rabbitmq-config');
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
      const response = await fetch('http://localhost:1981/rabbitmq-config', {
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
        <Button type="primary" htmlType="submit">
          <FormattedMessage id="settings.save" />
        </Button>
      </Form.Item>
    </Form>
  );
};

export default Settings;