/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import { useHistory } from 'react-router-dom'
import React, { useState } from 'react'
import { Container, Layout, FlexExpander, ButtonVariation, Button } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import { CodeIcon, GitInfoProps, SettingTypeMode } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { permissionProps } from 'utils/Utils'
import css from './BranchProtectionHeader.module.scss'
const BranchProtectionHeader = ({
  repoMetadata,
  loading,
  onSearchTermChanged,
  activeTab
}: BranchProtectionHeaderProps) => {
  const history = useHistory()
  const [searchTerm, setSearchTerm] = useState('')
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const { hooks, standalone } = useAppContext()

  const space = useGetSpaceParam()

  const permPushResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY',
        resourceIdentifier: repoMetadata?.identifier as string
      },
      permissions: ['code_repo_edit']
    },
    [space]
  )

  return (
    <Container className={css.main} padding="xlarge">
      <Layout.Horizontal spacing="medium">
        <Button
          variation={ButtonVariation.PRIMARY}
          text={getString('branchProtection.newRule')}
          icon={CodeIcon.Add}
          onClick={() =>
            history.push(
              routes.toCODESettings({
                repoPath: repoMetadata?.path as string,
                settingSection: activeTab,
                settingSectionMode: SettingTypeMode.NEW
              })
            )
          }
          {...permissionProps(permPushResult, standalone)}
        />
        <FlexExpander />
        <SearchInputWithSpinner
          spinnerPosition="right"
          loading={loading}
          query={searchTerm}
          setQuery={value => {
            setSearchTerm(value)
            onSearchTermChanged(value)
          }}
        />
      </Layout.Horizontal>
    </Container>
  )
}

export default BranchProtectionHeader

interface BranchProtectionHeaderProps extends Pick<GitInfoProps, 'repoMetadata'> {
  loading?: boolean
  activeTab?: string
  onSearchTermChanged: (searchTerm: string) => void
}
