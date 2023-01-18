import React from 'react'
import { Button, Container, ButtonVariation, NoDataCard, IconName } from '@harness/uicore'
import { noop } from 'lodash-es'
import { CodeIcon } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import emptyStateImage from 'images/empty-state.svg'
import css from './NoResultCard.module.scss'

interface NoResultCardProps {
  showWhen: () => boolean
  forSearch: boolean
  message?: string
  buttonText?: string
  buttonIcon?: IconName
  onButtonClick?: () => void
}

export const NoResultCard: React.FC<NoResultCardProps> = ({
  showWhen,
  forSearch,
  message,
  buttonText = '',
  buttonIcon = CodeIcon.Add,
  onButtonClick = noop
}) => {
  const { getString } = useStrings()

  if (!showWhen()) {
    return null
  }

  return (
    <Container className={css.main}>
      <NoDataCard
        image={emptyStateImage}
        messageTitle={forSearch ? getString('noResultTitle') : undefined}
        message={forSearch ? getString('noResultMessage') : message || getString('noResultMessage')}
        button={
          forSearch ? undefined : (
            <Button
              variation={ButtonVariation.PRIMARY}
              text={buttonText}
              icon={buttonIcon as IconName}
              onClick={onButtonClick}
            />
          )
        }
      />
    </Container>
  )
}
