import React from 'react'
import { Link, useLocation } from 'react-router-dom'
import './Navigation.css'

export const Navigation: React.FC = () => {
  const location = useLocation()

  const isActive = (path: string) => location.pathname === path

  return (
    <aside className="hero-sidebar">
      <div className="hero-sidebar-header">Go Word Create</div>
      <nav className="hero-menu">
        <Link
          to="/"
          className={`hero-menu-item ${isActive('/') ? 'active' : ''}`}
        >
          Dashboard
        </Link>
        <Link
          to="/sprints"
          className={`hero-menu-item ${isActive('/sprints') ? 'active' : ''}`}
        >
          Sprints
        </Link>
        <Link
          to="/month"
          className={`hero-menu-item ${isActive('/month') ? 'active' : ''}`}
        >
          Month
        </Link>
        <Link
          to="/settings"
          className={`hero-menu-item ${isActive('/settings') ? 'active' : ''}`}
        >
          Settings
        </Link>
      </nav>
    </aside>
  )
}
