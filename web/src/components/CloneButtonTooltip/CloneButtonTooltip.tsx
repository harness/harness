import React, { useState } from 'react'
import { Button, ButtonVariation, Container, Layout, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import { CopyButton } from 'components/CopyButton/CopyButton'
import { CodeIcon } from 'utils/GitUtils'
import CloneCredentialDialog from 'components/CloneCredentialDialog/CloneCredentialDialog'
import css from './CloneButtonTooltip.module.scss'

interface CloneButtonTooltipProps {
  httpsURL: string
}

export function CloneButtonTooltip({ httpsURL }: CloneButtonTooltipProps) {
  const { getString } = useStrings()
  const [flag, setFlag] = useState(false)

  return (
    <Container className={css.container} padding="xlarge">
      <Layout.Vertical spacing="small">
        <Text font={{ variation: FontVariation.H4 }}>{getString('cloneHTTPS')}</Text>
        <Text
          icon={'code-info'}
          iconProps={{ size: 16 }}
          color={Color.GREY_700}
          font={{ variation: FontVariation.BODY2_SEMI, size: 'small' }}>
          {getString('generateCloneText')}
        </Text>

        <Container>
          <Layout.Horizontal className={css.layout}>
            <Text className={css.url}>{httpsURL}</Text>

            <CopyButton content={httpsURL} id={css.cloneCopyButton} icon={CodeIcon.Copy} iconProps={{ size: 14 }} />
          </Layout.Horizontal>
        </Container>
        <Button
          onClick={() => {
            setFlag(true)
          }}
          variation={ButtonVariation.SECONDARY}>
          {getString('generateCloneCred')}
        </Button>
      </Layout.Vertical>
      <CloneCredentialDialog flag={flag} setFlag={setFlag} />
    </Container>
  )
}