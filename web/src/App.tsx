import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { I18nProvider } from '@/lib/i18n'
import Home from '@/customer/Home'
import DropPage from '@/customer/DropPage'
import TicketPage from '@/customer/TicketPage'
import { RequireStaff, ShopAuthProvider } from '@/shop/auth'
import LoginPage from '@/shop/LoginPage'
import BoardPage from '@/shop/BoardPage'
import SettingsPage from '@/shop/SettingsPage'
import QrPage from '@/shop/QrPage'

// Customer routes (/, /s/:slug, /t/:jobId) are public; /shop/* needs a staff session.
export default function App() {
  return (
    <BrowserRouter>
      <I18nProvider>
        <ShopAuthProvider>
          <Routes>
            <Route path="/" element={<Home />} />
            <Route path="/s/:slug" element={<DropPage />} />
            <Route path="/t/:jobId" element={<TicketPage />} />
            <Route path="/shop/login" element={<LoginPage />} />
            <Route path="/shop" element={<RequireStaff><BoardPage /></RequireStaff>} />
            <Route path="/shop/qr" element={<RequireStaff><QrPage /></RequireStaff>} />
            <Route path="/shop/settings" element={<RequireStaff owner><SettingsPage /></RequireStaff>} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </ShopAuthProvider>
      </I18nProvider>
    </BrowserRouter>
  )
}
