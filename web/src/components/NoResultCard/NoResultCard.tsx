import React from 'react'
import { Button, Container, ButtonVariation, NoDataCard, IconName } from '@harness/uicore'
import { noop } from 'lodash-es'
import { CodeIcon } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import { Images } from 'images'
import css from './NoResultCard.module.scss'

interface NoResultCardProps {
  showWhen?: () => boolean
  forSearch: boolean
  title?: string
  message?: string
  emptySearchMessage?: string
  buttonText?: string
  buttonIcon?: IconName
  onButtonClick?: () => void
}

export const NoResultCard: React.FC<NoResultCardProps> = ({
  showWhen = () => true,
  forSearch,
  title,
  message,
  emptySearchMessage,
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
        image={Images.EmptyState}
        messageTitle={forSearch ? title || getString('noResultTitle') : undefined}
        message={
          forSearch ? emptySearchMessage || getString('noResultMessage') : message || getString('noResultMessage')
        }
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
