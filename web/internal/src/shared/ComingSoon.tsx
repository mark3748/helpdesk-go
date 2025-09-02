import { Card, Typography } from 'antd';

export default function ComingSoon({ title = 'Coming soon', detail }: { title?: string; detail?: string }) {
  return (
    <Card style={{ maxWidth: 720 }}>
      <Typography.Title level={4} style={{ marginTop: 0 }}>{title}</Typography.Title>
      <Typography.Paragraph type="secondary">
        {detail || 'This section is not implemented yet. Functionality will be added in a future release.'}
      </Typography.Paragraph>
    </Card>
  );
}

