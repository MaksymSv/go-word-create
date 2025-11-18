import React from 'react'
import './Views.css'

export const Dashboard: React.FC = () => (
  <div className="hero-card">
    <h2>Welcome</h2>
    <p>Use the left menu to select a view. This SPA is built with React and talks to the Go backend to generate reports.</p>
  </div>
)

export const Sprints: React.FC = () => (
  <div className="hero-card">
    <h2>Sprints</h2>
    <p>Fetch issues by sprint and generate reports.</p>
    <div className="sprint-form">
      <label htmlFor="sprintInput">Sprint name</label>
      <input id="sprintInput" type="text" placeholder="e.g., Sprint 16" />
      <button className="hero-btn">Show</button>
    </div>
    <div id="sprintResult" className="sprint-result"></div>
  </div>
)

export const Month: React.FC = () => {
  const [month, setMonth] = React.useState('')
  const [result, setResult] = React.useState('')

  const handleShow = () => {
    if (!month) {
      setResult('Please select a month.')
      return
    }
    setResult(`Showing issues in <strong>${month}</strong>`)
  }

  return (
    <div className="hero-card">
      <h2>Month</h2>
      <p>
        Select month to fetch issues that entered <strong>In Progress</strong> during the month.
      </p>
      <div className="month-form">
        <label htmlFor="monthInput">Choose month</label>
        <input
          id="monthInput"
          type="month"
          value={month}
          onChange={(e) => setMonth(e.target.value)}
        />
        <button className="hero-btn" onClick={handleShow}>
          Show
        </button>
      </div>
      {result && (
        <div className="month-result" aria-live="polite">
          <p dangerouslySetInnerHTML={{ __html: result }} />
        </div>
      )}
    </div>
  )
}

export const Settings: React.FC = () => (
  <div className="hero-card">
    <h2>Settings</h2>
    <p>Configure Jira credentials and preferences.</p>
  </div>
)
