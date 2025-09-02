import React from 'react'
import { Tabs } from 'antd'
import UserRoles from './components/UserRoles'
import TestConnection from './components/TestConnection'
import StorageSettings from '../../internal/StorageSettings'
import OIDCSettings from '../../internal/OIDCSettings'
import MailSettings from '../../internal/MailSettings'

export default function App() {
  return (
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
  )
}
