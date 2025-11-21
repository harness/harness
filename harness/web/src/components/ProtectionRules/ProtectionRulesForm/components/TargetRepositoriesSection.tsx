import React, { useEffect, useMemo, useState } from 'react'
import cx from 'classnames'
import {
  Button,
  ButtonVariation,
  Dialog,
  ExpandingSearchInput,
  Layout,
  PageSpinner,
  SplitButton
} from '@harnessio/uicore'
import { Container } from '@harnessio/uicore'
import { Text } from '@harnessio/uicore'
import { PopoverPosition } from '@blueprintjs/core'
import { Menu } from '@blueprintjs/core'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { useGet } from 'restful-react'
import type { CellProps, Column } from 'react-table'
import type { FormikProps } from 'formik'
import { compact } from 'lodash-es'
import { RulesTargetType } from 'utils/GitUtils'
import { CodeIcon } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import { useModalHook } from 'hooks/useModalHook'
import type { RepoRepositoryOutput } from 'services/code'
import ResourceHandlerTable, { ResourceHandlerTableData } from 'components/ResourceHandlerTable/ResourceHandlerTable'
import type { RulesFormPayload } from 'components/ProtectionRules/ProtectionRulesUtils'
import { ScopeBadge } from 'components/ScopeBadge/ScopeBadge'
import { getRelativeSpaceRef, getScopeFromParams, type ScopeEnum } from 'utils/Utils'
import Include from '../../../../icons/Include.svg?url'
import Exclude from '../../../../icons/Exclude.svg?url'
import css from '../ProtectionRulesForm.module.scss'

export function TargetRepositories({
  formik,
  fieldName,
  targets,
  currentScope,
  standalone
}: {
  formik: FormikProps<RulesFormPayload>
  fieldName: string
  targets?: string[][]
  currentScope: ScopeEnum
  standalone: boolean
}) {
  if (!targets?.length) return null

  return (
    <Layout.Horizontal spacing={'small'} className={css.targetBox}>
      {targets.map((tgt, idx) => {
        const path = tgt[1]
        const repoName = path?.split('/').at(-1)
        const [accountId, repoOrgIdentifier, repoProjectIdentifier] = path?.split('/').slice(0, -1) || []
        const repoScope = getScopeFromParams(
          { accountId, orgIdentifier: repoOrgIdentifier, projectIdentifier: repoProjectIdentifier },
          standalone
        )
        const relativeSpaceRef = getRelativeSpaceRef(currentScope, repoScope, repoOrgIdentifier, repoProjectIdentifier)

        return (
          <Container key={`${idx}-${path}`} className={cx(css.greyButton, css.target)}>
            <img width={16} height={16} src={tgt[0] === RulesTargetType.INCLUDE ? Include : Exclude} />
            <Text lineClamp={1} alwaysShowTooltip tooltip={compact([relativeSpaceRef, repoName]).join('/')}>
              {repoName}
            </Text>
            <Icon
              name="code-close"
              onClick={() => {
                const filteredData = targets.filter(item => !(item[0] === tgt[0] && item[1] === path))
                formik.setFieldValue(fieldName, filteredData)
              }}
              className={css.codeClose}
            />
          </Container>
        )
      })}
    </Layout.Horizontal>
  )
}

const AddRepoModal = ({
  space,
  searchTerm,
  selectedData,
  disabledRows,
  onSelectChange,
  standalone,
  currentScope
}: {
  space: string
  searchTerm: string
  selectedData?: string[]
  disabledRows?: string[]
  onSelectChange: (items: string[]) => void
  standalone: boolean
  currentScope: ScopeEnum
}) => {
  const { getString } = useStrings()
  const [page, setPage] = useState(0)

  useEffect(() => {
    setPage(0)
  }, [searchTerm])

  const {
    data: repositories,
    loading,
    response
  } = useGet<RepoRepositoryOutput[]>({
    path: `/api/v1/spaces/${space}/+/repos`,
    queryParams: {
      page: page + 1,
      limit: 10,
      query: searchTerm,
      recursive: true
    }
  })

  const columns: Column<ResourceHandlerTableData>[] = useMemo(
    () => [
      {
        Header: getString('repositories'),
        accessor: 'id',
        width: '55%',
        Cell: ({ row }: CellProps<RepoRepositoryOutput>) => {
          return (
            <Layout.Horizontal spacing={'small'}>
              <Text
                icon={'code-repo'}
                iconProps={{ size: 20 }}
                flex={{ align: 'center-center' }}
                font={{ variation: FontVariation.BODY2_SEMI }}
                color={Color.GREY_600}>
                {row.original.identifier}
              </Text>
            </Layout.Horizontal>
          )
        }
      },
      {
        Header: '',
        id: 'scope',
        width: '40%',
        Cell: ({ row }: CellProps<RepoRepositoryOutput>) => {
          return <ScopeBadge standalone={standalone} currentScope={currentScope} path={row.original.path} />
        }
      }
    ],
    [getString]
  )

  if (loading) return <PageSpinner />

  return repositories?.length ? (
    <Container>
      <ResourceHandlerTable
        data={repositories as ResourceHandlerTableData[]}
        selectedData={selectedData}
        disabledRows={disabledRows}
        columns={columns}
        onSelectChange={onSelectChange}
        pagination={{
          itemCount: parseInt(response?.headers?.get('x-total') || '') || 0,
          pageSize: parseInt(response?.headers?.get('x-per-page') || '') || 10,
          pageCount: parseInt(response?.headers?.get('x-total-pages') || '') || -1,
          pageIndex: page || 0,
          gotoPage: pageIndex => setPage(pageIndex)
        }}
      />
    </Container>
  ) : (
    <Layout.Vertical flex={{ align: 'center-center' }} spacing="small" className={css.noDataContainer}>
      <Text
        icon={'code-repo'}
        iconProps={{ size: 20 }}
        flex={{ align: 'center-center' }}
        font={{ variation: FontVariation.BODY1 }}
        color={Color.BLACK}>
        {getString('repos.noDataMessage')}
      </Text>
    </Layout.Vertical>
  )
}

