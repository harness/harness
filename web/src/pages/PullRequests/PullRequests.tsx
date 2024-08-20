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

import React, { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Container,
  PageBody,
  Text,
  TableV2,
  Layout,
  StringSubstitute,
  FlexExpander,
  Utils,
  stringSubstitute
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
import type { CellProps, Column } from 'react-table'
import { Case, Match, Render, Truthy } from 'react-jsx-match'
import { defaultTo, isEmpty, noop } from 'lodash-es'
import { makeDiffRefs, PullRequestFilterOption } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import {
  voidFn,
  getErrorMessage,
  LIST_FETCHING_LIMIT,
  permissionProps,
  PageBrowserProps,
  ColorName,
  LabelFilterObj,
  LabelFilterType
} from 'utils/Utils'
import { usePageIndex } from 'hooks/usePageIndex'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import { useQueryParams } from 'hooks/useQueryParams'
import type { TypesPullReq, RepoRepositoryOutput, TypesLabelPullReqAssignmentInfo } from 'services/code'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import { GitRefLink } from 'components/GitRefLink/GitRefLink'
import { PullRequestStateLabel } from 'components/PullRequestStateLabel/PullRequestStateLabel'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import useSpaceSSE from 'hooks/useSpaceSSE'
import { TimePopoverWithLocal } from 'utils/timePopoverLocal/TimePopoverWithLocal'
import { Label } from 'components/Label/Label'
import { PullRequestsContentHeader } from './PullRequestsContentHeader/PullRequestsContentHeader'
import css from './PullRequests.module.scss'

const SSE_EVENTS = ['pullreq_updated']

export default function PullRequests() {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes, hooks, standalone } = useAppContext()
  const { CODE_PULLREQ_LABELS: isLabelEnabled } = hooks?.useFeatureFlags()
  const [searchTerm, setSearchTerm] = useState<string | undefined>()
  const browserParams = useQueryParams<PageBrowserProps>()
  const [filter, setFilter] = useState(browserParams?.state || (PullRequestFilterOption.OPEN as string))
  const [authorFilter, setAuthorFilter] = useState<string>()
  const [labelFilter, setLabelFilter] = useState<LabelFilterObj[]>([])
  const space = useGetSpaceParam()
  const { updateQueryParams, replaceQueryParams } = useUpdateQueryParams()
  const pageInit = browserParams.page ? parseInt(browserParams.page) : 1
  const [page, setPage] = usePageIndex(pageInit)
  useEffect(() => {
    const params = {
      ...browserParams,
      ...(page > 1 && { page: page.toString() }),
      ...(filter && { state: filter })
    }
    updateQueryParams(params, undefined, true)

    if (page <= 1) {
      const updateParams = { ...params }
      delete updateParams.page
      replaceQueryParams(updateParams, undefined, true)
    }
  }, [page, filter]) // eslint-disable-line react-hooks/exhaustive-deps
  const { repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()
  const {
    data,
    error: prError,
    loading: prLoading,
    refetch: refetchPrs,
    response
  } = useGet<TypesPullReq[]>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/pullreq`,
    queryParams: {
      limit: String(LIST_FETCHING_LIMIT),
      page: browserParams.page,
      sort: filter == PullRequestFilterOption.MERGED ? 'merged' : 'number',
      order: 'desc',
      query: searchTerm,
      state: browserParams.state ? browserParams.state : filter == PullRequestFilterOption.ALL ? '' : filter,
      ...(authorFilter && { created_by: Number(authorFilter) }),

      ...(labelFilter.filter(({ type, valueId }) => type === 'label' || valueId === -1).length && {
        label_id: labelFilter
          .filter(({ type, valueId }) => type === 'label' || valueId === -1)
          .map(({ labelId }) => labelId)
      }),
      ...(labelFilter.filter(({ type }) => type === 'value').length && {
        value_id: labelFilter
          .filter(({ type, valueId }) => type === 'value' && valueId !== -1)
          .map(({ valueId }) => valueId)
      })
    },
    queryParamStringifyOptions: {
      arrayFormat: 'repeat'
    },
    debounce: 500,
    lazy: !repoMetadata
  })

  const eventHandler = useCallback(
    (pr: TypesPullReq) => {
      // ensure this update belongs to the repo we are looking at right now - to avoid unnecessary reloads
      if (!pr || !repoMetadata || pr.target_repo_id !== repoMetadata.id) {
        return
      }

      refetchPrs()
    },
    [repoMetadata, refetchPrs]
  )
  useSpaceSSE({
    space,
    events: SSE_EVENTS,
    onEvent: eventHandler
  })

  const permPushResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY',
        resourceIdentifier: repoMetadata?.identifier as string
      },
      permissions: ['code_repo_push']
    },
    [space]
  )

  const handleLabelClick = (labelFilterArr: LabelFilterObj[], clickedLabel: TypesLabelPullReqAssignmentInfo) => {
    // if not present - add :
    const isLabelAlreadyAdded = labelFilterArr.map(({ labelId }) => labelId).includes(clickedLabel.id || -1)
    const updatedLabelsList = [...labelFilterArr]
    if (!isLabelAlreadyAdded && clickedLabel?.id) {
      if (clickedLabel.value && clickedLabel.value_id) {
        updatedLabelsList.push({
          labelId: clickedLabel.id,
          type: LabelFilterType.VALUE,
          valueId: clickedLabel.value_id,
          labelObj: clickedLabel,
          valueObj: {
            id: clickedLabel.value_id,
            color: clickedLabel.value_color,
            label_id: clickedLabel.id,
            value: clickedLabel.value
          }
        })
      } else if (clickedLabel.value_count && !clickedLabel.value_id) {
        updatedLabelsList.push({
          labelId: clickedLabel.id,
          type: LabelFilterType.VALUE,
          valueId: -1,
          labelObj: clickedLabel,
          valueObj: {
            id: -1,
            color: clickedLabel.value_color,
            label_id: clickedLabel.id,
            value: getString('labels.anyValueOption')
          }
        })
      } else {
        updatedLabelsList.push({
          labelId: clickedLabel.id,
          type: LabelFilterType.LABEL,
          valueId: undefined,
          labelObj: clickedLabel,
          valueObj: undefined
        })
      }
      setLabelFilter(updatedLabelsList)
    }

    // if 'any value' label present - replace :
    const replacedAnyValueIfPresent = updatedLabelsList.map(filterObj => {
      if (
        filterObj.valueId === -1 &&
        filterObj.labelId === clickedLabel.id &&
        clickedLabel.value_id &&
        clickedLabel.value
      ) {
        return {
          ...filterObj,
          valueId: clickedLabel.value_id,
          valueObj: {
            id: clickedLabel.value_id,
            color: clickedLabel.value_color,
            label_id: clickedLabel.id,
            value: clickedLabel.value
          }
        }
      }
      return filterObj
    })
    const isUpdated = !updatedLabelsList.every((obj, index) => obj === replacedAnyValueIfPresent[index])
    if (isUpdated) {
      setLabelFilter(replacedAnyValueIfPresent)
    }
  }

  const columns: Column<TypesPullReq>[] = useMemo(
    () => [
      {
        id: 'title',
        width: '100%',
        Cell: ({ row }: CellProps<TypesPullReq>) => {
          return (
            <Container
              className={css.rowLink}
              onClick={() =>
                history.push(
                  routes.toCODEPullRequest({
                    repoPath: repoMetadata?.path as string,
                    pullRequestId: String(row.original.number)
                  })
                )
              }>
              <Layout.Horizontal className={css.titleRow} spacing="medium">
                <PullRequestStateLabel iconSize={22} data={row.original} iconOnly />
                <Container padding={{ left: 'small' }}>
                  <Layout.Vertical spacing="small">
                    <Container>
                      <Layout.Horizontal flex={{ alignItems: 'center' }} className={css.prLabels}>
                        <Layout.Horizontal spacing={'xsmall'}>
                          <Text color={Color.GREY_800} className={css.title} lineClamp={1}>
                            {row.original.title}
                          </Text>

                          <Container className={css.convo}>
                            <Icon
                              className={css.convoIcon}
                              padding={{ left: 'small', right: 'small' }}
                              name="code-chat"
                              size={15}
                            />
                            <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_500} tag="span">
                              {row.original.stats?.conversations}
                            </Text>
                          </Container>
                        </Layout.Horizontal>
                        <Render
                          when={
                            (isLabelEnabled || standalone) &&
                            row.original &&
                            row.original.labels &&
                            row.original.labels.length !== 0 &&
                            !prLoading
                          }>
                          {row.original?.labels?.map((label, index) => (
                            <Label
                              key={index}
                              name={label.key as string}
                              label_color={label.color as ColorName}
                              label_value={{
                                name: label.value as string,
                                color: label.value_color as ColorName
                              }}
                              scope={label.scope}
                              onClick={() => {
                                handleLabelClick(labelFilter, label)
                              }}
                            />
                          ))}
                        </Render>
                      </Layout.Horizontal>
                    </Container>
                    <Container>
                      <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }}>
                        <Text color={Color.GREY_500} font={{ size: 'small' }}>
                          <StringSubstitute
                            str={getString('pr.statusLine')}
                            vars={{
                              state: row.original.state,
                              number: <Text inline>{row.original.number}</Text>,
                              time: (
                                <strong>
                                  <TimePopoverWithLocal
                                    time={defaultTo(
                                      (row.original.state == 'merged'
                                        ? row.original.merged
                                        : row.original.created) as number,
                                      0
                                    )}
                                    inline={false}
                                    font={{ variation: FontVariation.SMALL_BOLD }}
                                    color={Color.GREY_500}
                                    tag="span"
                                  />
                                </strong>
                              ),
                              user: (
                                <strong>{row.original.author?.display_name || row.original.author?.email || ''}</strong>
                              )
                            }}
                          />
                        </Text>
                        <PipeSeparator height={10} />
                        <Container>
                          <Layout.Horizontal
                            spacing="xsmall"
                            style={{ alignItems: 'center' }}
                            onClick={Utils.stopEvent}>
                            <GitRefLink
                              text={row.original.target_branch as string}
                              url={routes.toCODERepository({
                                repoPath: repoMetadata?.path as string,
                                gitRef: row.original.target_branch
                              })}
                              showCopy={false}
                            />
                            <Text color={Color.GREY_500}>←</Text>
                            <GitRefLink
                              text={row.original.source_branch as string}
                              url={routes.toCODERepository({
                                repoPath: repoMetadata?.path as string,
                                gitRef: row.original.source_branch
                              })}
                              showCopy={false}
                            />
                          </Layout.Horizontal>
                        </Container>
                      </Layout.Horizontal>
                    </Container>
                  </Layout.Vertical>
                </Container>
                <FlexExpander />
                {/* TODO: Pass proper state when check api is fully implemented */}
                {/* <ExecutionStatusLabel data={{ state: 'success' }} /> */}
              </Layout.Horizontal>
            </Container>
          )
        }
      }
    ],
    [getString] // eslint-disable-line react-hooks/exhaustive-deps
  )

  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('pullRequests')}
        dataTooltipId="repositoryPullRequests"
      />
      <PageBody error={getErrorMessage(error || prError)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={loading || (prLoading && !searchTerm)} withBorder={!searchTerm} />

        <Render when={repoMetadata}>
          <Layout.Vertical>
            <PullRequestsContentHeader
              loading={prLoading && searchTerm !== undefined}
              repoMetadata={repoMetadata as RepoRepositoryOutput}
              activePullRequestFilterOption={filter}
              onPullRequestFilterChanged={_filter => {
                setFilter(_filter)
                setPage(1)
              }}
              onSearchTermChanged={value => {
                setSearchTerm(value)
                setPage(1)
              }}
              activePullRequestAuthorFilterOption={authorFilter}
              activePullRequestLabelFilterOption={labelFilter}
              onPullRequestAuthorFilterChanged={_authorFilter => {
                setAuthorFilter(_authorFilter)
                setPage(1)
              }}
              onPullRequestLabelFilterChanged={_labelFilter => {
                setLabelFilter(_labelFilter)
                setPage(1)
              }}
            />
            <Container padding="xlarge">
              <Container padding={{ top: 'medium', bottom: 'large' }}>
                <Layout.Horizontal
                  flex={{ alignItems: 'center', justifyContent: 'flex-start' }}
                  style={{ flexWrap: 'wrap', gap: '5px' }}>
                  <Render when={!prLoading}>
                    {isEmpty(data) ? (
                      <Text color={Color.GREY_400}>{getString('labels.noResults')}</Text>
                    ) : (
                      <Text color={Color.GREY_400}>
                        {
                          stringSubstitute(getString('labels.prCount'), {
                            count: data?.length
                          }) as string
                        }
                      </Text>
                    )}
                  </Render>
                  {(isLabelEnabled || standalone) &&
                    labelFilter &&
                    labelFilter?.length !== 0 &&
                    labelFilter?.map((label, index) => (
                      <Label
                        key={index}
                        name={label.labelObj.key as string}
                        label_color={label.labelObj.color as ColorName}
                        label_value={{
                          name: label.valueObj?.value as string,
                          color: label.valueObj?.color as ColorName
                        }}
                        scope={label.labelObj.scope}
                        removeLabelBtn={true}
                        handleRemoveClick={() => {
                          if (label.type === 'value') {
                            const updateFilterObjArr = labelFilter.filter(filterObj => {
                              if (!(filterObj.labelId === label.labelId && filterObj.type === 'value')) {
                                return filterObj
                              }
                            })
                            setLabelFilter(updateFilterObjArr)
                            setPage(1)
                          } else if (label.type === 'label') {
                            const updateFilterObjArr = labelFilter.filter(filterObj => {
                              if (!(filterObj.labelId === label.labelId && filterObj.type === 'label')) {
                                return filterObj
                              }
                            })
                            setLabelFilter(updateFilterObjArr)
                            setPage(1)
                          }
                        }}
                        disableRemoveBtnTooltip={true}
                      />
                    ))}
                </Layout.Horizontal>
              </Container>
              <Match expr={data?.length}>
                <Truthy>
                  <>
                    <TableV2<TypesPullReq>
                      className={css.table}
                      hideHeaders
                      columns={columns}
                      data={data || []}
                      getRowClassName={() => css.row}
                      onRowClick={noop}
                    />
                    <ResourceListingPagination response={response} page={page} setPage={setPage} />
                  </>
                </Truthy>
                <Case val={0}>
                  <NoResultCard
                    forSearch={!!searchTerm}
                    forFilter={!isEmpty(labelFilter) || !isEmpty(authorFilter)}
                    emptyFilterMessage={getString('pullRequestNotFoundforFilter')}
                    message={getString('pullRequestEmpty')}
                    buttonText={getString('newPullRequest')}
                    onButtonClick={() =>
                      history.push(
                        routes.toCODECompare({
                          repoPath: repoMetadata?.path as string,
                          diffRefs: makeDiffRefs(repoMetadata?.default_branch as string, '')
                        })
                      )
                    }
                    permissionProp={permissionProps(permPushResult, standalone)}
                  />
                </Case>
              </Match>
            </Container>
          </Layout.Vertical>
        </Render>
      </PageBody>
    </Container>
  )
}
