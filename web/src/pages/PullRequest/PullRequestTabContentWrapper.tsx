import React from 'react'
import { Container, PageError } from '@harness/uicore'
import { ContainerSpinner } from 'components/ContainerSpinner/ContainerSpinner'
import { getErrorMessage } from 'utils/Utils'

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
    {loading && <ContainerSpinner />}
    {error && <PageError message={getErrorMessage(error)} onClick={onRetry} />}
    {!loading && !error && children}
  </Container>
)
