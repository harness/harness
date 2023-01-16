import { useEffect } from 'react'

export function useDisableCodeMainLinks(disabled: boolean) {
  useEffect(() => {
    if (disabled) {
      const nav = document.querySelectorAll('[data-code-repo-section]')

      nav?.forEach(element => {
        if (element.getAttribute('data-code-repo-section') !== 'files') {
          element.setAttribute('aria-disabled', 'true')
        }
      })

      return () => {
        nav?.forEach(element => {
          element.removeAttribute('aria-disabled')
        })
      }
    }
  }, [disabled])
}
