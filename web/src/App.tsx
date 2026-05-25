import { Routes, Route, Navigate } from 'react-router-dom'
import AppLayout from './components/AppLayout'
import InstancesPage from './pages/InstancesPage'
import ComparePage from './pages/ComparePage'
import PlayerPage from './pages/PlayerPage'

export default function App() {
  return (
    <Routes>
      <Route element={<AppLayout />}>
        <Route path="/" element={<InstancesPage />} />
        <Route path="/instances/:instanceId" element={<InstancesPage />} />
        <Route path="/compare/:id1/:id2" element={<ComparePage />} />
        <Route path="/players/:userId" element={<PlayerPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  )
}
