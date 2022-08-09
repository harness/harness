import { useLocalStorage } from 'hooks/useLocalStorage'

const API_TOKEN_KEY = 'HARNESS_STANDALONE_APP_API_TOKEN'

/**
 * Get and Set API token to use in Restful React calls.
 *
 * This function is called to inject token to Restful React calls only when
 * application is run under standalone mode. In embedded mode, token is passed
 * from parent app.
 *
 * @param initialToken initial API token.
 * @returns [token, setToken].
 */
export function useAPIToken(initialToken = ''): [string, React.Dispatch<React.SetStateAction<string>>] {
  return useLocalStorage(API_TOKEN_KEY, initialToken)
}
