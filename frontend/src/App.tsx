import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { AuthProvider } from '@/contexts/AuthContext';
import { ProtectedRoute } from '@/components/protected-route';
import { PublicRoute } from '@/components/public-route';
import { SetupCheck } from '@/components/setup-check';
import { ErrorBoundary } from '@/components/ErrorBoundary';
import { Toaster } from '@/components/ui/sonner';
import { Home } from '@/pages/Home';
import { HealthCheckPage } from '@/pages/HealthCheckPage';
import { LoginPage } from '@/pages/LoginPage';
import { AuthCallbackPage } from '@/pages/AuthCallbackPage';
import { RegisterPage } from '@/pages/RegisterPage';
import { DashboardPage } from '@/pages/DashboardPage';
import { ProjectsPage } from '@/pages/ProjectsPage';
import { ProjectDetailPage } from '@/pages/ProjectDetailPage';
import { EnvironmentsPage } from '@/pages/EnvironmentsPage';
import { PoliciesPage } from '@/pages/PoliciesPage';
import { DatabaseInstancesPage } from '@/pages/DatabaseInstancesPage';
import { DatabaseInstanceDetailPage } from '@/pages/DatabaseInstanceDetailPage';
import { DatabasesPage } from '@/pages/DatabasesPage';
import { DatabaseDetailPage } from '@/pages/DatabaseDetailPage';
import { UsersPage } from '@/pages/UsersPage';
import { UserDetailPage } from '@/pages/UserDetailPage';
import { ProfilePage } from '@/pages/ProfilePage';
import { ChangePlanCreatePage } from '@/pages/ChangePlanCreatePage';
import { ChangePlansPage } from '@/pages/ChangePlansPage';
import { ChangePlanDetailPage } from '@/pages/ChangePlanDetailPage';
import { ChangelogDetailPage } from '@/pages/ChangelogDetailPage';
import { AuditLogPage } from '@/pages/AuditLogPage';
import { NotFoundPage } from '@/pages/NotFoundPage';
import './App.css';

function App() {
  return (
    <ErrorBoundary>
      <BrowserRouter>
        <AuthProvider>
          <SetupCheck>
            <Routes>
              <Route path="/" element={<Home />} />
              <Route path="/health" element={<HealthCheckPage />} />
              <Route
                path="/login"
                element={
                  <PublicRoute>
                    <LoginPage />
                  </PublicRoute>
                }
              />
              {/*
                Public route: the auth cookie is already set by the time the
                OAuth roundtrip lands here, so it must NOT be wrapped in
                PublicRoute (which would bounce the now-authenticated user to
                /dashboard and defeat the returnTo forwarding).
              */}
              <Route path="/auth/callback" element={<AuthCallbackPage />} />
              <Route
                path="/register"
                element={
                  <PublicRoute>
                    <RegisterPage />
                  </PublicRoute>
                }
              />
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
                path="/projects/:id"
                element={
                  <ProtectedRoute>
                    <ProjectDetailPage />
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
                path="/database-instances/:id"
                element={
                  <ProtectedRoute>
                    <DatabaseInstanceDetailPage />
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
                path="/databases/:id"
                element={
                  <ProtectedRoute>
                    <DatabaseDetailPage />
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
                path="/users/:id"
                element={
                  <ProtectedRoute>
                    <UserDetailPage />
                  </ProtectedRoute>
                }
              />
              <Route
                path="/profile"
                element={
                  <ProtectedRoute>
                    <ProfilePage />
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
              <Route
                path="/change-plans/:id"
                element={
                  <ProtectedRoute>
                    <ChangePlanDetailPage />
                  </ProtectedRoute>
                }
              />
              <Route
                path="/changelog/:id"
                element={
                  <ProtectedRoute>
                    <ChangelogDetailPage />
                  </ProtectedRoute>
                }
              />
              <Route
                path="/audit"
                element={
                  <ProtectedRoute>
                    <AuditLogPage />
                  </ProtectedRoute>
                }
              />
              {/* 404 Not Found - must be last */}
              <Route path="*" element={<NotFoundPage />} />
            </Routes>
          </SetupCheck>
          <Toaster />
        </AuthProvider>
      </BrowserRouter>
    </ErrorBoundary>
  );
}

export default App;
