import React from 'react'
import { Container, PageError } from '@harnessio/uicore'
import { getErrorMessage } from 'utils/Utils'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'

interface TabContentWrapperProps {
  className?: string
  loading?: boolean
  error?: Unknown
  onRetry: () => void
}

export const TabContentWrapper: React.FC<TabContentWrapperProps> = ({
  className,
  loading,
  error,
  onRetry,
  children
}) => {
  return (
    <Container className={className} padding="xlarge">
      <LoadingSpinner visible={loading} withBorder={true} />
      {error && <PageError message={getErrorMessage(error)} onClick={onRetry} />}
      {!error && children}
    </Container>
  )
}
