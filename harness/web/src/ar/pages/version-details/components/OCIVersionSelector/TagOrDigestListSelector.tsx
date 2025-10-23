/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useEffect, useState } from 'react'
import { Icon } from '@harnessio/icons'
import { Container, Layout, Tab, Tabs } from '@harnessio/uicore'

import { OCIVersionType } from '@ar/common/types'
import { useStrings } from '@ar/frameworks/strings'
import SearchInput from '@ar/components/ManageMetadata/SearchInput'
import TabsContainer from '@ar/components/TabsContainer/TabsContainer'

import type { OCIVersionValue } from './type'
import TagListSelector from './TagListSelector'
import DigestListSelector from './DigestListSelector'

import css from './OCIVersionSelector.module.scss'

interface TagOrDigestListSelectorProps {
  onChange: (value: OCIVersionValue) => void
  value: OCIVersionValue
}

function TagOrDigestListSelectorProps(props: TagOrDigestListSelectorProps) {
  const { onChange, value } = props
  const [tab, setTab] = useState('')
  const [query, setQuery] = useState('')
  const { getString } = useStrings()

  const handleSelect = (val: OCIVersionValue) => {
    onChange(val)
  }

  useEffect(() => {
    if (value.tag) {
      setTab(OCIVersionType.TAG)
    } else {
      setTab(OCIVersionType.DIGEST)
    }
  }, [value])

  return (
    <Layout.Vertical className={css.combinedListSelector}>
      <Container className={css.searchInputContainer}>
        <SearchInput
          autoFocus
          leftElement={<Icon margin="small" name="search" size={12} />}
          className={css.searchInput}
          value={query}
          onChange={q => setQuery(q)}
        />
      </Container>
      <TabsContainer className={css.tabsContainer}>
        <Tabs id="versionSelector" selectedTabId={tab} onChange={(newTab: string) => setTab(newTab)}>
          <Tab
            id={OCIVersionType.TAG}
            title={getString('versionDetails.OCIVersionSelectorTab.tag')}
            panel={<TagListSelector query={query} value={value} onSelect={handleSelect} />}
          />
          <Tab
            id={OCIVersionType.DIGEST}
            title={getString('versionDetails.OCIVersionSelectorTab.digest')}
            panel={<DigestListSelector query={query} value={value} onSelect={handleSelect} />}
          />
        </Tabs>
      </TabsContainer>
    </Layout.Vertical>
  )
}

export default TagOrDigestListSelectorProps
