import React, { useEffect, useState } from 'react'
import { Container, Layout, Button, ButtonVariation, Utils, Text, Color } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import css from './CloneButtonTooltip.module.scss'

interface CloneButtonTooltipProps {
  httpsURL: string
}

export function CloneButtonTooltip({ httpsURL }: CloneButtonTooltipProps) {
  const { getString } = useStrings()
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    let timeoutId: number
    if (copied) {
      timeoutId = window.setTimeout(() => setCopied(false), 2500)
    }
    return () => {
      clearTimeout(timeoutId)
    }
  }, [copied])

  return (
    <Container className={css.container} padding="xlarge">
      <Layout.Vertical spacing="small">
        <Text className={css.label}>{getString('cloneHTTPS')}</Text>
        <Container>
          <Layout.Horizontal className={css.layout}>
            <Text className={css.url}>{httpsURL}</Text>
            <Button
              id={css.cloneCopyButton}
              variation={ButtonVariation.ICON}
              icon={copied ? 'tick' : 'copy-alt'}
              iconProps={{ size: 14, color: copied ? Color.GREEN_500 : undefined }}
              onClick={() => {
                setCopied(true)
                Utils.copy(httpsURL)
              }}
            />
          </Layout.Horizontal>
        </Container>
      </Layout.Vertical>
    </Container>
  )
}
