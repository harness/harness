import React from 'react'
import { Layout, Text, Icon, FontVariation } from '@harness/uicore'
// import { PopoverInteractionKind } from '@blueprintjs/core'
// import { ButtonRoleProps } from 'utils/Utils'
import { CodeIcon, GitInfoProps } from 'utils/GitUtils'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import css from './RepositoryHeader.module.scss'

export function RepositoryHeader({ repoMetadata }: Pick<GitInfoProps, 'repoMetadata'>) {
  return (
    <RepositoryPageHeader
      repoMetadata={repoMetadata}
      title={
        <Layout.Horizontal spacing="small" className={css.name}>
          <Icon name={CodeIcon.Repo} size={20} />
          <Text
            inline
            className={css.repoDropdown}
            // rightIcon="main-chevron-down"
            // rightIconProps={{
            //   size: 14,
            //   color: Color.GREY_500,
            //   margin: { left: 'xsmall' }
            // }}
            // tooltip={<Container padding="xlarge">TBD...</Container>}
            // tooltipProps={{
            //   interactionKind: PopoverInteractionKind.CLICK,
            //   targetClassName: css.targetClassName,
            //   minimal: true
            // }}
            font={{ variation: FontVariation.H4 }}
            // {...ButtonRoleProps}
          >
            {repoMetadata.uid}
          </Text>
        </Layout.Horizontal>
      }
      dataTooltipId="repositoryTitle"
    />
  )
}