export const TargetRepositoriesSection = ({
  formik,
  space,
  standalone,
  currentScope
}: {
  formik: FormikProps<RulesFormPayload>
  space: string
  standalone: boolean
  currentScope: ScopeEnum
}) => {
  const { repoList = [] } = formik.values
  const { getString } = useStrings()
  const [repoTargetType, setRepoTargetType] = useState(RulesTargetType.INCLUDE)
  const [searchTerm, setSearchTerm] = useState('')
  const [includedRepos, setIncludedRepos] = useState<string[]>([])
  const [excludedRepos, setExcludedRepos] = useState<string[]>([])
  const isRepoTargetIncluded = useMemo(() => repoTargetType === RulesTargetType.INCLUDE, [repoTargetType])

  // Had to set this as the data was not present in the initial render
  useEffect(() => {
    setIncludedRepos(
      repoList.filter(([type]) => type === RulesTargetType.INCLUDE).map(([, path, id]) => [path, id].join('/'))
    )
    setExcludedRepos(
      repoList.filter(([type]) => type === RulesTargetType.EXCLUDE).map(([, path, id]) => [path, id].join('/'))
    )
  }, [repoList])

  const [openModal, hideModal] = useModalHook(() => {
    const onClose = () => {
      setSearchTerm('')
      hideModal()
    }

    const onSuccess = () => {
      const includedArr =
        includedRepos?.map(repo => {
          const parts = repo.split('/')
          return [RulesTargetType.INCLUDE, parts.slice(0, -1).join('/'), parts.at(-1)]
        }) ?? []

      const excludedArr =
        excludedRepos?.map(repo => {
          const parts = repo.split('/')
          return [RulesTargetType.EXCLUDE, parts.slice(0, -1).join('/'), parts.at(-1)]
        }) ?? []
      formik.setFieldValue('repoList', [...includedArr, ...excludedArr])
      setSearchTerm('')
      hideModal()
    }

    return (
      <Dialog
        isOpen
        enforceFocus={false}
        onClose={onClose}
        className={css.dialog}
        title={
          (repoTargetType === RulesTargetType.INCLUDE ? getString('include') : getString('exclude')) +
          ' ' +
          getString('repositories')
        }>
        <Layout.Vertical padding="xsmall">
          <ExpandingSearchInput
            alwaysExpanded
            onChange={text => {
              setSearchTerm(text.trim())
            }}
          />
          <Container className={css.modal}>
            <AddRepoModal
              standalone={standalone}
              currentScope={currentScope}
              space={space}
              searchTerm={searchTerm}
              selectedData={isRepoTargetIncluded ? includedRepos : excludedRepos}
              disabledRows={isRepoTargetIncluded ? excludedRepos : includedRepos}
              onSelectChange={items => {
                isRepoTargetIncluded ? setIncludedRepos(items) : setExcludedRepos(items)
              }}
            />
          </Container>
          <Layout.Horizontal spacing="small">
            <Button
              variation={ButtonVariation.PRIMARY}
              text={getString('repos.confirmSelection')}
              onClick={onSuccess}
            />
            <Button text={getString('cancel')} onClick={onClose} />
          </Layout.Horizontal>
        </Layout.Vertical>
      </Dialog>
    )
  }, [includedRepos, excludedRepos, repoTargetType, searchTerm])

  return (
    <Container padding={{ bottom: 'small' }}>
      <Text font={{ variation: FontVariation.FORM_LABEL }} className={css.labelText} padding={{ bottom: 'xsmall' }}>
        {getString('repositories')}
      </Text>
      <SplitButton
        variation={ButtonVariation.TERTIARY}
        text={
          <Container flex={{ alignItems: 'center' }}>
            <img width={16} height={16} src={isRepoTargetIncluded ? Include : Exclude} />
            <Text
              padding={{ left: 'xsmall' }}
              color={Color.BLACK}
              font={{ variation: FontVariation.BODY2_SEMI, weight: 'bold' }}>
              Select {getString(repoTargetType)}d
            </Text>
          </Container>
        }
        onClick={() => {
          openModal()
        }}
        popoverProps={{
          interactionKind: 'click',
          usePortal: true,
          popoverClassName: css.popover,
          position: PopoverPosition.BOTTOM_RIGHT
        }}>
        {Object.values(RulesTargetType).map(type => (
          <Menu.Item
            key={type}
            className={css.menuItem}
            text={
              <Container flex={{ justifyContent: 'flex-start' }}>
                <Icon name={type === repoTargetType ? CodeIcon.Tick : CodeIcon.Blank} />
                <Text padding={{ left: 'xsmall' }} color={Color.BLACK} font={{ variation: FontVariation.BODY2_SEMI }}>
                  Select {getString(type)}d
                </Text>
              </Container>
            }
            onClick={() => setRepoTargetType(type)}
          />
        ))}
      </SplitButton>
      <Text className={css.hintText} margin={{ top: 'xsmall', bottom: 'small' }}>
        {getString('protectionRules.repoSelectionHint')}
      </Text>
      <TargetRepositories
        currentScope={currentScope}
        standalone={standalone}
        formik={formik}
        fieldName={'repoList'}
        targets={repoList}
      />
    </Container>
  )
}

export default TargetRepositoriesSection
