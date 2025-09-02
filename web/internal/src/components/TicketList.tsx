import { List } from 'antd';
import { useTickets } from '../api';

export default function TicketList() {
  const { data } = useTickets();
  return (
    <List
      dataSource={data || []}
      renderItem={(t) => <List.Item key={String((t as any).id)}>{String((t as any).title || (t as any).number)}</List.Item>}
    />
  );
}
