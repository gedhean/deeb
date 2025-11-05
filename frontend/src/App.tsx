import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { AuthProvider } from '@/contexts/AuthContext';
import { ProtectedRoute } from '@/components/protected-route';
import { SetupCheck } from '@/components/setup-check';
import { Home } from '@/pages/Home';
import { HealthCheckPage } from '@/pages/HealthCheckPage';
import { LoginPage } from '@/pages/LoginPage';
import { RegisterPage } from '@/pages/RegisterPage';
import { DashboardPage } from '@/pages/DashboardPage';
import { ProjectsPage } from '@/pages/ProjectsPage';
import { EnvironmentsPage } from '@/pages/EnvironmentsPage';
import { PoliciesPage } from '@/pages/PoliciesPage';
import { DatabaseInstancesPage } from '@/pages/DatabaseInstancesPage';
import { DatabasesPage } from '@/pages/DatabasesPage';
import { UsersPage } from '@/pages/UsersPage';
import { ChangePlanCreatePage } from '@/pages/ChangePlanCreatePage';
import { ChangePlansPage } from '@/pages/ChangePlansPage';
import { NotFoundPage } from '@/pages/NotFoundPage';
import './App.css';

function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <SetupCheck>
          <Routes>
            <Route path="/" element={<Home />} />
            <Route path="/health" element={<HealthCheckPage />} />
            <Route path="/login" element={<LoginPage />} />
            <Route path="/register" element={<RegisterPage />} />
            {/* For testing - dashboard without auth */}
            <Route path="/dash" element={<DashboardPage />} />
            <Route
              path="/dashboard"
              element={
                <ProtectedRoute>
                  <DashboardPage />
                </ProtectedRoute>
              }
            />
            <Route
              path="/projects"
              element={
                <ProtectedRoute>
                  <ProjectsPage />
                </ProtectedRoute>
              }
            />
            <Route
              path="/environments"
              element={
                <ProtectedRoute>
                  <EnvironmentsPage />
                </ProtectedRoute>
              }
            />
            <Route
              path="/policies"
              element={
                <ProtectedRoute>
                  <PoliciesPage />
                </ProtectedRoute>
              }
            />
            <Route
              path="/database-instances"
              element={
                <ProtectedRoute>
                  <DatabaseInstancesPage />
                </ProtectedRoute>
              }
            />
            <Route
              path="/databases"
              element={
                <ProtectedRoute>
                  <DatabasesPage />
                </ProtectedRoute>
              }
            />
            <Route
              path="/users"
              element={
                <ProtectedRoute>
                  <UsersPage />
                </ProtectedRoute>
              }
            />
            <Route
              path="/change-plans"
              element={
                <ProtectedRoute>
                  <ChangePlansPage />
                </ProtectedRoute>
              }
            />
            <Route
              path="/change-plans/create"
              element={
                <ProtectedRoute>
                  <ChangePlanCreatePage />
                </ProtectedRoute>
              }
            />
            {/* 404 Not Found - must be last */}
            <Route path="*" element={<NotFoundPage />} />
          </Routes>
        </SetupCheck>
      </BrowserRouter>
    </AuthProvider>
  );
}

export default App;
