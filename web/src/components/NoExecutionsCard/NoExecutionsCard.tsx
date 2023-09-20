import React from 'react'
import { Container, NoDataCard } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import noExecutionImage from '../../pages/RepositoriesListing/no-repo.svg'
import css from './NoExecutionsCard.module.scss'

interface NoResultCardProps {
  showWhen?: () => boolean
}

export const NoExecutionsCard: React.FC<NoResultCardProps> = ({ showWhen = () => true }) => {
  const { getString } = useStrings()

  if (!showWhen()) {
    return null
  }

  return (
    <Container className={css.main}>
      <NoDataCard image={noExecutionImage} message={getString('executions.noData')} />
    </Container>
  )
}
