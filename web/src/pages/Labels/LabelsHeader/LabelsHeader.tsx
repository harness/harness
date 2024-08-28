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
import React, { useState } from 'react'
import { Container, Layout, FlexExpander, ButtonVariation, Button, Checkbox } from '@harnessio/uicore'
import { Render } from 'react-jsx-match'
import { useStrings } from 'framework/strings'
import { CodeIcon } from 'utils/GitUtils'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import type { LabelsPageScope } from 'utils/Utils'
import type { RepoRepositoryOutput } from 'services/code'
import css from './LabelsHeader.module.scss'

const LabelsHeader = ({
  loading,
  onSearchTermChanged,
  showParentScopeFilter,
  inheritLabels,
  setInheritLabels,
  openLabelCreateModal
}: LabelsHeaderProps) => {
  const [searchTerm, setSearchTerm] = useState('')
  const { getString } = useStrings()

  //ToDo: check space permissions as well in case of spaces

  return (
    <Container className={css.main} padding={{ top: 'medium', right: 'xlarge', left: 'xlarge', bottom: 'medium' }}>
      <Layout.Horizontal spacing="medium">
        <Button
          variation={ButtonVariation.PRIMARY}
          text={getString('labels.newLabel')}
          icon={CodeIcon.Add}
          onClick={openLabelCreateModal}
        />
        <Render when={showParentScopeFilter}>
          <Checkbox
            className={css.scopeCheckbox}
            label={getString('labels.showLabelsScope')}
            data-testid={`INCLUDE_ORG_RESOURCES`}
            checked={inheritLabels}
            onChange={event => {
              setInheritLabels(event.currentTarget.checked)
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

export default LabelsHeader

interface LabelsHeaderProps {
  loading?: boolean
  activeTab?: string
  onSearchTermChanged: (searchTerm: string) => void
  repoMetadata?: RepoRepositoryOutput
  spaceRef?: string
  currentPageScope: LabelsPageScope
  showParentScopeFilter: boolean
  setInheritLabels: (value: boolean) => void
  inheritLabels: boolean
  openLabelCreateModal: () => void
}
