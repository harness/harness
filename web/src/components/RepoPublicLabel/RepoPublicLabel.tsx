import React from 'react'
import { Text, TextProps } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import css from './RepoPublicLabel.module.scss'

export const RepoPublicLabel: React.FC<{ isPublic?: boolean; margin?: TextProps['margin'] }> = ({
  isPublic,
  margin
}) => {
  const { getString } = useStrings()

  return (
    <Text inline className={css.label} margin={margin}>
      {getString(isPublic ? 'public' : 'private')}
    </Text>
  )
}
