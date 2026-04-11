import { Navigate, Outlet, Route, Routes } from 'react-router-dom'
import { AppShell } from './components/AppShell'
import { TenantGate } from './components/TenantGate'
import { LoginPage } from './pages/LoginPage'
import { RegisterPage } from './pages/RegisterPage'
import { OverviewPage } from './pages/OverviewPage'
import { PlatformAdminPage } from './pages/PlatformAdminPage'
import { StudioConfiguratorPage } from './pages/StudioConfiguratorPage'
import { AppellatePage } from './pages/AppellatePage'
import { RegistriesPage } from './pages/RegistriesPage'
import { AuditorPage } from './pages/AuditorPage'
import { CitizenPage } from './pages/CitizenPage'
import { OperatorPage } from './pages/OperatorPage'
import { AccountAdminPage } from './pages/AccountAdminPage'
import { AccountSetupPage } from './pages/AccountSetupPage'
import { ServiceAdminPage } from './pages/ServiceAdminPage'
import { WelcomePage } from './pages/WelcomePage'
import { AuthProvider, useAuth } from './state/auth'
import { ConsoleScopeProvider } from './state/consoleScope'
import { SelectionProvider } from './state/selection'

function App() {
  return (
    <AuthProvider>
      <SelectionProvider>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route path="/" element={<RequireAuth />}>
            <Route path="welcome" element={<WelcomePage />} />
            <Route element={<TenantGate />}>
              <Route
                element={
                  <ConsoleScopeProvider>
                    <AppShell />
                  </ConsoleScopeProvider>
                }
              >
                <Route index element={<OverviewPage />} />
                <Route path="platform" element={<PlatformAdminPage />} />
                <Route path="account" element={<AccountAdminPage />} />
                <Route path="account/setup" element={<Navigate to="/account/setup/rulesets" replace />} />
                <Route path="account/setup/:tab" element={<AccountSetupPage />} />
                <Route path="service" element={<ServiceAdminPage />} />
                <Route path="regulator" element={<StudioConfiguratorPage />} />
                <Route path="registries" element={<RegistriesPage />} />
                <Route path="appellate" element={<AppellatePage />} />
                <Route path="operator" element={<OperatorPage />} />
                <Route path="audit" element={<AuditorPage />} />
                <Route path="citizen" element={<CitizenPage />} />
              </Route>
            </Route>
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </SelectionProvider>
    </AuthProvider>
  )
}

export default App

function RequireAuth() {
  const auth = useAuth()
  if (!auth.ready) return <div style={{ padding: 16 }}>Loading…</div>
  if (!auth.isAuthenticated) return <Navigate to="/login" replace />
  return <Outlet />
}
