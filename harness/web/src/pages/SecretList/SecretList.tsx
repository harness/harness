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

import React, { useMemo, useState } from 'react'
import {
  ButtonVariation,
  Container,
  FlexExpander,
  Layout,
  PageBody,
  PageHeader,
  StringSubstitute,
  TableV2 as Table,
  Text,
  useToaster
} from '@harnessio/uicore'
import { Color, Intent } from '@harnessio/design-system'
import cx from 'classnames'
import type { CellProps, Column } from 'react-table'
import Keywords from 'react-keywords'
import { useGet, useMutate } from 'restful-react'
import { String, useStrings } from 'framework/strings'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { LIST_FETCHING_LIMIT, PageBrowserProps, formatDate, getErrorMessage, truncateString, voidFn } from 'utils/Utils'
import type { TypesSecret } from 'services/code'
import { usePageIndex } from 'hooks/usePageIndex'
import { useQueryParams } from 'hooks/useQueryParams'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { NewSecretModalButton } from 'components/NewSecretModalButton/NewSecretModalButton'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import useUpdateSecretModal from 'components/UpdateSecretModal/UpdateSecretModal'
import noSecretsImage from '../RepositoriesListing/no-repo.svg?url'
import css from './SecretList.module.scss'

const SecretList = () => {
  const space = useGetSpaceParam()
  const { getString } = useStrings()
  const [searchTerm, setSearchTerm] = useState<string | undefined>()
  const pageBrowser = useQueryParams<PageBrowserProps>()
  const pageInit = pageBrowser.page ? parseInt(pageBrowser.page) : 1
  const [page, setPage] = usePageIndex(pageInit)

  const {
    data: secrets,
    error,
    loading,
    refetch,
    response
  } = useGet<TypesSecret[]>({
    path: `/api/v1/spaces/${space}/+/secrets`,
    queryParams: { page, limit: LIST_FETCHING_LIMIT, query: searchTerm }
  })

  const NewSecretButton = (
    <NewSecretModalButton
      space={space}
      modalTitle={getString('secrets.create')}
      text={getString('secrets.newSecretButton')}
      variation={ButtonVariation.PRIMARY}
      icon="plus"
      onSuccess={() => refetch()}></NewSecretModalButton>
  )

  const { openModal: openUpdateSecretModal } = useUpdateSecretModal()

  const columns: Column<TypesSecret>[] = useMemo(
    () => [
      {
        Header: getString('secrets.name'),
        width: 'calc(100% - 180px)',
        Cell: ({ row }: CellProps<TypesSecret>) => {
          const record = row.original
          return (
            <Container className={css.nameContainer}>
              <Layout.Horizontal spacing="small" style={{ flexGrow: 1 }}>
                <Layout.Vertical flex className={css.name}>
                  <Text className={css.repoName} lineClamp={1}>
                    <Keywords value={searchTerm}>{record.identifier}</Keywords>
                  </Text>
                  {record.description && (
                    <Text className={css.desc} lineClamp={1}>
                      {record.description}
                    </Text>
                  )}
                </Layout.Vertical>
              </Layout.Horizontal>
            </Container>
          )
        }
      },
      {
        Header: getString('updatedDate'),
        width: '180px',
        Cell: ({ row }: CellProps<TypesSecret>) => {
          return (
            <Layout.Horizontal style={{ alignItems: 'center' }}>
              <Text color={Color.BLACK} lineClamp={1} rightIconProps={{ size: 10 }} width={120}>
                {formatDate(row.original.updated as number)}
              </Text>
            </Layout.Horizontal>
          )
        },
        disableSortBy: true
      },
      {
        id: 'action',
        width: '30px',
        Cell: ({ row }: CellProps<TypesSecret>) => {
          const { mutate: deleteSecret } = useMutate({
            verb: 'DELETE',
            path: `/api/v1/secrets/${space}/${row.original.identifier}/+`
          })
          const { showSuccess, showError } = useToaster()
          const confirmDeleteSecret = useConfirmAct()

          // TODO - add edit option
          return (
            <OptionsMenuButton
              isDark
              width="100px"
              items={[
                {
                  text: getString('edit'),
                  isDanger: true,
                  onClick: () => openUpdateSecretModal({ secretToUpdate: row.original, openSecretUpdate: refetch })
                },
                {
                  text: getString('delete'),
                  isDanger: true,
                  onClick: () =>
                    confirmDeleteSecret({
                      title: getString('secrets.deleteSecret'),
                      confirmText: getString('delete'),
                      intent: Intent.DANGER,
                      message: (
                        <String
                          useRichText
                          stringID="secrets.deleteSecretConfirm"
                          vars={{ uid: row.original.identifier }}
                        />
                      ),
                      action: async () => {
                        deleteSecret({})
                          .then(() => {
                            showSuccess(
                              <StringSubstitute
                                str={getString('secrets.secretDeleted')}
                                vars={{
                                  uid: truncateString(row.original.identifier as string, 20)
                                }}
                              />,
                              5000
                            )
                            refetch()
                          })
                          .catch(secretDeleteError => {
                            showError(getErrorMessage(secretDeleteError), 0, 'secrets.failedToDeleteSecret')
                          })
                      }
                    })
                }
              ]}
            />
          )
        }
      }
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [getString, refetch, searchTerm, space]
  )

  return (
    <Container className={css.main}>
      <PageHeader title={getString('pageTitle.secrets')} />
      <PageBody
        className={cx({ [css.withError]: !!error })}
        error={error ? getErrorMessage(error) : null}
        retryOnError={voidFn(refetch)}
        noData={{
          when: () => secrets?.length === 0 && searchTerm === undefined,
          image: noSecretsImage,
          message: getString('secrets.noData'),
          button: NewSecretButton
        }}>
        <LoadingSpinner visible={loading && !searchTerm} />

        <Container padding="xlarge">
          <Layout.Horizontal spacing="large" className={css.layout}>
            {NewSecretButton}
            <FlexExpander />
            <SearchInputWithSpinner loading={loading} query={searchTerm} setQuery={setSearchTerm} />
          </Layout.Horizontal>

          <Container margin={{ top: 'medium' }}>
            {!!secrets?.length && (
              <Table<TypesSecret>
                className={css.table}
                columns={columns}
                data={secrets || []}
                getRowClassName={row => cx(css.row, !row.original.description && css.noDesc)}
              />
            )}
            <NoResultCard
              showWhen={() => !!secrets && secrets?.length === 0 && !!searchTerm?.length}
              forSearch={true}
            />
          </Container>
          <ResourceListingPagination response={response} page={page} setPage={setPage} />
        </Container>
      </PageBody>
    </Container>
  )
}

export default SecretList
