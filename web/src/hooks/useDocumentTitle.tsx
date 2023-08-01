import { useEffect } from 'react'
import { useStrings } from 'framework/strings'

export function useDocumentTitle(title: string) {
  const { getString } = useStrings()

  useEffect(() => {
    const _title = document.title

    document.title = `${title} - ${getString('gitness')}`

    return () => {
      document.title = _title
    }
  }, [title, getString])
}
