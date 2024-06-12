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
  Button,
  ButtonVariation,
  Container,
  Heading,
  Layout,
  NoDataCard,
  TableV2 as Table,
  Text
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import cx from 'classnames'
import Keywords from 'react-keywords'
import { NavArrowRight } from 'iconoir-react'
import { useGet } from 'restful-react'
import type { CellProps, Column } from 'react-table'
import { useHistory } from 'react-router-dom'
import { Classes, Popover, Position } from '@blueprintjs/core'
import { useStrings } from 'framework/strings'
import { ButtonRoleProps, voidFn } from 'utils/Utils'
import { useShowRequestError } from 'hooks/useShowRequestError'
import { TypesSpace, useGetSpace } from 'services/code'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { NewSpaceModalButton } from 'components/NewSpaceModalButton/NewSpaceModalButton'
import { useAppContext } from 'AppContext'
import css from './SpaceSelector.module.scss'

interface SpaceSelectorProps {
  onSelect: (space: TypesSpace, isUserAction: boolean) => void
}

export const SpaceSelector: React.FC<SpaceSelectorProps> = ({ onSelect }) => {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const history = useHistory()
  const [selectedSpace, setSelectedSpace] = useState<TypesSpace | undefined>()
  const space = useGetSpaceParam()
  const [opened, setOpened] = React.useState(false)
  const [searchTerm, setSearchTerm] = useState('')
  // const [page, setPage] = usePageIndex(1)
  const { data, error } = useGetSpace({ space_ref: encodeURIComponent(space), lazy: !space })

  const {
    data: spaces,
    refetch,
    response
  } = useGet({
    path: '/api/v1/user/memberships',
    queryParams: { query: searchTerm },
    debounce: 500
  })

  const selectSpace = useCallback(
    (_space: TypesSpace, isUserAction: boolean) => {
      setSelectedSpace(_space)
      onSelect(_space, isUserAction)
    },
    [onSelect]
  )

  useEffect(() => {
    //space is used in the api call to get data, so it'll always be the same
    if (space && data) {
      selectSpace(data, false)
      refetch()
    }
  }, [data, refetch, selectSpace, space])

  useEffect(() => {
    if (space && !selectedSpace && data) {
      selectSpace(data, false)
    } else if (!space && selectSpace && data && selectedSpace?.id !== -1) {
      selectSpace(
        {
          created: 0,
          created_by: 0,
          description: '',
          id: -1,
          is_public: false,
          parent_id: 0,
          path: '',
          uid: getString('selectSpace'),
          updated: 0
        },
        false
      )
    }
  }, [space, selectedSpace, data, onSelect, selectSpace, getString])

  useEffect(() => {
    if (response?.status === 403) {
      history.push(routes.toSignIn())
    }
  }, [response, history, routes])

  useShowRequestError(error)
  const NewSpaceButton = (
    <NewSpaceModalButton
      space={space}
      modalTitle={getString('createSpace')}
      text={getString('newSpace')}
      variation={ButtonVariation.PRIMARY}
      icon="plus"
      onRefetch={voidFn(refetch)}
      onSubmit={spaceData => {
        history.push(routes.toCODERepositories({ space: spaceData.path as string }))
        setOpened(false)
      }}
      fromSpace={true}
      handleNavigation={spaceData => {
        history.push(routes.toCODERepositories({ space: spaceData as string }))
        setOpened(false)
      }}
    />
  )

  const columns: Column<{ space: TypesSpace }>[] = useMemo(
    () => [
      {
        Header: getString('spaces'),
        width: 'calc(100% - 180px)',
        Cell: ({ row }: CellProps<{ space: TypesSpace }>) => {
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
                    <Keywords value={searchTerm}>{(record.space as any).identifier}</Keywords>
                  </Text>
                  {record.space.description && (
                    <Text className={css.desc} lineClamp={1}>
                      {record.space.description}
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
            {/* <Layout.Vertical>
              <Container>
                <Text className={css.spaceLabel} icon="nav-project" iconProps={{ size: 12 }}>
                  {getString('space').toUpperCase()}
                </Text>
              </Container>
            </Layout.Vertical> */}
            <Text className={css.spaceName} lineClamp={1}>
              {selectedSpace ? selectedSpace.uid : getString('selectSpace')}
            </Text>
          </Container>
          <Container className={css.icon}>
            <NavArrowRight width={24} height={24} strokeWidth={1} />
          </Container>
        </Layout.Horizontal>
      </Container>

      <Container padding="large">
        <Heading level={2} padding={{ left: 'small' }} color={Color.BLACK}>
          <Layout.Horizontal flex={{ justifyContent: 'space-between', alignItems: 'center' }}>
            <Text font={{ variation: FontVariation.H5 }}>{getString('selectSpaceText')}</Text>
            {(!!spaces?.length || searchTerm.length >= 1) && (
              <Layout.Horizontal flex={{ justifyContent: 'space-between' }}>
                <SearchInputWithSpinner
                  loading={false}
                  spinnerPosition="left"
                  query={searchTerm}
                  setQuery={setSearchTerm}
                />
                <Container padding={{ left: 'small', right: 'small' }}></Container>
                {spaces?.length === 0 ? null : NewSpaceButton}
                <Button icon={'small-cross'} variation={ButtonVariation.ICON} className={Classes.POPOVER_DISMISS} />
              </Layout.Horizontal>
            )}
          </Layout.Horizontal>
        </Heading>
        <Container padding={{ left: 'small' }}>
          <Layout.Vertical padding={{ top: 'xxlarge' }} spacing="small">
            {!!spaces?.length && (
              <Table<{ space: TypesSpace }>
                hideHeaders
                className={cx(css.table, css.tableContainer)}
                columns={columns}
                data={spaces || []}
                onRowClick={spaceData => {
                  setOpened(false)
                  selectSpace({ uid: spaceData?.space?.uid, path: spaceData?.space?.path }, true)
                }}
                getRowClassName={row => cx(css.row, !row.original.space.description && css.noDesc)}
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
                    onRefetch={voidFn(refetch)}
                    onSubmit={spaceData => {
                      history.push(routes.toCODERepositories({ space: spaceData.path as string }))
                    }}
                    fromSpace={true}
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
