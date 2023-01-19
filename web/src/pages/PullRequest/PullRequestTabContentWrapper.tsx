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
}) => (
  <Container className={className} padding="xlarge" {...(!!loading || !!error ? { flex: true } : {})}>
    <LoadingSpinner visible={loading} />
    {error && <PageError message={getErrorMessage(error)} onClick={onRetry} />}
    {!loading && !error && children}
  </Container>
)
