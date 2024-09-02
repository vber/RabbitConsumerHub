import React, { useState, useEffect } from 'react';
import { Form, Input, Button, message } from 'antd';
import { FormattedMessage, useIntl } from 'react-intl';

const Settings = () => {
  const [form] = Form.useForm();
  const intl = useIntl();

  useEffect(() => {
    fetchRabbitMQConfig();
  }, []);

  const fetchRabbitMQConfig = async () => {
    try {
      const response = await fetch('http://localhost:1981/rabbitmq-config');
      const data = await response.json();
      console.log("data vhost",data.VHOST, data)
      form.setFieldsValue({
        host: data.HOSTNAME,
        port: data.PORT,
        user: data.USERNAME,
        password: data.PASSWORD,
        heartbeat: data.HEARTBEAT,
        frameMax: data.FRAMEMAX,
        vhost: data.VHOST,
      });
    } catch (error) {
      message.error(intl.formatMessage({ id: 'settings.fetchError' }));
    }
  };

  const onFinish = async (values) => {
    console.log(JSON.stringify({
        host: values.host,
        port: values.port,
        user: values.user,
        password: values.password,
        vhost: values.vhost,
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
          vhost: values.vhost,
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
    <Form form={form} onFinish={onFinish} layout="vertical">
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
        name="vhost"
        label={<FormattedMessage id="settings.vhost" />}
        rules={[{ required: true, message: intl.formatMessage({ id: 'validation.vhost' }) }]}
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