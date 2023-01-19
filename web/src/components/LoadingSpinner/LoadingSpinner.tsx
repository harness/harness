import React from 'react'
import cx from 'classnames'
import { Container, Layout, Icon, Text, Color } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import css from './LoadingSpinner.module.scss'

interface LoadingSpinnerProps {
  visible: boolean | null | undefined
  withBorder?: boolean
}

export const LoadingSpinner: React.FC<LoadingSpinnerProps> = ({ visible, withBorder }) => {
  const { standalone } = useAppContext()
  const { getString } = useStrings()

  return visible ? (
    <Container className={cx(css.main, { [css.standalone]: standalone, [css.withBorder]: withBorder })}>
      <Layout.Vertical spacing="medium" className={css.layout}>
        <Icon name="steps-spinner" size={32} color={Color.GREY_600} />
        <Text font={{ size: 'medium', align: 'center' }} color={Color.GREY_600} className={css.text}>
          {getString('pageLoading')}
        </Text>
      </Layout.Vertical>
    </Container>
  ) : null
}
