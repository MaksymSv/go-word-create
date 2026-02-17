import React from 'react'
import { useTheme } from '../context/ThemeContext'

export const ThemeToggle: React.FC = () => {
  const { theme, toggleTheme } = useTheme()

  return (
    <button
      className="hero-btn hero-btn-ghost"
      onClick={toggleTheme}
      aria-label="Toggle theme"
    >
      {theme === 'dark' ? (
        <svg
          className="icon"
          viewBox="0 0 24 24"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
          aria-hidden="true"
        >
          <path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z" fill="currentColor" />
        </svg>
      ) : (
        <svg
          className="icon"
          viewBox="0 0 24 24"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
          aria-hidden="true"
        >
          <path d="M6.76 4.84l-1.8-1.79L3.17 5.84l1.79 1.79 1.8-1.79zM1 13h3v-2H1v2zm10 9h2v-3h-2v3zm7.04-2.46l1.79 1.79 1.79-1.79-1.79-1.8-1.79 1.8zM20 13v-2h3v2h-3zM12 6a6 6 0 100 12 6 6 0 000-12zm0-4h-2v3h2V2zM4.22 19.78l1.79-1.79-1.79-1.79L2.43 18l1.79 1.78zM18.36 5.64l1.79-1.79-1.79-1.79-1.79 1.79 1.79 1.79z" fill="currentColor" />
        </svg>
      )}
    </button>
  )
}
