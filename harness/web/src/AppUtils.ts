/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { routes } from 'RouteDefinitions'

/**
 * Handle 401 error from API.
 *
 * This function is called to handle 401 (unauthorized) API calls under standalone mode.
 * In embedded mode, the parent app is responsible to pass its handler down.
 *
 * Mostly, the implementation of this function is just a redirection to signin page.
 */
export function handle401() {
  const signinUrl = window.location.origin + routes.toSignIn()

  if (window.location.href !== signinUrl) {
    window.location.href = signinUrl
  }
}

/**
 * Build Restful React Request Options.
 *
 * This function is an extension to configure HTTP headers before passing to Restful
 * React to make an API call. Customizations to fulfill the micro-frontend backend
 * service happen here.
 *
 * @param token API token
 * @returns Restful React RequestInit object.
 */
export function buildRestfulReactRequestOptions(token?: string): Partial<RequestInit> {
  const headers: RequestInit['headers'] = {}

  if (token?.length) {
    headers.Authorization = `Bearer ${token}`
  }

  return { headers }
}
