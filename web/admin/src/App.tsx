import { Tabs } from 'antd'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import UserRoles from './components/UserRoles'
import TestConnection from './components/TestConnection'
import StorageSettings from '../../internal/src/components/admin/StorageSettings'
import OIDCSettings from '../../internal/src/components/admin/OIDCSettings'
import MailSettings from '../../internal/src/components/admin/MailSettings'

const queryClient = new QueryClient()

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <div>
        <h1>Admin</h1>
        <Tabs
          items={[
            { key: 'storage', label: 'Storage', children: <StorageSettings /> },
            { key: 'oidc', label: 'OIDC', children: <OIDCSettings /> },
            { key: 'mail', label: 'SMTP/IMAP', children: <MailSettings /> },
          ]}
        />
        <UserRoles />
        <TestConnection />
      </div>
    </QueryClientProvider>
  )
}
