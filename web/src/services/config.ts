export const getConfig = (str: string): string => {
  // 'code/api/v1' -> 'api/v1'       (standalone)
  //               -> 'code/api/v1'  (embedded inside Harness platform)
  if (window.STRIP_CODE_PREFIX) {
    str = str.replace(/^code/, 'api/v1')
  }

  return window.apiUrl ? `${window.apiUrl}/${str}` : `${window.harnessNameSpace || ''}/${str}`
}
