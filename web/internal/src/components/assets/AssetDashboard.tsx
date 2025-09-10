import {
  Card,
  Row,
  Col,
  Statistic,
  Progress,
  Table,
  Tag,
  Typography,
  Alert,
  List,
} from "antd";
import {
  DesktopOutlined,
  ToolOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
} from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { api } from "../../shared/api";

const { Title } = Typography;

interface DashboardStats {
  total_assets: number;
  active_assets: number;
  maintenance_assets: number;
  retired_assets: number;
  active_checkouts: number;
  overdue_checkouts: number;
  assets_by_status: Record<string, number>;
  assets_by_condition: Record<string, number>;
  recent_activity: Array<{
    id: string;
    action: string;
    asset_name: string;
    asset_tag: string;
    actor: string;
    created_at: string;
  }>;
  overdue_items: Array<{
    id: string;
    asset_tag: string;
    asset_name: string;
    checked_out_to: string;
    expected_return_date: string;
    days_overdue: number;
  }>;
  upcoming_maintenance: Array<{
    id: string;
    asset_tag: string;
    asset_name: string;
    next_maintenance_date: string;
    days_until: number;
  }>;
}

const statusColors = {
  active: "#52c41a",
  inactive: "#d9d9d9",
  maintenance: "#faad14",
  retired: "#ff4d4f",
  disposed: "#ff7875",
};

const conditionColors = {
  excellent: "#52c41a",
  good: "#1890ff",
  fair: "#faad14",
  poor: "#ff7a45",
  broken: "#ff4d4f",
};

