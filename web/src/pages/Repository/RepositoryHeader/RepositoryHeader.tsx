import React from 'react'
import { Container, Layout, Text, Color, Icon } from '@harness/uicore'
import { Link, useParams } from 'react-router-dom'
import { PopoverInteractionKind } from '@blueprintjs/core'
import { useStrings } from 'framework/strings'
import { ButtonRoleProps } from 'utils/Utils'
import { GitIcon } from 'utils/GitUtils'
import type { RepositoryDTO } from 'types/SCMTypes'
import { useAppContext } from 'AppContext'
import type { SCMPathProps } from 'RouteDefinitions'
import css from './RepositoryHeader.module.scss'

interface RepositoryHeaderProps {
  repoMetadata: RepositoryDTO
}

export function RepositoryHeader({ repoMetadata }: RepositoryHeaderProps): JSX.Element {
  const { getString } = useStrings()
  const { space: spaceFromPath = '' } = useParams<SCMPathProps>()
  const { space = spaceFromPath || '', routes } = useAppContext()

  return (
    <Container className={css.header}>
      <Container>
        <Layout.Horizontal spacing="small" className={css.breadcrumb}>
          {/* <Link to="">SCM_Project</Link>
          <Icon name="main-chevron-right" size={10} color={Color.GREY_500} /> */}
          <Link to={routes.toSCMRepositoriesListing({ space })}>{getString('repositories')}</Link>
          <Icon name="main-chevron-right" size={10} color={Color.GREY_500} />
          <Link to={routes.toSCMRepository({ repoPath: repoMetadata.path })}>{repoMetadata.name}</Link>
        </Layout.Horizontal>
        <Container padding={{ top: 'medium', bottom: 'medium' }}>
          <Text
            inline
            className={css.repoDropdown}
            icon={GitIcon.REPOSITORY}
            iconProps={{
              size: 14,
              color: Color.GREY_500,
              margin: { right: 'small' }
            }}
            rightIcon="main-chevron-down"
            rightIconProps={{
              size: 14,
              color: Color.GREY_500,
              margin: { left: 'xsmall' }
            }}
            tooltip={<Container padding="xlarge">TBD...</Container>}
            tooltipProps={{
              interactionKind: PopoverInteractionKind.CLICK,
              targetClassName: css.targetClassName,
              minimal: true
            }}
            {...ButtonRoleProps}>
            {repoMetadata.name}
          </Text>
        </Container>
      </Container>
    </Container>
  )
}
