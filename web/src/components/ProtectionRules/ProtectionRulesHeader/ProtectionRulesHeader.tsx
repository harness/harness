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
import { ButtonVariation, Container, Checkbox, FlexExpander, Layout, SplitButton, Text } from '@harnessio/uicore'
import { Render } from 'react-jsx-match'
import { Icon } from '@harnessio/icons'
import { Menu } from '@blueprintjs/core'
import { Color, FontVariation } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import { CodeIcon, DashboardFilter, GitInfoProps, ProtectionRulesType, SettingTypeMode } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { getEditPermissionRequestFromIdentifier, PageBrowserProps, permissionProps, ScopeEnum } from 'utils/Utils'
import ToggleTabsBtn from 'components/ToggleTabs/ToggleTabsBtn'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import css from './ProtectionRulesHeader.module.scss'

interface ProtectionRulesHeaderProps extends Partial<Pick<GitInfoProps, 'repoMetadata'>> {
  loading?: boolean
  activeTab?: string
  currentPageScope: ScopeEnum
  inheritRules: boolean
  ruleTypeFilter: ProtectionRulesType
  onSearchTermChanged: (searchTerm: string) => void
}

const ProtectionRulesHeader = ({
  repoMetadata,
  loading,
  onSearchTermChanged,
  activeTab,
  currentPageScope,
  inheritRules,
  ruleTypeFilter
}: ProtectionRulesHeaderProps) => {
  const history = useHistory()
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const { hooks, standalone } = useAppContext()
  const space = useGetSpaceParam()
  const { updateQueryParams } = useUpdateQueryParams<PageBrowserProps | { type: ProtectionRulesType }>()
  const [searchTerm, setSearchTerm] = useState('')
  const [ruleType, setRuleType] = useState(ProtectionRulesType.BRANCH)

  const permPushResult = hooks?.usePermissionTranslate(getEditPermissionRequestFromIdentifier(space, repoMetadata), [
    space,
    repoMetadata
  ])

  const ruleTypeFilters = [
    { label: getString('all'), value: DashboardFilter.ALL },
    { label: getString('branch'), value: ProtectionRulesType.BRANCH },
    { label: getString('tag'), value: ProtectionRulesType.TAG }
  ]

  return (
    <Container className={css.main} padding="xlarge">
      <Layout.Horizontal spacing="medium">
        <SplitButton
          variation={ButtonVariation.PRIMARY}
          text={getString('protectionRules.newRule', { ruleType })}
          icon={CodeIcon.Add}
          onClick={() =>
            repoMetadata
              ? history.push(
                  routes.toCODESettings({
                    repoPath: repoMetadata?.path as string,
                    settingSection: activeTab,
                    settingSectionMode: `${SettingTypeMode.NEW}?type=${ruleType}`
                  })
                )
              : standalone
              ? history.push(
                  routes.toCODESpaceSettings({
                    space,
                    settingSection: activeTab,
                    settingSectionMode: `${SettingTypeMode.NEW}?type=${ruleType}`
                  })
                )
              : history.push(
                  routes.toCODEManageRepositories({
                    space,
                    settingSection: activeTab,
                    settingSectionMode: `${SettingTypeMode.NEW}?type=${ruleType}`
                  })
                )
          }
          {...permissionProps(permPushResult, standalone)}>
          <Container>
            {Object.values(ProtectionRulesType)
              .filter(type => type !== ProtectionRulesType.PUSH)
              .map(type => {
                return (
                  <Menu.Item
                    key={type}
                    className={css.menuItem}
                    text={
                      <Layout.Horizontal>
                        <Icon name={ruleType === type ? CodeIcon.Tick : CodeIcon.Blank} />
                        <Text
                          padding={{ left: 'xsmall' }}
                          color={Color.BLACK}
                          font={{ variation: FontVariation.BODY2_SEMI }}>
                          {getString('protectionRules.newRule', { ruleType: type })}
                        </Text>
                      </Layout.Horizontal>
                    }
                    onClick={() => setRuleType(type)}
                  />
                )
              })}
          </Container>
        </SplitButton>
        <ToggleTabsBtn
          wrapperClassName={css.tabsContainer}
          ctnWrapperClassName={css.stateCtn}
          currentTab={ruleTypeFilter ?? DashboardFilter.ALL}
          tabsList={ruleTypeFilters}
          onTabChange={newTab => {
            updateQueryParams({ type: newTab as ProtectionRulesType })
          }}
        />
        <Render when={![ScopeEnum.ACCOUNT_SCOPE, ScopeEnum.SPACE_SCOPE].includes(currentPageScope)}>
          <Checkbox
            className={css.scopeCheckbox}
            label={getString('protectionRules.showRulesScope')}
            checked={inheritRules}
            onChange={event => {
              updateQueryParams({ inherit: event.currentTarget.checked.toString() })
            }}
          />
        </Render>
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

export default ProtectionRulesHeader
