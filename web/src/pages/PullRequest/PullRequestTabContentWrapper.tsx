import React from 'react'
import { Container, PageError } from '@harness/uicore'
import cx from 'classnames'
import { ContainerSpinner } from 'components/ContainerSpinner/ContainerSpinner'
import { getErrorMessage } from 'utils/Utils'
import css from './PullRequest.module.scss'

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
  <Container
    className={cx(css.tabContentContainer, className)}
    padding="xlarge"
    {...(!!loading || !!error ? { flex: true } : {})}>
    {loading && <ContainerSpinner />}
    {error && <PageError message={getErrorMessage(error)} onClick={onRetry} />}
    {!loading && !error && children}
  </Container>
)
