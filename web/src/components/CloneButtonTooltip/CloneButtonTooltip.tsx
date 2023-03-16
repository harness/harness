import React from 'react'
import { Container, Layout, Text } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import { CopyButton } from 'components/CopyButton/CopyButton'
import { CodeIcon } from 'utils/GitUtils'
import css from './CloneButtonTooltip.module.scss'

interface CloneButtonTooltipProps {
  httpsURL: string
}

export function CloneButtonTooltip({ httpsURL }: CloneButtonTooltipProps) {
  const { getString } = useStrings()

  return (
    <Container className={css.container} padding="xlarge">
      <Layout.Vertical spacing="small">
        <Text className={css.label}>{getString('cloneHTTPS')}</Text>
        <Container>
          <Layout.Horizontal className={css.layout}>
            <Text className={css.url}>{httpsURL}</Text>
            <CopyButton content={httpsURL} id={css.cloneCopyButton} icon={CodeIcon.Copy} iconProps={{ size: 14 }} />
          </Layout.Horizontal>
        </Container>
      </Layout.Vertical>
    </Container>
  )
}
