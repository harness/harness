import React from 'react'
import cx from 'classnames'
import { Button, ButtonSize, Layout } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import css from './PrevNextPagination.module.scss'

interface PrevNextPaginationProps {
  onPrev?: () => void
  onNext?: () => void
}

export function PrevNextPagination({ onPrev, onNext }: PrevNextPaginationProps) {
  const { getString } = useStrings()

  return (
    <Layout.Horizontal>
      <Button
        text={getString('prev')}
        icon="arrow-left"
        size={ButtonSize.SMALL}
        className={cx(css.roundedButton, css.buttonLeft)}
        iconProps={{ size: 12 }}
        onClick={onPrev}
        disabled={!onPrev}
      />
      <Button
        text={getString('next')}
        rightIcon="arrow-right"
        size={ButtonSize.SMALL}
        className={cx(css.roundedButton, css.buttonRight)}
        iconProps={{ size: 12 }}
        onClick={onNext}
        disabled={!onNext}
      />
    </Layout.Horizontal>
  )
}
