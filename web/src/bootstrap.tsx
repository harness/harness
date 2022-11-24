import React from 'react'
import ReactDOM from 'react-dom'
import { routes } from 'RouteDefinitions'
import App from './App'
import './bootstrap.scss'

// This flag is used in services/config.ts to customize API path when app is run
// in multiple modes (standalone vs. embedded).
// Also being used in when generating proper URLs inside the app.
window.STRIP_CODE_PREFIX = true

ReactDOM.render(<App standalone routes={routes} hooks={{}} />, document.getElementById('react-root'))
