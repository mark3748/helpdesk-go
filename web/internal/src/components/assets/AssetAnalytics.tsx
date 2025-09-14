import { useState } from 'react';
import { Card, Row, Col, Statistic, Select, DatePicker, Table, Progress, Space, Typography } from 'antd';
import { LineChart, Line, BarChart, Bar, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { useQuery } from '@tanstack/react-query';
import { api } from '../../shared/api';
import dayjs from 'dayjs';

const { RangePicker } = DatePicker;
const { Option } = Select;
const { Title } = Typography;

interface AssetAnalytics {
  summary: {
    total_assets: number;
    active_assets: number;
    maintenance_assets: number;
    retired_assets: number;
    total_value: number;
    depreciated_value: number;
  };
  status_distribution: Array<{
    status: string;
    count: number;
    percentage: number;
  }>;
  condition_distribution: Array<{
    condition: string;
    count: number;
    percentage: number;
  }>;
  category_breakdown: Array<{
    category: string;
    count: number;
    total_value: number;
    avg_age_months: number;
  }>;
  age_analysis: Array<{
    age_range: string;
    count: number;
    percentage: number;
  }>;
  monthly_acquisitions: Array<{
    month: string;
    count: number;
    value: number;
  }>;
  top_manufacturers: Array<{
    manufacturer: string;
    count: number;
    percentage: number;
  }>;
  location_distribution: Array<{
    location: string;
    count: number;
    percentage: number;
  }>;
  depreciation_trend: Array<{
    month: string;
    book_value: number;
    market_value: number;
  }>;
}

const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042', '#8884D8', '#82CA9D'];

export default function AssetAnalytics() {
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null);
  const [categoryFilter, setCategoryFilter] = useState<string | undefined>(undefined);

  // Fetch analytics data
  const { data: analytics, isLoading } = useQuery<AssetAnalytics>({
    queryKey: ['asset-analytics', { dateRange, categoryFilter }],
    queryFn: async () => {
      const params = new URLSearchParams();
      if (dateRange) {
        params.set('start_date', dateRange[0].format('YYYY-MM-DD'));
        params.set('end_date', dateRange[1].format('YYYY-MM-DD'));
      }
      if (categoryFilter) {
        params.set('category', categoryFilter);
      }

      const response = await api.get(`/assets/analytics?${params.toString()}`);
      return (response as any).data;
    },
  });

  // Fetch categories for filter
  const { data: categories } = useQuery({
    queryKey: ['asset-categories'],
    queryFn: async () => {
      const response = await api.get('/asset-categories');
      return (response as any).data;
    },
  });

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
    }).format(value);
  };

  const formatPercentage = (value: number) => {
    return `${value.toFixed(1)}%`;
  };

  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={2} style={{ margin: 0 }}>Asset Analytics</Title>
        <Space>
          <Select
            placeholder="Filter by category"
            style={{ width: 200 }}
            allowClear
            value={categoryFilter}
            onChange={setCategoryFilter}
          >
            {categories?.map((category: any) => (
              <Option key={category.id} value={category.id}>
                {category.name}
              </Option>
            ))}
          </Select>
          <RangePicker
            value={dateRange}
            onChange={(dates) => setDateRange(dates as [dayjs.Dayjs, dayjs.Dayjs] | null)}
            placeholder={['Start Date', 'End Date']}
          />
        </Space>
      </div>

      {analytics && (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          {/* Summary Statistics */}
          <Row gutter={[16, 16]}>
            <Col xs={24} sm={12} md={6}>
              <Card loading={isLoading}>
                <Statistic
                  title="Total Assets"
                  value={analytics.summary.total_assets}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Card loading={isLoading}>
                <Statistic
                  title="Active Assets"
                  value={analytics.summary.active_assets}
                  suffix={`/ ${analytics.summary.total_assets}`}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Card loading={isLoading}>
                <Statistic
                  title="Total Value"
                  value={analytics.summary.total_value}
                  formatter={(value) => formatCurrency(Number(value))}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Card loading={isLoading}>
                <Statistic
                  title="Depreciated Value"
                  value={analytics.summary.depreciated_value}
                  formatter={(value) => formatCurrency(Number(value))}
                />
              </Card>
            </Col>
          </Row>

          <Row gutter={[16, 16]}>
            {/* Asset Status Distribution */}
            <Col xs={24} lg={12}>
              <Card title="Asset Status Distribution" loading={isLoading}>
                <ResponsiveContainer width="100%" height={300}>
                  <PieChart>
                    <Pie
                      data={analytics.status_distribution}
                      cx="50%"
                      cy="50%"
                      labelLine={false}
                      label={({ status, percentage }) => `${status}: ${formatPercentage(percentage)}`}
                      outerRadius={80}
                      fill="#8884d8"
                      dataKey="count"
                    >
                      {analytics.status_distribution.map((_, index) => (
                        <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                      ))}
                    </Pie>
                    <Tooltip />
                  </PieChart>
                </ResponsiveContainer>
              </Card>
            </Col>

            {/* Asset Condition Distribution */}
            <Col xs={24} lg={12}>
              <Card title="Asset Condition Distribution" loading={isLoading}>
                <ResponsiveContainer width="100%" height={300}>
                  <BarChart data={analytics.condition_distribution}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="condition" />
                    <YAxis />
                    <Tooltip />
                    <Bar dataKey="count" fill="#8884d8" />
                  </BarChart>
                </ResponsiveContainer>
              </Card>
            </Col>
          </Row>

          {/* Monthly Acquisitions Trend */}
          <Card title="Monthly Asset Acquisitions" loading={isLoading}>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={analytics.monthly_acquisitions}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="month" />
                <YAxis yAxisId="left" />
                <YAxis yAxisId="right" orientation="right" />
                <Tooltip
                  formatter={(value, name) => [
                    name === 'value' ? formatCurrency(Number(value)) : value,
                    name === 'value' ? 'Total Value' : 'Count'
                  ]}
                />
                <Legend />
                <Bar yAxisId="left" dataKey="count" fill="#8884d8" name="Count" />
                <Line yAxisId="right" type="monotone" dataKey="value" stroke="#82ca9d" name="Value" />
              </BarChart>
            </ResponsiveContainer>
          </Card>

          <Row gutter={[16, 16]}>
            {/* Category Breakdown */}
            <Col xs={24} lg={12}>
              <Card title="Assets by Category" loading={isLoading}>
                <Table
                  dataSource={analytics.category_breakdown}
                  columns={[
                    {
                      title: 'Category',
                      dataIndex: 'category',
                      key: 'category',
                    },
                    {
                      title: 'Count',
                      dataIndex: 'count',
                      key: 'count',
                      width: 80,
                    },
                    {
                      title: 'Total Value',
                      dataIndex: 'total_value',
                      key: 'total_value',
                      width: 120,
                      render: (value: number) => formatCurrency(value),
                    },
                    {
                      title: 'Avg Age (months)',
                      dataIndex: 'avg_age_months',
                      key: 'avg_age_months',
                      width: 120,
                      render: (value: number) => Math.round(value),
                    },
                  ]}
                  pagination={false}
                  size="small"
                />
              </Card>
            </Col>

            {/* Top Manufacturers */}
            <Col xs={24} lg={12}>
              <Card title="Top Manufacturers" loading={isLoading}>
                <Table
                  dataSource={analytics.top_manufacturers}
                  columns={[
                    {
                      title: 'Manufacturer',
                      dataIndex: 'manufacturer',
                      key: 'manufacturer',
                    },
                    {
                      title: 'Count',
                      dataIndex: 'count',
                      key: 'count',
                      width: 80,
                    },
                    {
                      title: 'Percentage',
                      dataIndex: 'percentage',
                      key: 'percentage',
                      width: 100,
                      render: (value: number) => (
                        <div>
                          <Progress percent={value} size="small" />
                          {formatPercentage(value)}
                        </div>
                      ),
                    },
                  ]}
                  pagination={false}
                  size="small"
                />
              </Card>
            </Col>
          </Row>

          {/* Asset Age Analysis */}
          <Card title="Asset Age Distribution" loading={isLoading}>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={analytics.age_analysis}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="age_range" />
                <YAxis />
                <Tooltip />
                <Bar dataKey="count" fill="#8884d8" />
              </BarChart>
            </ResponsiveContainer>
          </Card>

          {/* Depreciation Trend */}
          <Card title="Asset Value Depreciation Trend" loading={isLoading}>
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={analytics.depreciation_trend}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="month" />
                <YAxis />
                <Tooltip formatter={(value) => formatCurrency(Number(value))} />
                <Legend />
                <Line type="monotone" dataKey="book_value" stroke="#8884d8" name="Book Value" />
                <Line type="monotone" dataKey="market_value" stroke="#82ca9d" name="Market Value" />
              </LineChart>
            </ResponsiveContainer>
          </Card>

          {/* Location Distribution */}
          <Card title="Assets by Location" loading={isLoading}>
            <Table
              dataSource={analytics.location_distribution}
              columns={[
                {
                  title: 'Location',
                  dataIndex: 'location',
                  key: 'location',
                },
                {
                  title: 'Count',
                  dataIndex: 'count',
                  key: 'count',
                  width: 100,
                },
                {
                  title: 'Percentage',
                  dataIndex: 'percentage',
                  key: 'percentage',
                  width: 200,
                  render: (value: number) => (
                    <div>
                      <Progress percent={value} size="small" />
                      {formatPercentage(value)}
                    </div>
                  ),
                },
              ]}
              pagination={{ pageSize: 10 }}
              size="small"
            />
          </Card>
        </Space>
      )}
    </div>
  );
}