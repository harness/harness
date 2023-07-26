import React, { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Button,
  ButtonVariation,
  Color,
  Container,
  FontVariation,
  Heading,
  Icon,
  Layout,
  NoDataCard,
  TableV2 as Table,
  Text
} from '@harness/uicore'
import cx from 'classnames'
import Keywords from 'react-keywords'

import { Classes, Popover, Position } from '@blueprintjs/core'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { ButtonRoleProps, LIST_FETCHING_LIMIT } from 'utils/Utils'
import { useShowRequestError } from 'hooks/useShowRequestError'
import { TypesRepository, TypesSpace, useGetSpace } from 'services/code'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { NewSpaceModalButton } from 'components/NewSpaceModalButton/NewSpaceModalButton'
// import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
// import { useGet } from 'restful-react'

// import { usePageIndex } from 'hooks/usePageIndex'
import type { CellProps, Column } from 'react-table'
import css from './SpaceSelector.module.scss'

interface SpaceSelectorProps {
  onSelect: (space: TypesSpace, isUserAction: boolean) => void
}

export const SpaceSelector: React.FC<SpaceSelectorProps> = ({ onSelect }) => {
  const { getString } = useStrings()
  const [selectedSpace, setSelectedSpace] = useState<TypesSpace | undefined>()
  const { space } = useGetRepositoryMetadata()
  const [opened, setOpened] = React.useState(false)
  const [searchTerm, setSearchTerm] = useState('')
  // const [page, setPage] = usePageIndex(1)

  const { data, error } = useGetSpace({ space_ref: encodeURIComponent(space), lazy: !space })
  // const {
  //   data: t,
  //   // loading,
  //   // refetch,
  //   response
  // } = useGet<TypesSpace[]>({
  //   path: `/api/v1/spaces/testspace`,
  //   queryParams: { page, limit: LIST_FETCHING_LIMIT, query: searchTerm }
  // })
  const spaces = [
    {
      id: 79,
      parent_id: 11,
      uid: 'root',
      path: 'root',
      description: 'This is a root description',
      is_public: false,
      created_by: 6,
      created: 1687816598981,
      updated: 1687816598981,
      default_branch: 'main',
      fork_id: 0,
      num_forks: 0,
      num_pulls: 0,
      num_closed_pulls: 0,
      num_open_pulls: 0,
      num_merged_pulls: 0,
      git_url: 'test.com/1'
    },
    {
      id: 8,
      parent_id: 11,
      uid: 'rootChild1',
      path: 'root/rootChild1',
      description: 'This is a rootchild1 description',
      is_public: false,
      created_by: 6,
      created: 1678875305900,
      updated: 1678875305900,
      default_branch: 'main',
      fork_id: 0,
      num_forks: 0,
      num_pulls: 0,
      num_closed_pulls: 0,
      num_open_pulls: 0,
      num_merged_pulls: 0,
      git_url: 'test.com/2'
    },
    {
      id: 80,
      parent_id: 11,
      uid: 'home',
      path: 'home',
      description: 'This is a home description',
      is_public: false,
      created_by: 6,
      created: 1687816598981,
      updated: 1687816598981,
      default_branch: 'main',
      fork_id: 0,
      num_forks: 0,
      num_pulls: 0,
      num_closed_pulls: 0,
      num_open_pulls: 0,
      num_merged_pulls: 0,
      git_url: 'test.com/3'
    }
  ]
  const selectSpace = useCallback(
    (_space: TypesSpace, isUserAction: boolean) => {
      setSelectedSpace(_space)
      onSelect(_space, isUserAction)
    },
    [onSelect]
  )

  useEffect(() => {
    if (space && !selectedSpace && data) {
      selectSpace(data, false)
    }
  }, [space, selectedSpace, data, onSelect, selectSpace])

  useShowRequestError(error)
  const NewSpaceButton = (
    <NewSpaceModalButton
      space={space}
      modalTitle={getString('createSpace')}
      text={getString('newSpace')}
      variation={ButtonVariation.PRIMARY}
      icon="plus"
    />
  )

  const columns: Column<TypesRepository>[] = useMemo(
    () => [
      {
        Header: getString('spaces'),
        width: 'calc(100% - 180px)',
        Cell: ({ row }: CellProps<TypesRepository>) => {
          const record = row.original
          return (
            <Container className={css.nameContainer}>
              <Layout.Horizontal spacing="small" style={{ flexGrow: 1 }}>
                <Icon
                  name={'nav-project'}
                  size={22}
                  className={css.iconContainer}
                  padding={{ bottom: 'small', left: 'small', right: 'medium' }}
                />
                <Layout.Vertical flex className={css.name}>
                  <Text className={css.repoName} lineClamp={2}>
                    <Keywords value={searchTerm}>{record.uid}</Keywords>
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
      }
    ],
    [getString, searchTerm]
  )

  return (
    <Popover
      portalClassName={css.popoverPortal}
      targetClassName={css.popoverTarget}
      popoverClassName={css.popoverContent}
      position={Position.RIGHT}
      usePortal={false}
      transitionDuration={0}
      captureDismiss
      onInteraction={setOpened}
      isOpen={opened}>
      <Container
        className={cx(css.spaceSelector, { [css.selected]: opened })}
        {...ButtonRoleProps}
        onClick={() => setOpened(!opened)}>
        <Layout.Horizontal>
          <Container className={css.label}>
            <Layout.Vertical>
              <Container>
                <Text className={css.spaceLabel} icon="nav-project" iconProps={{ size: 12 }}>
                  {getString('space').toUpperCase()}
                </Text>
              </Container>
            </Layout.Vertical>
            <Text className={css.spaceName} lineClamp={1}>
              {selectedSpace ? selectedSpace.uid : getString('selectSpace')}
            </Text>
          </Container>
          <Container className={css.icon}>
            <Icon name="main-chevron-right" size={10} />
          </Container>
        </Layout.Horizontal>
      </Container>

      <Container padding="large">
        <Heading level={2} padding={{ left: 'small' }} color={Color.BLACK}>
          <Layout.Horizontal flex={{ justifyContent: 'space-between', alignItems: 'center' }}>
            <Text font={{ variation: FontVariation.H5 }}>{getString('selectSpaceText')}</Text>
            {!!spaces?.length && (
              <Layout.Horizontal flex={{ justifyContent: 'space-between' }}>
                <SearchInputWithSpinner
                  loading={false}
                  spinnerPosition="left"
                  query={searchTerm}
                  setQuery={setSearchTerm}
                />
                <Container padding={{ left: 'small', right: 'small' }}></Container>
                {NewSpaceButton}
                <Button icon={'small-cross'} variation={ButtonVariation.ICON} className={Classes.POPOVER_DISMISS} />
              </Layout.Horizontal>
            )}
          </Layout.Horizontal>
        </Heading>
        <Container padding={{ left: 'small' }}>
          <Layout.Vertical padding={{ top: 'xxlarge' }} spacing="small">
            {!!spaces?.length && (
              <Table<TypesRepository>
                hideHeaders
                className={css.table}
                columns={columns}
                data={spaces || []}
                onRowClick={data => {
                  console.log(data)
                  setOpened(false)
                  selectSpace({ uid: data?.uid, path: data?.path }, true)
                }}
                getRowClassName={row => cx(css.row, !row.original.description && css.noDesc)}
              />
            )}
            {spaces?.length === 0 && (
              <NoDataCard
                button={
                  <NewSpaceModalButton
                    space={space}
                    modalTitle={getString('createSpace')}
                    text={getString('createSpace')}
                    variation={ButtonVariation.PRIMARY}
                    icon="plus"
                    // onSubmit={() => {}}
                  />
                }
                message={<Text font={{ variation: FontVariation.H4 }}> {getString('emptySpaceText')}</Text>}
              />
            )}
            {/* <ResourceListingPagination response={response} page={page} setPage={setPage} /> */}
          </Layout.Vertical>
        </Container>
      </Container>
    </Popover>
  )
}
