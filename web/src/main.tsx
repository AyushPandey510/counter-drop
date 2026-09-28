import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { registerSW } from 'virtual:pwa-register'
import App from './App'
import './lib/install' // start listening for the install prompt before any page renders
import './index.css'

// The service worker caches the app shell only; API calls and uploads always go to the network.
registerSW({ immediate: true })

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
