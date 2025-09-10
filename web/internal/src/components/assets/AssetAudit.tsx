import { useState } from "react";
import {
  Card,
  Table,
  Select,
  DatePicker,
  Input,
  Button,
  Space,
  Timeline,
  Tag,
  Statistic,
  Row,
  Col,
} from "antd";
import {
  SearchOutlined,
  HistoryOutlined,
  UserOutlined,
  ClockCircleOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  ExportOutlined,
  ImportOutlined,
  UserAddOutlined,
  UserDeleteOutlined,
  FileTextOutlined,
} from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../shared/api";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";

dayjs.extend(relativeTime);

const { RangePicker } = DatePicker;
const { Option } = Select;

interface AuditEntry {
  id: string;
  asset_id: string;
  asset_name: string;
  asset_tag: string;
  user_id: string;
  user_name: string;
  action: string;
  field_name?: string;
  old_value?: string;
  new_value?: string;
  notes?: string;
  timestamp: string;
  ip_address?: string;
}

interface AuditSummary {
  total_events: number;
  events_last_30_days: number;
  most_active_user: string;
  most_modified_asset: string;
  action_breakdown: Record<string, number>;
}

export default function AssetAudit() {
  const [filters, setFilters] = useState({
    asset_id: undefined,
    user_id: undefined,
    action: undefined,
    date_range: undefined,
    search: "",
  });

  // Fetch audit history
  const { data: auditHistory, isLoading: historyLoading } = useQuery<{
    items: AuditEntry[];
    total: number;
  }>({
    queryKey: ["asset-audit-history", filters],
    queryFn: async () => {
      const params = new URLSearchParams();
      if (filters.asset_id) params.append("asset_id", filters.asset_id);
      if (filters.user_id) params.append("user_id", filters.user_id);
      if (filters.action) params.append("action", filters.action);
      if (filters.date_range) {
        params.append("start_date", filters.date_range[0]);
        params.append("end_date", filters.date_range[1]);
      }
      if (filters.search) params.append("search", filters.search);

      const response = await api.get(`/assets/audit?${params.toString()}`);
      return (response as any).data;
    },
  });

  // Fetch audit summary
  const { data: auditSummary } = useQuery<AuditSummary>({
    queryKey: ["asset-audit-summary"],
    queryFn: async () => {
      const response = await api.get("/assets/audit/summary");
      return (response as any).data;
    },
  });

  // Fetch assets for filter dropdown
  const { data: assets } = useQuery({
    queryKey: ["assets-for-audit"],
    queryFn: async () => {
      const response = await api.get("/assets?limit=1000");
      return (response as any).data.items;
    },
  });

  const handleFilterChange = (key: string, value: any) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

  const clearFilters = () => {
    setFilters({
      asset_id: undefined,
      user_id: undefined,
      action: undefined,
      date_range: undefined,
      search: "",
    });
  };

  const getActionColor = (action: string) => {
    switch (action.toLowerCase()) {
      case "create":
        return "green";
      case "update":
        return "blue";
      case "delete":
        return "red";
      case "checkout":
        return "orange";
      case "checkin":
        return "purple";
      case "assign":
        return "cyan";
      case "unassign":
        return "geekblue";
      default:
        return "default";
    }
  };

  const getActionIcon = (action: string) => {
    switch (action.toLowerCase()) {
      case "create":
        return <PlusOutlined aria-label="create" />;
      case "update":
        return <EditOutlined aria-label="update" />;
      case "delete":
        return <DeleteOutlined aria-label="delete" />;
      case "checkout":
        return <ExportOutlined aria-label="checkout" />;
      case "checkin":
        return <ImportOutlined aria-label="checkin" />;
      case "assign":
        return <UserAddOutlined aria-label="assign" />;
      case "unassign":
        return <UserDeleteOutlined aria-label="unassign" />;
      default:
        return <FileTextOutlined aria-label="action" />;
    }
  };

  const columns = [
    {
      title: "Timestamp",
      dataIndex: "timestamp",
      key: "timestamp",
      render: (timestamp: string) => (
        <div>
          <div>{dayjs(timestamp).format("MMM D, YYYY")}</div>
          <div style={{ fontSize: "12px", color: "#666" }}>
            {dayjs(timestamp).format("h:mm A")}
          </div>
        </div>
      ),
      width: 120,
    },
    {
      title: "Asset",
      key: "asset",
      render: (record: AuditEntry) => (
        <div>
          <div style={{ fontWeight: 500 }}>{record.asset_name}</div>
          <div style={{ fontSize: "12px", color: "#666" }}>
            {record.asset_tag}
          </div>
        </div>
      ),
      width: 200,
    },
    {
      title: "Action",
      dataIndex: "action",
      key: "action",
      render: (action: string) => (
        <Tag color={getActionColor(action)} icon={getActionIcon(action)}>
          {action.toUpperCase()}
        </Tag>
      ),
      width: 100,
    },
    {
      title: "User",
      dataIndex: "user_name",
      key: "user_name",
      render: (name: string) => (
        <Space>
          <UserOutlined />
          {name}
        </Space>
      ),
      width: 150,
    },
    {
      title: "Changes",
      key: "changes",
      render: (record: AuditEntry) => {
        if (record.field_name && record.old_value && record.new_value) {
          return (
            <div>
              <div style={{ fontSize: "12px", color: "#666" }}>
                <strong>{record.field_name}</strong>
              </div>
              <div style={{ fontSize: "12px" }}>
                <span
                  style={{ color: "#ff4d4f", textDecoration: "line-through" }}
                >
                  {record.old_value}
                </span>
                {" → "}
                <span style={{ color: "#52c41a" }}>{record.new_value}</span>
              </div>
            </div>
          );
        }
        return record.notes || "-";
      },
      width: 250,
    },
    {
      title: "IP Address",
      dataIndex: "ip_address",
      key: "ip_address",
      width: 120,
    },
  ];

  const timelineItems = auditHistory?.items?.slice(0, 20).map((entry) => ({
    color: getActionColor(entry.action),
    dot: (
      <span style={{ fontSize: "16px" }}>{getActionIcon(entry.action)}</span>
    ),
    children: (
      <div>
        <div style={{ fontWeight: 500 }}>
          {entry.asset_name} ({entry.asset_tag})
        </div>
        <div style={{ fontSize: "12px", color: "#666" }}>
          {entry.action.toUpperCase()} by {entry.user_name} •{" "}
          {dayjs(entry.timestamp).fromNow()}
        </div>
        {entry.field_name && (
          <div style={{ fontSize: "12px", marginTop: 4 }}>
            {entry.field_name}: {entry.old_value} → {entry.new_value}
          </div>
        )}
      </div>
    ),
  }));

  return (
    <div>
      <h2>Asset Audit & Analytics</h2>

      <Space direction="vertical" size="large" style={{ width: "100%" }}>
        {/* Summary Statistics */}
        {auditSummary && (
          <Card title="Audit Summary">
            <Row gutter={[16, 16]}>
              <Col xs={24} sm={12} md={6}>
                <Statistic
                  title="Total Events"
                  value={auditSummary.total_events}
                  prefix={<HistoryOutlined />}
                />
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Statistic
                  title="Last 30 Days"
                  value={auditSummary.events_last_30_days}
                  prefix={<ClockCircleOutlined />}
                />
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Statistic
                  title="Most Active User"
                  value={auditSummary.most_active_user}
                  prefix={<UserOutlined />}
                />
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Statistic
                  title="Most Modified"
                  value={auditSummary.most_modified_asset}
                />
              </Col>
            </Row>

            {auditSummary.action_breakdown && (
              <div style={{ marginTop: 16 }}>
                <h4>Action Breakdown</h4>
                <Space wrap>
                  {Object.entries(auditSummary.action_breakdown).map(
                    ([action, count]) => (
                      <Tag key={action} color={getActionColor(action)}>
                        {action}: {count}
                      </Tag>
                    ),
                  )}
                </Space>
              </div>
            )}
          </Card>
        )}

        <Row gutter={[16, 16]}>
          {/* Audit History Table */}
          <Col xs={24} lg={16}>
            <Card
              title="Audit History"
              extra={
                <Space>
                  <Button size="small" onClick={clearFilters}>
                    Clear Filters
                  </Button>
                </Space>
              }
            >
              {/* Filters */}
              <Space wrap style={{ marginBottom: 16 }}>
                <Input
                  placeholder="Search assets..."
                  prefix={<SearchOutlined />}
                  value={filters.search}
                  onChange={(e) => handleFilterChange("search", e.target.value)}
                  style={{ width: 200 }}
                />

                <Select
                  placeholder="Filter by asset"
                  style={{ width: 200 }}
                  allowClear
                  showSearch
                  optionFilterProp="children"
                  value={filters.asset_id}
                  onChange={(value) => handleFilterChange("asset_id", value)}
                >
                  {assets?.map((asset: any) => (
                    <Option key={asset.id} value={asset.id}>
                      {asset.name} ({asset.asset_tag})
                    </Option>
                  ))}
                </Select>

                <Select
                  placeholder="Filter by action"
                  style={{ width: 150 }}
                  allowClear
                  value={filters.action}
                  onChange={(value) => handleFilterChange("action", value)}
                >
                  <Option value="create">Create</Option>
                  <Option value="update">Update</Option>
                  <Option value="delete">Delete</Option>
                  <Option value="checkout">Checkout</Option>
                  <Option value="checkin">Checkin</Option>
                  <Option value="assign">Assign</Option>
                  <Option value="unassign">Unassign</Option>
                </Select>

                <RangePicker
                  value={filters.date_range}
                  onChange={(dates) =>
                    handleFilterChange(
                      "date_range",
                      dates
                        ? [
                            dates[0]?.format("YYYY-MM-DD"),
                            dates[1]?.format("YYYY-MM-DD"),
                          ]
                        : undefined,
                    )
                  }
                />
              </Space>

              <Table
                dataSource={auditHistory?.items}
                columns={columns}
                rowKey="id"
                loading={historyLoading}
                pagination={{
                  pageSize: 50,
                  total: auditHistory?.total,
                  showSizeChanger: true,
                  showQuickJumper: true,
                }}
                scroll={{ x: 1000 }}
                size="small"
              />
            </Card>
          </Col>

          {/* Recent Activity Timeline */}
          <Col xs={24} lg={8}>
            <Card
              title="Recent Activity"
              style={{ height: 600, overflow: "auto" }}
            >
              <Timeline items={timelineItems} />
            </Card>
          </Col>
        </Row>
      </Space>
    </div>
  );
}
