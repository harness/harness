/*
 * Copyright 2021 Harness Inc. All rights reserved.
 * Use of this source code is governed by the PolyForm Shield 1.0.0 license
 * that can be found in the licenses directory at the root of this repository, also available at
 * https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
 */

import React, { useEffect, useMemo, useRef, useState } from 'react'
import { Classes, Icon as BPIcon, Menu, MenuItem, PopoverPosition } from '@blueprintjs/core'
import {
  Button,
  ButtonProps,
  Container,
  Layout,
  ButtonVariation,
  TextInput,
  Tabs,
  FontVariation,
  Text
} from '@harness/uicore'
import { Link, useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
import { String, useStrings } from 'framework/strings'
import { getErrorMessage, LIST_FETCHING_PER_PAGE } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { CodeIcon, GitInfoProps, GitRefType } from 'utils/GitUtils'
import { useCreateBranchModal } from 'components/CreateBranchModalButton/CreateBranchModalButton'
import css from './BranchTagSelect.module.scss'

export interface BranchTagSelectProps extends Omit<ButtonProps, 'onSelect'>, Pick<GitInfoProps, 'repoMetadata'> {
  gitRef: string
  gitRefType: GitRefType
  onSelect: (ref: string, type: GitRefType) => void
}

export const BranchTagSelect: React.FC<BranchTagSelectProps> = ({
  repoMetadata,
  gitRef,
  gitRefType = GitRefType.BRANCH,
  onSelect,
  ...props
}) => {
  const { routes } = useAppContext()
  const history = useHistory()
  const [query, onQuery] = useState('')
  const openCreateNewBranchModal = useCreateBranchModal({
    repoMetadata,
    onSuccess: branchInfo => {
      history.push(
        routes.toSCMRepository({
          repoPath: repoMetadata.path as string,
          gitRef: branchInfo.name
        })
      )
    },
    suggestedBranchName: query,
    suggestedSourceBranch: gitRef,
    showSuccessMessage: true
  })

  return (
    <Button
      text={gitRef}
      icon={gitRefType == GitRefType.BRANCH ? CodeIcon.Branch : CodeIcon.Tag}
      rightIcon="chevron-down"
      variation={ButtonVariation.TERTIARY}
      iconProps={{ size: 14 }}
      tooltip={
        <PopoverContent
          gitRef={gitRef}
          gitRefType={gitRefType}
          repoMetadata={repoMetadata}
          onSelect={onSelect}
          onQuery={onQuery}
          openCreateNewBranchModal={openCreateNewBranchModal}
        />
      }
      tooltipProps={{
        interactionKind: 'click',
        usePortal: true,
        position: PopoverPosition.BOTTOM_LEFT,
        popoverClassName: css.popover
      }}
      tabIndex={0}
      {...props}
    />
  )
}

interface PopoverContentProps extends BranchTagSelectProps {
  openCreateNewBranchModal: () => void
  onQuery: (query: string) => void
}

const PopoverContent: React.FC<PopoverContentProps> = ({
  repoMetadata,
  gitRef,
  gitRefType,
  onSelect,
  onQuery,
  openCreateNewBranchModal
}) => {
  const { getString } = useStrings()
  const [activeTab, setActiveTab] = useState(gitRefType)
  const isBranchesTabActive = useMemo(() => activeTab === GitRefType.BRANCH, [activeTab])
  const inputRef = useRef<HTMLInputElement | null>()
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(false)

  return (
    <Container padding="medium" className={css.main}>
      <Layout.Vertical spacing="small">
        <TextInput
          className={css.input}
          inputRef={ref => (inputRef.current = ref)}
          defaultValue={query}
          autoFocus
          placeholder={getString(isBranchesTabActive ? 'findOrCreateBranch' : 'findATag')}
          onInput={e => {
            const _value = (e.currentTarget.value || '').trim()
            setQuery(_value)
            onQuery(_value)
          }}
          leftIcon={loading ? CodeIcon.InputSpinner : CodeIcon.InputSearch}
        />

        <Container className={css.tabContainer}>
          <Tabs
            id="branchesTags"
            defaultSelectedTabId={activeTab}
            large={false}
            tabList={[
              {
                id: GitRefType.BRANCH,
                title: getString('branches'),
                panel: (
                  <GitRefList
                    gitRef={gitRef}
                    gitRefType={GitRefType.BRANCH}
                    openCreateNewBranchModal={openCreateNewBranchModal}
                    onSelect={branch => onSelect(branch, GitRefType.BRANCH)}
                    repoMetadata={repoMetadata}
                    query={query}
                    setLoading={setLoading}
                  />
                )
              },
              {
                id: GitRefType.TAG,
                title: getString('tags'),
                panel: (
                  <GitRefList
                    gitRef={gitRef}
                    gitRefType={GitRefType.TAG}
                    openCreateNewBranchModal={openCreateNewBranchModal}
                    onSelect={branch => onSelect(branch, GitRefType.TAG)}
                    repoMetadata={repoMetadata}
                    query={query}
                    setLoading={setLoading}
                  />
                )
              }
            ]}
            onChange={tabId => {
              setActiveTab(tabId as GitRefType)
              inputRef.current?.focus()
            }}
          />
        </Container>
      </Layout.Vertical>
    </Container>
  )
}

interface GitRefListProps extends Omit<PopoverContentProps, 'onQuery'> {
  query: string
  setLoading: React.Dispatch<React.SetStateAction<boolean>>
}

function GitRefList({
  gitRef,
  gitRefType,
  repoMetadata,
  query,
  onSelect,
  openCreateNewBranchModal,
  setLoading
}: GitRefListProps) {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const { data, error, loading } = useGet<{ name: string }[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/${gitRefType === GitRefType.BRANCH ? 'branches' : 'tags'}`,
    queryParams: {
      per_page: LIST_FETCHING_PER_PAGE,
      page: 1,
      sort: 'date',
      direction: 'desc',
      include_commit: false,
      query
    }
  })

  useEffect(() => {
    setLoading(loading)
  }, [setLoading, loading])

  return (
    <Container>
      {!!error && (
        <Container flex={{ align: 'center-center' }} padding="large">
          {!!error && <Text font={{ variation: FontVariation.FORM_MESSAGE_DANGER }}>{getErrorMessage(error)}</Text>}
        </Container>
      )}

      {!!data?.length && (
        <Container className={css.listContainer}>
          <Menu>
            {data.map(({ name }) => (
              <MenuItem
                key={name}
                text={name}
                labelElement={name === gitRef ? <BPIcon icon="small-tick" /> : undefined}
                disabled={name === gitRef}
                onClick={() => onSelect(name as string, gitRefType)}
              />
            ))}
          </Menu>
        </Container>
      )}

      {data?.length === 0 && (
        <Container flex={{ align: 'center-center' }} padding="large">
          {(gitRefType === GitRefType.BRANCH && (
            <Button
              text={
                <String
                  stringID="createBranchFromBranch"
                  tagName="span"
                  className={css.newBtnText}
                  vars={{ newBranch: query, targetBranch: gitRef }}
                  useRichText
                />
              }
              icon={CodeIcon.Branch}
              variation={ButtonVariation.SECONDARY}
              onClick={openCreateNewBranchModal}
              className={Classes.POPOVER_DISMISS}
            />
          )) || (
            <Text>
              <String stringID="tagNotFound" tagName="span" vars={{ tag: query }} useRichText />
            </Text>
          )}
        </Container>
      )}

      {gitRefType === GitRefType.BRANCH && (
        <Container border={{ top: true }} flex={{ align: 'center-center' }} padding={{ top: 'small' }}>
          <Link to={routes.toSCMRepositoryBranches({ repoPath: repoMetadata.path as string })}>
            {getString('viewAllBranches')}
          </Link>
        </Container>
      )}
    </Container>
  )
}
