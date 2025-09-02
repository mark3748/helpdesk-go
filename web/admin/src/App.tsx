import React from 'react'
import UserRoles from './components/UserRoles'
import TestConnection from './components/TestConnection'

export default function App() {
  return (
    <div>
      <h1>Admin</h1>
      <UserRoles />
      <TestConnection />
    </div>
  )
}
