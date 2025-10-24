import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { Home } from '@/pages/Home'
import { HealthCheckPage } from '@/pages/HealthCheckPage'
import './App.css'

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/health" element={<HealthCheckPage />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
