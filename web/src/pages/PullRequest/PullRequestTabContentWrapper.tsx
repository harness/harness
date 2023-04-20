import React from 'react'
import { Container, PageError } from '@harness/uicore'
import { getErrorMessage } from 'utils/Utils'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'

interface PullRequestTabContentWrapperProps {
  className?: string
  loading?: boolean
  error?: Unknown
  onRetry: () => void
}

export const PullRequestTabContentWrapper: React.FC<PullRequestTabContentWrapperProps> = ({
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
