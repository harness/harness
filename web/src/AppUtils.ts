/**
 * Handle 401 error from API.
 *
 * This function is called to handle 401 (unauthorized) API calls under standalone mode.
 * In embedded mode, the parent app is responsible to pass its handler down.
 *
 * Mostly, the implementation of this function is just a redirection to signin page.
 */
export function handle401(): void {
  // eslint-disable-next-line no-console
  console.error('TODO: Handle 401 error...')
}

/**
 * Build Restful React Request Options.
 *
 * This function is an extension to configure HTTP headers before passing to Restful
 * React to make an API call. Customizations to fulfill the micro-frontend backend
 * service happen here.
 *
 * @param token API token
 * @returns Resful React RequestInit object.
 */
export function buildResfulReactRequestOptions(token?: string): Partial<RequestInit> {
  const headers: RequestInit['headers'] = {}

  if (token?.length) {
    headers.Authorization = `Bearer ${token}`
  }

  return { headers }
}
