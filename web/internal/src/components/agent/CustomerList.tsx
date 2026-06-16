import { Typography, Card, Table, Empty } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { apiFetch } from '../../shared/api';
import type { Requester } from '../../shared/api';

export default function CustomerList() {
    const { data: customers, isLoading } = useQuery({
        queryKey: ['customers'],
        queryFn: () => apiFetch<Requester[]>('/requesters'),
    });

    const columns = [
        { title: 'Name', dataIndex: 'display_name', key: 'display_name' },
        { title: 'Email', dataIndex: 'email', key: 'email' },
        { title: 'Phone', dataIndex: 'phone', key: 'phone' },
    ];

    return (
        <div style={{ padding: 24 }}>
            <Typography.Title level={2}>Customers</Typography.Title>
            <Card>
                <Table
                    dataSource={customers || []}
                    columns={columns}
                    loading={isLoading}
                    rowKey="id"
                    locale={{ emptyText: <Empty description="No requesters found" /> }}
                />
            </Card>
        </div>
    );
}
