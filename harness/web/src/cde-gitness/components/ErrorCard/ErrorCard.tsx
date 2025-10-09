import React from 'react'
import { Button, ButtonVariation, Container, Layout, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { Icon } from '@harnessio/icons'
import { useStrings } from 'framework/strings'
import { docLink } from 'cde-gitness/constants'
import css from './ErrorCard.module.scss'

export const ErrorCard = ({
  message,
  triggerGitspace,
  loading,
  viewLogs
}: {
  message: string
  triggerGitspace: () => void
  loading: boolean
  viewLogs: () => void
}) => {
  const { getString } = useStrings()
  const iconProps = { size: 14, color: Color.PRIMARY_7 }
  return (
    <Layout.Horizontal className={css.cardContainer}>
      <Layout.Vertical className={css.errorCard} spacing="medium">
        <Layout.Horizontal>
          <Container className={css.iconContainer}>
            <Icon name="warning-sign" color="red500" size={20} />
          </Container>
          <Text className={css.errorTitle}>{getString('cde.errorCard.title')}</Text>
        </Layout.Horizontal>
        <Layout.Vertical className={css.errorBody} spacing="medium">
          <Container className={css.preClass}>
            <Text className={css.errorMessage}>{message}</Text>
          </Container>
          <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'space-between' }}>
            <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'start' }}>
              <Button
                variation={ButtonVariation.SECONDARY}
                text={getString('cde.errorCard.viewLog')}
                onClick={viewLogs}
                className={css.logBtn}
              />
              <Text
                iconProps={iconProps}
                color={Color.PRIMARY_7}
                icon={loading ? 'loading' : 'repeat'}
                className={css.linkButton}
                onClick={triggerGitspace}>
                {getString('cde.errorCard.retryGitspace')}
              </Text>
              <Text
                iconProps={iconProps}
                color={Color.PRIMARY_7}
                icon="document"
                onClick={e => {
                  e.preventDefault()
                  e.stopPropagation()
                  window.open(docLink, '_blank')
                }}
                rightIcon="share"
                rightIconProps={iconProps}
                className={css.linkButton}>
                {getString('cde.errorCard.learnMore')}
              </Text>
            </Layout.Horizontal>
            <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'end' }} spacing="small">
              <Text className={css.resolveText}>{getString('cde.errorCard.unabletoResolve')}</Text>
              <Text
                color={Color.PRIMARY_7}
                className={css.contactBtn}
                onClick={() => {
                  const element: HTMLElement = window?.document?.querySelector(
                    '#sidenav-footer [__type="SIDENAV_LINK"]'
                  ) as HTMLElement
                  element?.click()
                }}>
                {getString('cde.errorCard.contactUs')}
              </Text>
            </Layout.Horizontal>
          </Layout.Horizontal>
        </Layout.Vertical>
      </Layout.Vertical>
    </Layout.Horizontal>
  )
}
