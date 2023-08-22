import React, { useMemo, useState } from 'react'
import {
  ButtonVariation,
  Container,
  FlexExpander,
  Layout,
  PageBody,
  PageHeader,
  TableV2 as Table,
  Text
} from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import cx from 'classnames'
import type { CellProps, Column } from 'react-table'
import Keywords from 'react-keywords'
import { useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
import { useStrings } from 'framework/strings'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { LIST_FETCHING_LIMIT, PageBrowserProps, formatDate, getErrorMessage, voidFn } from 'utils/Utils'
import type { TypesSecret } from 'services/code'
import { usePageIndex } from 'hooks/usePageIndex'
import { useQueryParams } from 'hooks/useQueryParams'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { NewSecretModalButton } from 'components/NewSecretModalButton/NewSecretModalButton'
import { useAppContext } from 'AppContext'
import noSecretsImage from '../RepositoriesListing/no-repo.svg'
import css from './SecretList.module.scss'

const SecretList = () => {
  const { routes } = useAppContext()
  const space = useGetSpaceParam()
  const history = useHistory()
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
      modalTitle={getString('secrets.newSecretButton')}
      text={getString('secrets.newSecretButton')}
      variation={ButtonVariation.PRIMARY}
      icon="plus"
      onSubmit={secretInfo =>
        history.push(routes.toCODESecret({ space, secret: secretInfo.uid as string }))
      }></NewSecretModalButton>
  )

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
                  <Text className={css.repoName}>
                    <Keywords value={searchTerm}>{record.uid}</Keywords>
                  </Text>
                  {record.description && <Text className={css.desc}>{record.description}</Text>}
                </Layout.Vertical>
              </Layout.Horizontal>
            </Container>
          )
        }
      },
      {
        Header: getString('repos.updated'),
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
      }
    ],
    [getString, searchTerm]
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
                onRowClick={secretInfo =>
                  history.push(routes.toCODESecret({ space: 'root', secret: secretInfo.uid as string }))
                }
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
