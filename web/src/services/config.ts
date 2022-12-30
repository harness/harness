export const getConfig = (str: string): string => {
  // NOTE: Replace /^code\// with your service prefixes when running in standalone mode
  // I.e: 'code/api/v1' -> 'api/v1'     (standalone)
  //                   -> 'code/api/v1' (embedded inside Harness platform)
  if (window.STRIP_CODE_PREFIX) {
    str = str.replace(/^code\//, '')
  }

  return window.apiUrl ? `${window.apiUrl}/${str}` : `${window.harnessNameSpace || ''}/${str}`
}
