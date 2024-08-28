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
  TableV2,
  Text,
  Button,
  ButtonVariation,
  useToaster,
  StringSubstitute,
  Layout,
  Utils
} from '@harnessio/uicore'

import type { CellProps, Column, Renderer, Row, UseExpandedRowProps } from 'react-table'
import { useGet, useMutate } from 'restful-react'
import { Color } from '@harnessio/design-system'
import { Intent } from '@blueprintjs/core'
import { useHistory } from 'react-router-dom'
import { Icon } from '@harnessio/icons'
import { isEmpty } from 'lodash-es'
import { useQueryParams } from 'hooks/useQueryParams'
import { usePageIndex } from 'hooks/usePageIndex'
import {
  getErrorMessage,
  LIST_FETCHING_LIMIT,
  type PageBrowserProps,
  ColorName,
  LabelTypes,
  LabelListingProps,
  LabelsPageScope,
  getScopeData
} from 'utils/Utils'
import { CodeIcon } from 'utils/GitUtils'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { useStrings, String } from 'framework/strings'
import { useConfirmAction } from 'hooks/useConfirmAction'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { useAppContext } from 'AppContext'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import { LabelTitle, LabelValuesList, LabelValuesListQuery } from 'components/Label/Label'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { getConfig } from 'services/config'
import LabelsHeader from './LabelsHeader/LabelsHeader'
import useLabelModal from './LabelModal/LabelModal'
import css from './LabelsListing.module.scss'

