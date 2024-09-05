import React, { useState, useEffect } from 'react';
import { Table, Button, message, Popconfirm, Dropdown, Menu } from 'antd';
import { DownOutlined } from '@ant-design/icons';
import { FormattedMessage, useIntl } from 'react-intl';

function FailedCallbacks() {
  const [failedCallbacks, setFailedCallbacks] = useState([]);
  const [selectedRowKeys, setSelectedRowKeys] = useState([]);
  const [loading, setLoading] = useState(false);
  const [retryLoading, setRetryLoading] = useState({});
  const [deleteLoading, setDeleteLoading] = useState({});
  const [bulkActionLoading, setBulkActionLoading] = useState(false);
  const intl = useIntl();

  useEffect(() => {
    fetchFailedCallbacks();
  }, []);

  const fetchFailedCallbacks = async () => {
    setLoading(true);
    try {
      const response = await fetch(`${process.env.REACT_APP_API_BASE_URL}/failed-callbacks`);
      if (!response.ok) {
        throw new Error('Failed to fetch callbacks');
      }
      const data = await response.json();
      setFailedCallbacks(data);
    } catch (error) {
      message.error(intl.formatMessage({ id: 'error.failedToFetchCallbacks' }));
    } finally {
      setLoading(false);
    }
  };

  const handleRetry = async (id) => {
    setRetryLoading(prev => ({ ...prev, [id]: true }));
    try {
      await fetch(`${process.env.REACT_APP_API_BASE_URL}/failed-callbacks/${id}/retry`, { method: 'POST' });
      message.success(intl.formatMessage({ id: 'success.retryInitiated' }));
      await fetchFailedCallbacks(); // 等待获取最新数据
    } catch (error) {
      message.error(intl.formatMessage({ id: 'error.failedToRetry' }));
    } finally {
      setRetryLoading(prev => ({ ...prev, [id]: false })); // 确保在任何情况下都重置加载状态
    }
  };

  const handleDelete = async (id) => {
    setDeleteLoading(prev => ({ ...prev, [id]: true }));
    try {
      await fetch(`${process.env.REACT_APP_API_BASE_URL}/failed-callbacks/${id}`, { method: 'DELETE' });
      message.success(intl.formatMessage({ id: 'success.callbackDeleted' }));
      fetchFailedCallbacks();
    } catch (error) {
      message.error(intl.formatMessage({ id: 'error.failedToDelete' }));
    }
    setDeleteLoading(prev => ({ ...prev, [id]: false }));
  };

  const handleBulkAction = async (action) => {
    if (selectedRowKeys.length === 0) {
      message.warning(intl.formatMessage({ id: 'warning.noItemsSelected' }));
      return;
    }

    setBulkActionLoading(true);
    try {
      await fetch(`${process.env.REACT_APP_API_BASE_URL}/failed-callbacks/bulk`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ids: selectedRowKeys, action }),
      });
      message.success(intl.formatMessage({ id: `success.bulk${action}Initiated` }));
      fetchFailedCallbacks();
      setSelectedRowKeys([]);
    } catch (error) {
      message.error(intl.formatMessage({ id: `error.failedToBulk${action}` }));
    }
    setBulkActionLoading(false);
  };

  const columns = [
    {
      title: <FormattedMessage id="table.queueName" />,
      dataIndex: 'queue_name',
      key: 'queue_name',
    },
    {
      title: <FormattedMessage id="table.requestData" />,
      dataIndex: 'request_data',
      key: 'request_data',
      ellipsis: true,
    },
    {
      title: <FormattedMessage id="table.responseCode" />,
      dataIndex: 'response_code',
      key: 'response_code',
    },
    {
      title: <FormattedMessage id="table.responseContent" />,
      dataIndex: 'response_content',
      key: 'response_content',
      ellipsis: true,
    },
    {
      title: <FormattedMessage id="table.createdAt" />,
      dataIndex: 'created_at',
      key: 'created_at',
    },
    {
      title: <FormattedMessage id="table.actions" />,
      key: 'actions',
      render: (_, record) => (
        <>
          <Button type="link" onClick={() => handleRetry(record.id)} loading={retryLoading[record.id]}>
            <FormattedMessage id="button.retry" />
          </Button>
          <Popconfirm
            title={<FormattedMessage id="confirm.deleteCallback" />}
            onConfirm={() => handleDelete(record.id)}
            okText={<FormattedMessage id="button.yes" />}
            cancelText={<FormattedMessage id="button.no" />}
          >
            <Button type="link" loading={deleteLoading[record.id]}>
              <FormattedMessage id="button.delete" />
            </Button>
          </Popconfirm>
        </>
      ),
    },
  ];

  const rowSelection = {
    selectedRowKeys,
    onChange: (selectedKeys) => setSelectedRowKeys(selectedKeys),
  };

  const bulkMenu = (
    <Menu>
      <Menu.Item key="retry" onClick={() => handleBulkAction('retry')}>
        <FormattedMessage id="button.retry" />
      </Menu.Item>
      <Menu.Item key="delete" onClick={() => handleBulkAction('delete')}>
        <FormattedMessage id="button.delete" />
      </Menu.Item>
    </Menu>
  );

  return (
    <div>
      <h1><FormattedMessage id="page.failedCallbacks" /></h1>
      <div style={{ marginBottom: 16 }}>
        <Dropdown overlay={bulkMenu} disabled={selectedRowKeys.length === 0 || bulkActionLoading}>
          <Button loading={bulkActionLoading}>
            <FormattedMessage id="button.bulkActions" /> <DownOutlined />
          </Button>
        </Dropdown>
      </div>
      <Table
        rowSelection={rowSelection}
        columns={columns}
        dataSource={failedCallbacks}
        rowKey="id"
        loading={loading}
        pagination={{ pageSize: 10 }}
      />
    </div>
  );
}

export default FailedCallbacks;