export default function AssetDashboard() {
  const { data, isLoading, error } = useQuery<DashboardStats>({
    queryKey: ["assets", "dashboard"],
    queryFn: async () => {
      // TODO: Replace mock data with real dashboard endpoint
      await api.get("/assets?limit=1");

      // Placeholder dashboard data to be replaced once endpoint exists
      return {
        total_assets: 150,
        active_assets: 125,
        maintenance_assets: 15,
        retired_assets: 10,
        active_checkouts: 23,
        overdue_checkouts: 3,
        assets_by_status: {
          active: 125,
          inactive: 8,
          maintenance: 15,
          retired: 10,
          disposed: 2,
        },
        assets_by_condition: {
          excellent: 45,
          good: 78,
          fair: 20,
          poor: 5,
          broken: 2,
        },
        recent_activity: [
          {
            id: "1",
            action: "Asset Created",
            asset_name: "Dell Laptop",
            asset_tag: "LP-001",
            actor: "admin@company.com",
            created_at: "2024-01-15T10:30:00Z",
          },
          {
            id: "2",
            action: "Asset Assigned",
            asset_name: "iPhone 15",
            asset_tag: "PH-042",
            actor: "manager@company.com",
            created_at: "2024-01-15T09:15:00Z",
          },
        ],
        overdue_items: [
          {
            id: "1",
            asset_tag: "CAM-001",
            asset_name: "Canon Camera",
            checked_out_to: "john.doe@company.com",
            expected_return_date: "2024-01-10",
            days_overdue: 5,
          },
        ],
        upcoming_maintenance: [
          {
            id: "1",
            asset_tag: "SRV-001",
            asset_name: "Main Server",
            next_maintenance_date: "2024-01-20",
            days_until: 3,
          },
        ],
      };
    },
    refetchInterval: 30000, // Refresh every 30 seconds
  });

  if (error) {
    return (
      <Alert
        message="Error"
        description="Failed to load dashboard data"
        type="error"
        showIcon
      />
    );
  }

  const overallHealth = data
    ? Math.round((data.active_assets / data.total_assets) * 100)
    : 0;
  const utilizationRate = data
    ? Math.round((data.active_checkouts / data.active_assets) * 100)
    : 0;

  return (
    <div>
      <Title level={2}>Asset Dashboard</Title>

      {/* Key Metrics */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={12} sm={8} lg={6}>
          <Card>
            <Statistic
              title="Total Assets"
              value={data?.total_assets || 0}
              loading={isLoading}
              prefix={<DesktopOutlined />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={6}>
          <Card>
            <Statistic
              title="Active Assets"
              value={data?.active_assets || 0}
              loading={isLoading}
              valueStyle={{ color: "#52c41a" }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={6}>
          <Card>
            <Statistic
              title="In Maintenance"
              value={data?.maintenance_assets || 0}
              loading={isLoading}
              valueStyle={{ color: "#faad14" }}
              prefix={<ToolOutlined />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} lg={6}>
          <Card>
            <Statistic
              title="Active Checkouts"
              value={data?.active_checkouts || 0}
              loading={isLoading}
              prefix={<ClockCircleOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* Health & Utilization */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12}>
          <Card title="Asset Health" loading={isLoading}>
            <Progress
              type="circle"
              percent={overallHealth}
              format={(percent) => `${percent}%`}
              status={
                overallHealth > 90
                  ? "success"
                  : overallHealth > 70
                    ? "normal"
                    : "exception"
              }
            />
            <div style={{ marginTop: 16, textAlign: "center" }}>
              <div>Overall Health Score</div>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12}>
          <Card title="Utilization Rate" loading={isLoading}>
            <Progress
              type="circle"
              percent={utilizationRate}
              format={(percent) => `${percent}%`}
              strokeColor="#1890ff"
            />
            <div style={{ marginTop: 16, textAlign: "center" }}>
              <div>Assets Currently in Use</div>
            </div>
          </Card>
        </Col>
      </Row>

      {/* Alerts & Issues */}
      {data &&
        (data.overdue_checkouts > 0 ||
          data.upcoming_maintenance.length > 0) && (
          <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
            {data.overdue_checkouts > 0 && (
              <Col xs={24} lg={12}>
                <Card
                  title="Overdue Returns"
                  extra={<Link to="/assets/checkouts/overdue">View All</Link>}
                >
                  <Alert
                    message={`${data.overdue_checkouts} overdue checkout${data.overdue_checkouts !== 1 ? "s" : ""}`}
                    type="error"
                    showIcon
                    style={{ marginBottom: 16 }}
                  />
                  <List
                    size="small"
                    dataSource={data.overdue_items}
                    renderItem={(item) => (
                      <List.Item>
                        <div>
                          <strong>{item.asset_tag}</strong> - {item.asset_name}
                          <br />
                          <small>
                            Checked out to: {item.checked_out_to} •{" "}
                            {item.days_overdue} days overdue
                          </small>
                        </div>
                      </List.Item>
                    )}
                  />
                </Card>
              </Col>
            )}

            {data.upcoming_maintenance.length > 0 && (
              <Col xs={24} lg={12}>
                <Card
                  title="Upcoming Maintenance"
                  extra={<Link to="/assets/maintenance">View All</Link>}
                >
                  <Alert
                    message={`${data.upcoming_maintenance.length} asset${data.upcoming_maintenance.length !== 1 ? "s" : ""} need maintenance soon`}
                    type="warning"
                    showIcon
                    style={{ marginBottom: 16 }}
                  />
                  <List
                    size="small"
                    dataSource={data.upcoming_maintenance}
                    renderItem={(item) => (
                      <List.Item>
                        <div>
                          <strong>{item.asset_tag}</strong> - {item.asset_name}
                          <br />
                          <small>Due in {item.days_until} days</small>
                        </div>
                      </List.Item>
                    )}
                  />
                </Card>
              </Col>
            )}
          </Row>
        )}

      {/* Status Distribution */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} lg={12}>
          <Card title="Assets by Status" loading={isLoading}>
            {data?.assets_by_status &&
              Object.entries(data.assets_by_status).map(([status, count]) => (
                <div key={status} style={{ marginBottom: 8 }}>
                  <div
                    style={{
                      display: "flex",
                      justifyContent: "space-between",
                      alignItems: "center",
                    }}
                  >
                    <span style={{ textTransform: "capitalize" }}>
                      {status}
                    </span>
                    <span>{count}</span>
                  </div>
                  <Progress
                    percent={(count / data.total_assets) * 100}
                    strokeColor={
                      statusColors[status as keyof typeof statusColors]
                    }
                    showInfo={false}
                    size="small"
                  />
                </div>
              ))}
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <Card title="Assets by Condition" loading={isLoading}>
            {data?.assets_by_condition &&
              Object.entries(data.assets_by_condition).map(
                ([condition, count]) => (
                  <div key={condition} style={{ marginBottom: 8 }}>
                    <div
                      style={{
                        display: "flex",
                        justifyContent: "space-between",
                        alignItems: "center",
                      }}
                    >
                      <span style={{ textTransform: "capitalize" }}>
                        {condition}
                      </span>
                      <span>{count}</span>
                    </div>
                    <Progress
                      percent={(count / data.total_assets) * 100}
                      strokeColor={
                        conditionColors[
                          condition as keyof typeof conditionColors
                        ]
                      }
                      showInfo={false}
                      size="small"
                    />
                  </div>
                ),
              )}
          </Card>
        </Col>
      </Row>

      {/* Recent Activity */}
      <Card title="Recent Activity" loading={isLoading}>
        <Table
          dataSource={data?.recent_activity || []}
          columns={[
            {
              title: "Action",
              dataIndex: "action",
              key: "action",
              render: (action) => <Tag>{action}</Tag>,
            },
            {
              title: "Asset",
              key: "asset",
              render: (_, record) => (
                <div>
                  <strong>{record.asset_tag}</strong> - {record.asset_name}
                </div>
              ),
            },
            {
              title: "Actor",
              dataIndex: "actor",
              key: "actor",
            },
            {
              title: "Time",
              dataIndex: "created_at",
              key: "created_at",
              render: (date) => new Date(date).toLocaleString(),
            },
          ]}
          pagination={false}
          size="small"
        />
      </Card>
    </div>
  );
}
