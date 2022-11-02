import React from 'react'
import { Link } from 'react-router-dom'
import cx from 'classnames'
import { Container, Layout, Button, ButtonVariation, Utils } from '@harness/uicore'
import css from './CommitActions.module.scss'

interface CommitActionButtonProps {
  sha: string
  href: string
  enableCopy?: boolean
}

export function CommitActions({ sha, href, enableCopy }: CommitActionButtonProps): JSX.Element {
  return (
    <Container className={css.container}>
      <Layout.Horizontal className={cx(css.layout, !enableCopy ? css.noCopy : '')}>
        <Link to={href}>{sha.substring(0, 6)}</Link>
        {enableCopy && (
          <Button
            id={css.commitCopyButton}
            variation={ButtonVariation.ICON}
            icon="copy-alt"
            iconProps={{ size: 14 }}
            onClick={() => Utils.copy(sha)}
          />
        )}
      </Layout.Horizontal>
    </Container>
  )
}
