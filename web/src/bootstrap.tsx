import React from 'react'
import ReactDOM from 'react-dom'
import App from './App'
import './App.scss'

// This flag is used in services/config.ts to customize API path when app is run
// in multiple modes (standalone vs. embedded).
// Also being used in when generating proper URLs inside the app.
window.STRIP_SCM_PREFIX = true

ReactDOM.render(
  <App
    standalone
    accountId="default"
    apiToken="default"
    baseRoutePath="/account/default/settings/governance"
    hooks={{}}
    components={{}}
  />,
  document.getElementById('react-root')
)
