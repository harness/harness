import React from 'react'
import { Container, Layout, Text, Color, Icon, FontVariation } from '@harness/uicore'
import { Link } from 'react-router-dom'
import { PopoverInteractionKind } from '@blueprintjs/core'
import { useStrings } from 'framework/strings'
import { ButtonRoleProps } from 'utils/Utils'
import { GitIcon } from 'utils/GitUtils'
import type { TypesRepository } from 'services/scm'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useAppContext } from 'AppContext'
import css from './RepositoryHeader.module.scss'

interface RepositoryHeaderProps {
  repoMetadata: TypesRepository
}

export function RepositoryHeader({ repoMetadata }: RepositoryHeaderProps) {
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const { routes } = useAppContext()

  return (
    <Container className={css.header}>
      <Container>
        <Layout.Horizontal spacing="small" className={css.breadcrumb}>
          <Link to={routes.toSCMRepositoriesListing({ space })}>{getString('repositories')}</Link>
          <Icon name="main-chevron-right" size={10} color={Color.GREY_500} />
          <Link to={routes.toSCMRepository({ repoPath: repoMetadata.path as string })}>{repoMetadata.name}</Link>
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
            font={{ variation: FontVariation.H4 }}
            {...ButtonRoleProps}>
            {repoMetadata.name}
          </Text>
        </Container>
      </Container>
    </Container>
  )
}