const LabelsListing = (props: LabelListingProps) => {
  const { activeTab, currentPageScope, repoMetadata, space } = props
  const { standalone } = useAppContext()
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()
  const history = useHistory()
  const pageBrowser = useQueryParams<PageBrowserProps>()
  const { updateQueryParams, replaceQueryParams } = useUpdateQueryParams()
  const pageInit = pageBrowser.page ? parseInt(pageBrowser.page) : 1
  const [page, setPage] = usePageIndex(pageInit)
  const [searchTerm, setSearchTerm] = useState('')
  const [showParentScopeFilter, setShowParentScopeFilter] = useState<boolean>(true)
  const [inheritLabels, setInheritLabels] = useState<boolean>(false)

  useEffect(() => {
    const params = {
      ...pageBrowser,
      ...(page > 1 && { page: page.toString() })
    }
    updateQueryParams(params, undefined, true)

    if (page <= 1) {
      const updateParams = { ...params }
      delete updateParams.page
      replaceQueryParams(updateParams, undefined, true)
    }
  }, [page]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (currentPageScope) {
      if (currentPageScope === LabelsPageScope.ACCOUNT) setShowParentScopeFilter(false)
      else if (currentPageScope === LabelsPageScope.SPACE) setShowParentScopeFilter(false)
    }
  }, [currentPageScope, standalone])

  const getLabelPath = () =>
    currentPageScope === LabelsPageScope.REPOSITORY
      ? `/repos/${repoMetadata?.path}/+/labels`
      : `/spaces/${space}/+/labels`

  const {
    data: labelsList,
    loading: labelsListLoading,
    refetch,
    response
  } = useGet<LabelTypes[]>({
    base: getConfig('code/api/v1'),
    path: getLabelPath(),
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      inherited: inheritLabels,
      page: page,
      query: searchTerm
    },
    debounce: 500
  })

  const refetchlabelsList = useCallback(
    () =>
      refetch({
        queryParams: {
          limit: LIST_FETCHING_LIMIT,
          inherited: inheritLabels,
          page: page,
          query: searchTerm
        }
      }),
    [inheritLabels, LIST_FETCHING_LIMIT, page, searchTerm]
  )

  const { openModal: openLabelCreateModal, openUpdateLabelModal } = useLabelModal({ refetchlabelsList })
  const renderRowSubComponent = React.useCallback(({ row }: { row: Row<LabelTypes> }) => {
    if (standalone) {
      return (
        <LabelValuesList
          name={row.original?.key as string}
          scope={row.original?.scope as number}
          repoMetadata={repoMetadata}
          space={space}
          standalone={standalone}
        />
      )
    } else {
      return (
        <LabelValuesListQuery
          name={row.original?.key as string}
          scope={row.original?.scope as number}
          repoMetadata={repoMetadata}
          space={space}
          standalone={standalone}
        />
      )
    }
  }, [])

  const ToggleAccordionCell: Renderer<{
    row: UseExpandedRowProps<CellProps<LabelTypes>> & {
      original: LabelTypes
    }
    value: LabelTypes
  }> = ({ row }) => {
    if (row.original.value_count) {
      return (
        <Layout.Horizontal onClick={e => e?.stopPropagation()}>
          <Button
            data-testid="row-expand-btn"
            {...row.getToggleRowExpandedProps()}
            color={Color.GREY_600}
            icon={row.isExpanded ? 'chevron-down' : 'chevron-right'}
            variation={ButtonVariation.ICON}
            iconProps={{ size: 19 }}
            className={css.toggleAccordion}
          />
        </Layout.Horizontal>
      )
    }
    return null
  }

  const columns: Column<LabelTypes>[] = useMemo(
    () => [
      {
        Header: '',
        id: 'rowSelectOrExpander',
        width: '5%',
        Cell: ToggleAccordionCell
      },
      {
        Header: getString('name'),
        id: 'name',
        sort: 'true',
        width: '25%',
        Cell: ({ row }: CellProps<LabelTypes>) => {
          return (
            <Container className={css.labelCtn}>
              <LabelTitle
                name={row.original?.key as string}
                value_count={row.original.value_count}
                label_color={row.original.color as ColorName}
                scope={row.original.scope}
              />
            </Container>
          )
        }
      },
      {
        Header: getString('labels.createdIn'),
        id: 'scope',
        sort: 'true',
        width: '30%',
        Cell: ({ row }: CellProps<LabelTypes>) => {
          const { scopeIcon, scopeId } = getScopeData(space as string, row.original.scope ?? 1, standalone)
          return (
            <Layout.Horizontal spacing={'xsmall'} flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
              <Icon size={16} name={row.original.scope === 0 ? CodeIcon.Repo : scopeIcon} />
              <Text>{row.original.scope === 0 ? repoMetadata?.identifier : scopeId}</Text>
            </Layout.Horizontal>
          )
        }
      },
      {
        Header: getString('description'),
        id: 'description',
        width: '40%',
        sort: 'true',
        Cell: ({ row }: CellProps<LabelTypes>) => {
          return <Text lineClamp={1}>{row.original?.description}</Text>
        }
      },
      {
        id: 'action',
        width: '5%',
        Cell: ({ row }: CellProps<LabelTypes>) => {
          const encodedLabelKey = row.original.key ? encodeURIComponent(row.original.key) : ''
          const { scopeRef } = getScopeData(space as string, row.original?.scope ?? 1, standalone)
          const deleteLabelPath =
            row.original?.scope === 0
              ? `/repos/${encodeURIComponent(repoMetadata?.path as string)}/labels/${encodedLabelKey}`
              : `/spaces/${encodeURIComponent(scopeRef as string)}/labels/${encodedLabelKey}`

          const { mutate: deleteLabel } = useMutate({
            verb: 'DELETE',
            base: getConfig('code/api/v1'),
            path: deleteLabelPath
          })

          //ToDo : Remove the following block when Encoding is handled by BE for Harness
          const deleteLabelPathHarness =
            row.original?.scope === 0
              ? `/repos/${repoMetadata?.identifier}/labels/${encodedLabelKey}`
              : `/labels/${encodedLabelKey}`

          const { mutate: deleteLabelQueryCall } = useMutate({
            verb: 'DELETE',
            base: getConfig('code/api/v1'),
            path: deleteLabelPathHarness,
            queryParams: {
              accountIdentifier: scopeRef?.split('/')[0],
              orgIdentifier: scopeRef?.split('/')[1],
              projectIdentifier: scopeRef?.split('/')[2]
            }
          })

          //ToDo: remove type check of standalone when Encoding is handled by BE for Harness
          const confirmLabelDelete = useConfirmAction({
            title: getString('labels.deleteLabel'),
            confirmText: getString('delete'),
            intent: Intent.DANGER,
            message: <String useRichText stringID="labels.deleteLabelConfirm" vars={{ name: row.original.key }} />,
            action: async e => {
              e.stopPropagation()
              const handleSuccess = (tag: string) => {
                showSuccess(<StringSubstitute str={getString('labels.deletedLabel')} vars={{ tag }} />, 5000)
                refetchlabelsList()
                setPage(1)
              }

              const handleError = (error: any) => {
                showError(getErrorMessage(error), 0, getString('labels.failedToDeleteLabel'))
              }

              const deleteAction = standalone ? deleteLabel({}) : deleteLabelQueryCall({})

              deleteAction.then(() => handleSuccess(row.original.key ?? '')).catch(handleError)
            }
          })
          return (
            <Container margin={{ left: 'medium' }} onClick={Utils.stopEvent}>
              <OptionsMenuButton
                width="100px"
                items={[
                  {
                    text: getString('edit'),
                    iconName: CodeIcon.Edit,
                    hasIcon: true,
                    iconSize: 20,
                    className: css.optionItem,
                    onClick: () => {
                      openUpdateLabelModal(row.original)
                    }
                  },
                  {
                    text: getString('delete'),
                    iconName: CodeIcon.Delete,
                    iconSize: 20,
                    hasIcon: true,
                    isDanger: true,
                    className: css.optionItem,
                    onClick: confirmLabelDelete
                  }
                ]}
                isDark
              />
            </Container>
          )
        }
      }
    ], // eslint-disable-next-line react-hooks/exhaustive-deps
    [history, getString, repoMetadata?.path, space, setPage, showError, showSuccess]
  )

  return (
    <Container>
      <LabelsHeader
        activeTab={activeTab}
        onSearchTermChanged={(value: React.SetStateAction<string>) => {
          setSearchTerm(value)
          setPage(1)
        }}
        showParentScopeFilter={showParentScopeFilter}
        inheritLabels={inheritLabels}
        setInheritLabels={setInheritLabels}
        openLabelCreateModal={openLabelCreateModal}
        repoMetadata={repoMetadata}
        spaceRef={space}
        currentPageScope={currentPageScope}
      />

      <Container className={css.main} padding={{ bottom: 'large', right: 'xlarge', left: 'xlarge' }}>
        {labelsList && !labelsListLoading && labelsList.length !== 0 && (
          <TableV2<LabelTypes>
            className={css.table}
            columns={columns}
            data={labelsList}
            sortable
            renderRowSubComponent={renderRowSubComponent}
            autoResetExpanded={true}
            onRowClick={rowData => openUpdateLabelModal(rowData)}
          />
        )}
        <LoadingSpinner visible={labelsListLoading} />
        <ResourceListingPagination response={response} page={page} setPage={setPage} />
      </Container>
      <NoResultCard
        showWhen={() => !labelsListLoading && isEmpty(labelsList)}
        forSearch={!!searchTerm}
        message={getString('labels.noLabelsFound')}
        buttonText={getString('labels.newLabel')}
        onButtonClick={() => openLabelCreateModal()}
      />
    </Container>
  )
}

export default LabelsListing
