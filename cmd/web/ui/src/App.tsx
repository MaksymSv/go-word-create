import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { ThemeProvider } from './context/ThemeContext'
import { Layout } from './components/Layout'
import { Dashboard, Sprints, Month, Settings } from './components/Views'
import './App.css'

function App() {
  return (
    <ThemeProvider>
      <BrowserRouter basename="/">
        <Routes>
          <Route element={<Layout />}>
            <Route index element={<Dashboard />} />
            <Route path="sprints" element={<Sprints />} />
            <Route path="month" element={<Month />} />
            <Route path="settings" element={<Settings />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </ThemeProvider>
  )
}

export default App
