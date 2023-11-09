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

import React, { useCallback, useMemo, useState } from 'react'
import { noop } from 'lodash-es'
import {
  Container,
  Layout,
  Button,
  ButtonSize,
  FlexExpander,
  ButtonVariation,
  Text,
  Utils,
  Dialog
} from '@harnessio/uicore'
import cx from 'classnames'
import { Icon } from '@harnessio/icons'
import { useHotkeys } from 'react-hotkeys-hook'
import { LongArrowDownLeft, Search } from 'iconoir-react'
import { Color } from '@harnessio/design-system'
import { Breadcrumbs, IBreadcrumbProps } from '@blueprintjs/core'
import { Link, useHistory } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { CloneButtonTooltip } from 'components/CloneButtonTooltip/CloneButtonTooltip'
import { CodeIcon, GitInfoProps, isDir, isGitRev, isRefATag } from 'utils/GitUtils'
import { BranchTagSelect } from 'components/BranchTagSelect/BranchTagSelect'
import { useCreateBranchModal } from 'components/CreateBranchModal/CreateBranchModal'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { ButtonRoleProps, permissionProps } from 'utils/Utils'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import svg from './search-background.svg'
import css from './ContentHeader.module.scss'

export function ContentHeader({
  repoMetadata,
  gitRef = repoMetadata.default_branch as string,
  resourcePath,
  resourceContent
}: Pick<GitInfoProps, 'repoMetadata' | 'gitRef' | 'resourcePath' | 'resourceContent'>) {
  const { getString } = useStrings()
  const { routes, standalone, hooks } = useAppContext()
  const history = useHistory()
  const _isDir = isDir(resourceContent)
  const space = useGetSpaceParam()
  const [showSearchModal, setShowSearchModal] = useState(false)
  const [searchSampleQueryIndex, setSearchSampleQueryIndex] = useState<number>(0)
  const [search, setSearch] = useState('')
  const performSearch = useCallback(
    (q: string) => {
      history.push({
        pathname: routes.toCODESearch({
          repoPath: repoMetadata.path as string
        }),
        search: `q=${q}`
      })
    },
    [history, repoMetadata.path, routes]
  )
  const onSearch = useCallback(() => {
    if (search?.trim()) {
      performSearch(search)
    } else if (searchSampleQueryIndex > 0 && searchSampleQueryIndex <= searchSampleQueries.length) {
      performSearch(searchSampleQueries[searchSampleQueryIndex - 1])
    }
  }, [performSearch, search, searchSampleQueryIndex])

  useHotkeys(
    'ctrl+k',
    () => {
      if (!showSearchModal) {
        setShowSearchModal(true)
      }
    },
    [showSearchModal]
  )

  const permPushResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY'
      },
      permissions: ['code_repo_push']
    },
    [space]
  )
  const openCreateNewBranchModal = useCreateBranchModal({
    repoMetadata,
    onSuccess: branchInfo => {
      history.push(
        routes.toCODERepository({
          repoPath: repoMetadata.path as string,
          gitRef: branchInfo.name
        })
      )
    },
    suggestedSourceBranch: gitRef,
    showSuccessMessage: true
  })
  const breadcrumbs = useMemo(() => {
    return resourcePath.split('/').map((_path, index, paths) => {
      const pathAtIndex = paths.slice(0, index + 1).join('/')
      const href = routes.toCODERepository({
        repoPath: repoMetadata.path as string,
        gitRef,
        resourcePath: pathAtIndex
      })

      return { href, text: _path }
    })
  }, [resourcePath, gitRef, repoMetadata.path, routes])

  return (
    <Container className={cx(css.main, { [css.mainContainer]: !isDir(resourceContent) })}>
      <Layout.Horizontal className={isDir(resourceContent) ? '' : css.mainBorder} spacing="medium">
        <BranchTagSelect
          repoMetadata={repoMetadata}
          gitRef={gitRef}
          onSelect={ref => {
            history.push(
              routes.toCODERepository({
                repoPath: repoMetadata.path as string,
                gitRef: ref,
                resourcePath
              })
            )
          }}
          onCreateBranch={openCreateNewBranchModal}
        />
        <Container style={{ maxWidth: 'calc(100vw - 750px)' }}>
          <Layout.Horizontal spacing="small">
            <Link
              id="repository-ref-root"
              className={css.refRoot}
              to={routes.toCODERepository({ repoPath: repoMetadata.path as string, gitRef })}>
              <Icon name={CodeIcon.Folder} />
            </Link>
            <Text className={css.rootSlash} color={Color.GREY_900}>
              /
            </Text>
            <Breadcrumbs
              items={breadcrumbs}
              breadcrumbRenderer={({ text, href }: IBreadcrumbProps) => {
                return (
                  <Link to={href as string}>
                    <Text color={Color.GREY_900}>{text}</Text>
                  </Link>
                )
              }}
            />
          </Layout.Horizontal>
        </Container>
        <FlexExpander />
        {_isDir && (
          <>
            <Button
              text={getString('clone')}
              variation={ButtonVariation.SECONDARY}
              icon={CodeIcon.Clone}
              className={css.btnColorFix}
              tooltip={<CloneButtonTooltip httpsURL={repoMetadata.git_url as string} />}
              tooltipProps={{
                interactionKind: 'click',
                minimal: true,
                position: 'bottom-right'
              }}
            />
            <Button
              text={getString('newFile')}
              icon={CodeIcon.Add}
              variation={ButtonVariation.PRIMARY}
              disabled={isRefATag(gitRef) || isGitRev(gitRef)}
              tooltip={isRefATag(gitRef) ? getString('newFileNotAllowed') : undefined}
              tooltipProps={{ isDark: true }}
              onClick={() => {
                history.push(
                  routes.toCODEFileEdit({
                    repoPath: repoMetadata.path as string,
                    resourcePath,
                    gitRef: gitRef || (repoMetadata.default_branch as string)
                  })
                )
              }}
              {...permissionProps(permPushResult, standalone)}
            />
          </>
        )}
      </Layout.Horizontal>
      {!standalone && false && (
        <Container
          className={css.searchBox}
          {...ButtonRoleProps}
          onClick={() => {
            setShowSearchModal(true)
          }}>
          <SearchInputWithSpinner
            readOnly
            placeholder={getString('codeSearch') + ` (ctrl-k)`}
            query={''}
            setQuery={noop}
          />
          {<img src={svg} width={95} height={22} />}

          {showSearchModal && (
            <Container onClick={Utils.stopEvent}>
              <Dialog
                className={css.searchModal}
                backdropClassName={css.backdrop}
                portalClassName={css.portal}
                isOpen={true}
                enforceFocus={false}
                onClose={() => {
                  setShowSearchModal(false)
                }}>
                <Container>
                  <Layout.Vertical spacing="large">
                    <Container>
                      <Layout.Horizontal className={css.layout}>
                        <Container className={css.searchContainer}>
                          <Search className={css.searchIcon} width={18} height={18} color="var(--ai-purple-600)" />
                          <SearchInputWithSpinner
                            placeholder={getString('codeSearchModal')}
                            spinnerPosition="right"
                            query={search}
                            type="text"
                            setQuery={q => {
                              if (q?.trim()) {
                                setSearchSampleQueryIndex(0)
                              }
                              setSearch(q)
                            }}
                            height={40}
                            onSearch={onSearch}
                            onKeyDown={e => {
                              if (!search?.trim()) {
                                switch (e.key) {
                                  case 'ArrowDown':
                                    setSearchSampleQueryIndex(index => {
                                      return index + 1 > searchSampleQueries.length ? 1 : index + 1
                                    })
                                    break
                                  case 'ArrowUp':
                                    setSearchSampleQueryIndex(index => {
                                      return index - 1 > 0 ? index - 1 : searchSampleQueries.length
                                    })
                                    break
                                }
                              }
                            }}
                          />
                          {!search && <img src={svg} width={132} height={28} />}
                        </Container>
                        <Button
                          variation={ButtonVariation.AI}
                          text={getString('search')}
                          size={ButtonSize.MEDIUM}
                          onClick={onSearch}
                        />
                      </Layout.Horizontal>
                    </Container>
                    <Text className={css.sectionHeader}>{getString('searchHeader')}</Text>
                    <Container>
                      <Layout.Vertical spacing="medium">
                        {searchSampleQueries.map((sampleQuery, index) => (
                          <Text
                            key={index}
                            className={cx(css.sampleQuery, { [css.selected]: index === searchSampleQueryIndex - 1 })}
                            {...ButtonRoleProps}
                            onClick={() => {
                              performSearch(sampleQuery)
                            }}>
                            {sampleQuery}
                            <LongArrowDownLeft color="" />
                          </Text>
                        ))}
                      </Layout.Vertical>
                    </Container>
                  </Layout.Vertical>
                </Container>
              </Dialog>
            </Container>
          )}
        </Container>
      )}
    </Container>
  )
}

// These sample queries are in English only - No need to do i18n for them
const searchSampleQueries = [
  `Where is the code that handles authentication?`,
  `Where is the application entry point?`,
  `Where do we configure the logger?`
]
