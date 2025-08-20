import React from 'react'
import { compact } from 'lodash-es'
import { Layout, Text } from '@harnessio/uicore'
import { getRelativeSpaceRef, getScopeData, getScopeFromParams, ScopeEnum } from 'utils/Utils'
import { LabelTitle } from 'components/Label/Label'

export const ScopeBadge = ({
  standalone,
  currentScope,
  path
}: {
  standalone: boolean
  currentScope: ScopeEnum
  path?: string
}) => {
  const [accountId, repoOrgIdentifier, repoProjectIdentifier] = path?.split('/').slice(0, -1) || []
  const repoScope = getScopeFromParams(
    { accountId, orgIdentifier: repoOrgIdentifier, projectIdentifier: repoProjectIdentifier },
    standalone
  )

  if ([ScopeEnum.REPO_SCOPE, ScopeEnum.PROJECT_SCOPE, ScopeEnum.SPACE_SCOPE].includes(currentScope)) {
    return null
  }

  const { scopeColor, scopeName } = getScopeData(
    compact([accountId, repoOrgIdentifier, repoProjectIdentifier]).join('/'),
    repoScope,
    standalone
  )

  // Show the relative space reference depending on the current scope
  const relativeSpaceRef = getRelativeSpaceRef(currentScope, repoScope, repoOrgIdentifier, repoProjectIdentifier)

  return (
    <Layout.Vertical spacing="xsmall">
      <LabelTitle name={scopeName} scope={repoScope} label_color={scopeColor} isScopeName />
      {relativeSpaceRef && (
        <Text font={{ size: 'small' }} lineClamp={1}>
          {relativeSpaceRef}
        </Text>
      )}
    </Layout.Vertical>
  )
}
