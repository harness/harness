import React from 'react'
import ReactDOM from 'react-dom'
import App from './App'

// This flag is used in services/config.ts to customize API path when app is run
// in multiple modes (standalone vs. embedded).
// Also being used in when generating proper URLs inside the app.
window.APP_RUN_IN_STANDALONE_MODE = true

ReactDOM.render(<App standalone hooks={{}} components={{}} />, document.getElementById('react-root'))
