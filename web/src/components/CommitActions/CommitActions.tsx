import React, { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import cx from 'classnames'
import { Container, Layout, Button, ButtonVariation, Utils, Text, Color } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import css from './CommitActions.module.scss'
import { GitIcon } from 'utils/GitUtils'

interface CommitActionButtonProps {
  sha: string
  href: string
  enableCopy?: boolean
}

export function CommitActions({ sha, href, enableCopy }: CommitActionButtonProps) {
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
    <Container className={css.container}>
      <Layout.Horizontal className={cx(css.layout, !enableCopy ? css.noCopy : '')}>
        <Link to={href}>
          <Text tooltip={getString('viewCommitDetails')} inline>
            {sha.substring(0, 6)}
          </Text>
        </Link>
        {enableCopy && (
          <Button
            id={css.commitCopyButton}
            variation={ButtonVariation.ICON}
            icon={copied ? 'tick' : GitIcon.CodeCopy}
            iconProps={{ size: 14, color: copied ? Color.GREEN_500 : undefined }}
            onClick={() => {
              setCopied(true)
              Utils.copy(sha)
            }}
            tooltip={getString('copyCommitSHA')}
          />
        )}
      </Layout.Horizontal>
    </Container>
  )
}
