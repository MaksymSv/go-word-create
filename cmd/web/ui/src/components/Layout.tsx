import React from 'react'
import { Outlet } from 'react-router-dom'
import { Navigation } from './Navigation'
import { ThemeToggle } from './ThemeToggle'
import './Layout.css'

export const Layout: React.FC = () => {
  return (
    <div className="hero-app">
      <Navigation />
      <main className="hero-main">
        <header className="hero-topbar">
          <div className="hero-topbar-title">Dashboard</div>
          <div className="hero-topbar-actions">
            <ThemeToggle />
          </div>
        </header>
        <section className="hero-container">
          <Outlet />
        </section>
      </main>
    </div>
  )
}
