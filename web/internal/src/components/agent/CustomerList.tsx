import { Typography, Card, Table } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { apiFetch } from '../../shared/api';

export default function CustomerList() {
    const { data: customers, isLoading } = useQuery({
        queryKey: ['customers'],
        queryFn: () => apiFetch<any[]>('/requesters'), // Use requester/customer listing endpoint accessible to agents
    });

    const columns = [
        { title: 'Name', dataIndex: 'display_name', key: 'display_name' },
        { title: 'Email', dataIndex: 'email', key: 'email' },
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
                />
            </Card>
        </div>
    );
}
