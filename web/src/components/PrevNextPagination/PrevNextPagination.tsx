import React from 'react'
import cx from 'classnames'
import { Button, ButtonSize, Container, Layout } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import css from './PrevNextPagination.module.scss'

interface PrevNextPaginationProps {
  onPrev?: false | (() => void)
  onNext?: false | (() => void)
  skipLayout?: boolean
}

export function PrevNextPagination({ onPrev, onNext, skipLayout }: PrevNextPaginationProps) {
  const { getString } = useStrings()

  return (
    <Container className={skipLayout ? undefined : css.main}>
      <Layout.Horizontal>
        <Button
          text={getString('prev')}
          icon="arrow-left"
          size={ButtonSize.SMALL}
          className={cx(css.roundedButton, css.buttonLeft)}
          iconProps={{ size: 12 }}
          onClick={onPrev ? onPrev : undefined}
          disabled={!onPrev}
        />
        <Button
          text={getString('next')}
          rightIcon="arrow-right"
          size={ButtonSize.SMALL}
          className={cx(css.roundedButton, css.buttonRight)}
          iconProps={{ size: 12 }}
          onClick={onNext ? onNext : undefined}
          disabled={!onNext}
        />
      </Layout.Horizontal>
    </Container>
  )
}
