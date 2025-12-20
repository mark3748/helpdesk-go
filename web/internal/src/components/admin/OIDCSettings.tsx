import { useEffect } from 'react';
import { Form, Input, Button, message, Space, Checkbox } from 'antd';
import { useSettings, useSaveOIDCSettings } from '../../api';

export default function OIDCSettings() {
  const [form] = Form.useForm();
  const { data } = useSettings();
  const save = useSaveOIDCSettings();

  useEffect(() => {
    if ((data as any)?.oidc) {
      const oidc = (data as any).oidc as any;
      form.setFieldsValue({
        ...oidc,
        value_to_roles: oidc.value_to_roles
          ? Object.entries(oidc.value_to_roles).map(([value, roles]: any) => ({
            value,
            roles: (roles as string[]).join(','),
          }))
          : [],
      });
    }
  }, [data, form]);

  const onFinish = (values: any) => {
    const mappings: Record<string, string[]> = {};
    (values.value_to_roles || []).forEach((m: any) => {
      mappings[m.value] = m.roles
        .split(',')
        .map((r: string) => r.trim())
        .filter(Boolean);
    });
    const payload = { ...values, value_to_roles: mappings };
    save.mutate(payload, {
      onSuccess: () => message.success('OIDC settings saved'),
      onError: () => message.error('Failed to save'),
    });
  };

  return (
    <>
      <div style={{ marginBottom: 24, padding: '12px 16px', background: '#f5f5f5', borderRadius: 8 }}>
        <p style={{ margin: 0 }}>
          <strong>Redirect URI:</strong> {window.location.protocol}//{window.location.host}/api/auth/oidc/callback
        </p>
        <span style={{ fontSize: 12, color: '#666' }}>
          Register this callback URL with your Identity Provider (e.g. Authentik, Keycloak).
        </span>
      </div>

      <Form form={form} layout="vertical" onFinish={onFinish}>
        <Form.Item
          label="Issuer (OIDC Endpoint)"
          name="issuer"
          rules={[{ required: true, message: 'Issuer URL is required' }]}
          help="The URL of your OpenID Connect provider (e.g. https://authentik.company.com/application/o/helpdesk/)"
        >
          <Input placeholder="https://..." />
        </Form.Item>

        <Form.Item label="Client ID" name="client_id" rules={[{ required: true }]}>
          <Input />
        </Form.Item>

        <Form.Item label="Client Secret" name="client_secret" rules={[{ required: true }]}>
          <Input.Password placeholder="Opaque secret from provider" />
        </Form.Item>

        <Form.Item label="Scopes" name="scopes" initialValue="openid,profile,email">
          <Input placeholder="Space or comma separated (e.g. openid profile email)" />
        </Form.Item>

        <Form.Item label="Group Claim Name" name="group_claim_name" initialValue="groups">
          <Input placeholder="Token claim containing user groups (default: groups)" />
        </Form.Item>

        <Form.Item label="Username Claim" name="username_claim" initialValue="preferred_username">
          <Input placeholder="Token claim to use as username (default: preferred_username)" />
        </Form.Item>

        <Form.Item label="Admin Group" name="admin_group">
          <Input placeholder="Name of the group that grants Admin access" />
        </Form.Item>

        <Form.Item name="auto_onboard" valuePropName="checked">
          <Checkbox>Automatic Onboarding (allow new users to login)</Checkbox>
        </Form.Item>

        <Form.Item label="Redirect URL Override" name="redirect_url">
          <Input placeholder="Leave empty to auto-detect" />
        </Form.Item>

        <h3>Role Mapping</h3>
        <p>Map specific groups (from the provider) to internal roles.</p>
        <Form.List name="value_to_roles">
          {(fields, { add, remove }) => (
            <>
              {fields.map(({ key, name, ...restField }) => (
                <Space key={key} align="baseline" style={{ marginBottom: 8, display: 'flex' }}>
                  <Form.Item
                    {...restField}
                    name={[name, 'value']}
                    rules={[{ required: true, message: 'Group is required' }]}
                  >
                    <Input placeholder="Group Name (e.g. 'Support')" />
                  </Form.Item>
                  <Form.Item
                    {...restField}
                    name={[name, 'roles']}
                    rules={[{ required: true, message: 'Role is required' }]}
                  >
                    <Input placeholder="Roles (e.g. 'agent,manager')" />
                  </Form.Item>
                  <Button onClick={() => remove(name)} danger>Remove</Button>
                </Space>
              ))}
              <Form.Item>
                <Button type="dashed" onClick={() => add()} block>
                  + Add Role Mapping
                </Button>
              </Form.Item>
            </>
          )}
        </Form.List>

        <Form.Item>
          <Button type="primary" htmlType="submit" loading={save.isPending}>
            Save Configuration
          </Button>
        </Form.Item>
      </Form>
    </>
  );
}
