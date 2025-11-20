import React from 'react'
import type { Error } from '@harnessio/react-har-service-client'
import { getErrorInfoFromErrorObject, PageError, PageSpinner } from '@harnessio/uicore'

interface PageContentProps {
  loading: boolean
  error: Error | null
  refetch: () => void
}

function PageContent(props: React.PropsWithChildren<PageContentProps>) {
  const { children, loading, error, refetch } = props
  switch (true) {
    case loading:
      return <PageSpinner />
    case !!error:
      return <PageError message={getErrorInfoFromErrorObject(error as Error)} onClick={refetch} />
    default:
      return <>{children}</>
  }
}

export default PageContent
