import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Input, Button, List } from 'antd';
import { apiFetch } from '../../shared/api';

export default function UserRoleManagement() {
  const [userId, setUserId] = useState('');
  const { data, refetch, isFetching, error } = useQuery({
    queryKey: ['user-roles', userId],
    queryFn: () => apiFetch<string[]>(`/users/${userId}/roles`),
    enabled: false,
  });

  return (
    <div>
      <h2>User & Role Management</h2>
      <Input
        placeholder="User ID"
        value={userId}
        onChange={(e) => setUserId(e.target.value)}
        style={{ width: 200, marginRight: 8 }}
      />
      <Button onClick={() => refetch()} disabled={!userId} loading={isFetching}>
        Load Roles
      </Button>
      {error && <div>Failed to load roles</div>}
      {data && (
        <List
          dataSource={data}
          renderItem={(r) => <List.Item>{r}</List.Item>}
          style={{ marginTop: 16 }}
        />
      )}
    </div>
  );
}
