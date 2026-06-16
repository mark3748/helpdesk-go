import { Card, Row, Col, Statistic, Table, Alert, Typography } from "antd";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../shared/api";

interface DashboardData {
  summary: {
    total_assets: number;
    active_assets: number;
    maintenance_assets: number;
    retired_assets: number;
    total_value: number;
    depreciated_value: number;
  };
  status_distribution: Array<{ status: string; count: number; percentage: number }>;
  category_breakdown: Array<{ category: string; count: number; total_value: number; avg_age_months: number }>;
}

export default function AssetDashboard() {
  const { data, isLoading, error } = useQuery<DashboardData>({
    queryKey: ["assets", "dashboard"],
    queryFn: async () => {
      const response = await api.get<{ data: DashboardData }>("/assets/analytics");
      return response.data;
    },
  });

  if (error) {
    return <Alert message="Failed to load asset dashboard" type="error" showIcon />;
  }

  return (
    <div style={{ padding: 24 }}>
      <Typography.Title level={2}>Asset Dashboard</Typography.Title>
      <Row gutter={[16, 16]}>
        <Col xs={24} md={8}>
          <Card loading={isLoading}>
            <Statistic
              title="Total Assets"
              value={data?.summary.total_assets ?? 0}
            />
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card loading={isLoading}>
            <Statistic
              title="Active Assets"
              value={data?.summary.active_assets ?? 0}
              suffix={`/ ${data?.summary.total_assets ?? 0}`}
            />
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card loading={isLoading}>
            <Statistic
              title="Maintenance"
              value={data?.summary.maintenance_assets ?? 0}
            />
          </Card>
        </Col>
      </Row>
      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} md={12}>
          <Card loading={isLoading}>
            <Statistic
              title="Total Value"
              value={data?.summary.total_value ?? 0}
              precision={2}
              prefix="$"
            />
          </Card>
        </Col>
        <Col xs={24} md={12}>
          <Card loading={isLoading}>
            <Statistic
              title="Depreciated Value"
              value={data?.summary.depreciated_value ?? 0}
              precision={2}
              prefix="$"
            />
          </Card>
        </Col>
      </Row>

      <Card
        title="Assets by Status"
        style={{ marginTop: 24 }}
        loading={isLoading}
      >
        <Table
          dataSource={data?.status_distribution || []}
          columns={[
            { title: "Status", dataIndex: "status", key: "status" },
            { title: "Count", dataIndex: "count", key: "count" },
            { title: "Share", dataIndex: "percentage", key: "percentage", render: (value: number) => `${value.toFixed(1)}%` },
          ]}
          pagination={false}
          size="small"
          rowKey={(row) => row.status}
        />
      </Card>

      <Card
        title="Assets by Category"
        style={{ marginTop: 24 }}
        loading={isLoading}
      >
        <Table
          dataSource={data?.category_breakdown || []}
          columns={[
            { title: "Category", dataIndex: "category", key: "category" },
            { title: "Count", dataIndex: "count", key: "count" },
            { title: "Total Value", dataIndex: "total_value", key: "total_value", render: (value: number) => `$${value.toFixed(2)}` },
            { title: "Avg Age (months)", dataIndex: "avg_age_months", key: "avg_age_months" },
          ]}
          pagination={false}
          size="small"
          rowKey={(row) => row.category}
        />
      </Card>
    </div>
  );
}
