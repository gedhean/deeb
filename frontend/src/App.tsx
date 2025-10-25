import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { AuthProvider } from '@/contexts/AuthContext';
import { Home } from '@/pages/Home';
import { HealthCheckPage } from '@/pages/HealthCheckPage';
import './App.css';

function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/health" element={<HealthCheckPage />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  );
}

export default App;
