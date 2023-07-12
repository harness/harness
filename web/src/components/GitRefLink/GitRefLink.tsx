import React from 'react'
import { Link } from 'react-router-dom'
import { Color, Layout } from '@harness/uicore'
import { CopyButton } from 'components/CopyButton/CopyButton'
import { useStrings } from 'framework/strings'
import { CodeIcon } from 'utils/GitUtils'
import css from './GitRefLink.module.scss'

export const GitRefLink: React.FC<{ text: string; url: string; showCopy: boolean }> = ({
  text,
  url,
  showCopy = true
}) => {
  const { getString } = useStrings()

  return (
    <Layout.Horizontal className={css.link} inline>
      <Link className={css.linkText} to={url}>
        {text}
      </Link>
      {showCopy ? (
        <CopyButton
          className={css.copyContainer}
          content={text}
          tooltip={getString('copyBranch')}
          icon={CodeIcon.Copy}
          color={Color.PRIMARY_7}
          iconProps={{ size: 14, color: Color.PRIMARY_7 }}
          background={Color.PRIMARY_1}
        />
      ) : null}
    </Layout.Horizontal>
  )
}
