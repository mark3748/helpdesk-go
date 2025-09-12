import { Card, Row, Col, Statistic, Table, Alert } from "antd";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../shared/api";

interface DashboardData {
  sla: { total: number; met: number; sla_attainment: number };
  avg_resolution_ms: number;
  volume: Array<{ day: string; count: number }>;
}

export default function AssetDashboard() {
  const { data, isLoading, error } = useQuery<DashboardData>({
    queryKey: ["metrics", "dashboard"],
    queryFn: async () => (await api.get("/metrics/dashboard")).data,
  });

  if (error) {
    return <Alert message="Failed to load metrics" type="error" showIcon />;
  }

  return (
    <div style={{ padding: 24 }}>
      <Row gutter={[16, 16]}>
        <Col xs={24} md={8}>
          <Card loading={isLoading}>
            <Statistic
              title="SLA Attainment"
              value={(data?.sla.sla_attainment ?? 0) * 100}
              precision={2}
              suffix="%"
            />
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card loading={isLoading}>
            <Statistic
              title="Avg Resolution (ms)"
              value={data?.avg_resolution_ms ?? 0}
            />
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card loading={isLoading}>
            <Statistic
              title="Tickets Resolved"
              value={data?.sla.met ?? 0}
              suffix={`/ ${data?.sla.total ?? 0}`}
            />
          </Card>
        </Col>
      </Row>

      <Card
        title="Ticket Volume (last 30 days)"
        style={{ marginTop: 24 }}
        loading={isLoading}
      >
        <Table
          dataSource={data?.volume || []}
          columns={[
            { title: "Day", dataIndex: "day", key: "day" },
            { title: "Count", dataIndex: "count", key: "count" },
          ]}
          pagination={false}
          size="small"
          rowKey={(row) => row.day}
        />
      </Card>
    </div>
  );
}
