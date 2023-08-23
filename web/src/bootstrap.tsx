import React from 'react'
import ReactDOM from 'react-dom'
import { noop } from 'lodash-es'
import { routes } from 'RouteDefinitions'
import { defaultCurrentUser } from 'AppContext'
import App from './App'
import './bootstrap.scss'

// This flag is used in services/config.ts to customize API path when app is run
// in multiple modes (standalone vs. embedded).
// Also being used in when generating proper URLs inside the app.
// In standalone mode, we don't need `code/` prefix in API URIs.
window.STRIP_CODE_PREFIX = true

ReactDOM.render(
  <App
    standalone
    routes={routes}
    hooks={{
      usePermissionTranslate: noop
    }}
    currentUser={defaultCurrentUser}
    currentUserProfileURL=""
    routingId=""
  />,
  document.getElementById('react-root')
)
