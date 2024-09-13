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

import React, { useEffect, useMemo, useRef, useState } from 'react'
import { Classes, Icon as BPIcon, Menu, MenuItem, PopoverPosition } from '@blueprintjs/core'
import { Button, ButtonProps, Container, Layout, ButtonVariation, TextInput, Tabs, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useHistory } from 'react-router-dom'
import cx from 'classnames'
import { useGet } from 'restful-react'
import { noop } from 'lodash-es'
import { String, useStrings } from 'framework/strings'
import { getErrorMessage, LIST_FETCHING_LIMIT } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { CodeIcon, GitInfoProps, GitRefType, isRefATag, REFS_TAGS_PREFIX } from 'utils/GitUtils'
import Branches from '../../icons/Branches.svg?url'
import css from './BranchTagSelect.module.scss'

export interface BranchTagSelectProps extends Omit<ButtonProps, 'onSelect'>, Pick<GitInfoProps, 'repoMetadata'> {
  gitRef: string
  onSelect: (ref: string, type: GitRefType) => void
  onCreateBranch?: (newBranch?: string) => void
  disableBranchCreation?: boolean
  disableViewAllBranches?: boolean
  forBranchesOnly?: boolean
  labelPrefix?: string
  placeHolder?: string
  popoverClassname?: string
  hidePopoverContent?: boolean
}

export const BranchTagSelect: React.FC<BranchTagSelectProps> = ({
  repoMetadata,
  gitRef,
  onSelect,
  onCreateBranch = noop,
  disableBranchCreation,
  disableViewAllBranches,
  forBranchesOnly,
  labelPrefix,
  placeHolder,
  className,
  popoverClassname,
  hidePopoverContent,
  ...props
}) => {
  const [query, onQuery] = useState('')
  const [gitRefType, setGitRefType] = useState(isRefATag(gitRef) ? GitRefType.TAG : GitRefType.BRANCH)
  const text = gitRef.replace(REFS_TAGS_PREFIX, '')
  return (
    <Button
      className={cx(css.button, className, gitRefType == GitRefType.BRANCH ? css.branchContainer : null)}
      text={
        text ? (
          labelPrefix ? (
            <>
              {gitRefType == GitRefType.BRANCH ? (
                <span className={css.branchSpan}>
                  <img src={Branches} width={14} height={14} />
                </span>
              ) : null}

              <span className={css.prefix}>{labelPrefix}</span>
              {text}
            </>
          ) : (
            <>
              {gitRefType == GitRefType.BRANCH ? (
                <span className={css.branchSpan}>
                  <img src={Branches} width={14} height={14} />
                </span>
              ) : null}
              {text}
            </>
          )
        ) : (
          <span className={css.prefix}>{placeHolder}</span>
        )
      }
      icon={gitRefType == GitRefType.BRANCH ? undefined : CodeIcon.Tag}
      rightIcon="chevron-down"
      variation={ButtonVariation.TERTIARY}
      iconProps={{ size: 14 }}
      tooltip={
        <PopoverContent
          hidePopoverContent={hidePopoverContent}
          gitRef={gitRef}
          gitRefType={gitRefType}
          repoMetadata={repoMetadata}
          onSelect={(ref, type) => {
            onSelect(type === GitRefType.BRANCH ? ref : `${REFS_TAGS_PREFIX}${ref}`, type)
            setGitRefType(type)
          }}
          onQuery={onQuery}
          forBranchesOnly={forBranchesOnly}
          disableBranchCreation={disableBranchCreation}
          disableViewAllBranches={disableViewAllBranches}
          onCreateBranch={() => onCreateBranch(query)}
          className={className}
          popoverClassname={popoverClassname}
        />
      }
      tooltipProps={{
        interactionKind: 'click',
        usePortal: true,
        position: PopoverPosition.BOTTOM_LEFT,
        popoverClassName: cx(css.popover, popoverClassname)
      }}
      tabIndex={0}
      {...props}
    />
  )
}

interface PopoverContentProps extends BranchTagSelectProps {
  gitRefType: GitRefType
  onQuery: (query: string) => void
}

const PopoverContent: React.FC<PopoverContentProps> = ({
  repoMetadata,
  gitRef,
  gitRefType,
  onSelect,
  onCreateBranch,
  onQuery,
  forBranchesOnly,
  disableBranchCreation,
  disableViewAllBranches,
  hidePopoverContent,
  className,
  popoverClassname
}) => {
  const { getString } = useStrings()
  const [activeTab, setActiveTab] = useState(gitRefType)
  const isBranchesTabActive = useMemo(() => activeTab === GitRefType.BRANCH, [activeTab])
  const inputRef = useRef<HTMLInputElement | null>()
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(false)

  return !hidePopoverContent ? (
    <Container padding="small" className={cx(css.main, { [css.maxWidth]: !className || !popoverClassname })}>
      <Layout.Vertical spacing="small" className={css.layout}>
        <TextInput
          className={css.input}
          inputRef={ref => (inputRef.current = ref)}
          defaultValue={query}
          autoFocus
          placeholder={getString(
            isBranchesTabActive ? (disableBranchCreation ? 'findBranch' : 'findOrCreateBranch') : 'findATag'
          )}
          onInput={e => {
            const _value = (e.currentTarget.value || '').trim()
            setQuery(_value)
            onQuery(_value)
          }}
          leftIcon={loading ? CodeIcon.InputSpinner : CodeIcon.InputSearch}
          leftIconProps={{
            name: loading ? CodeIcon.InputSpinner : CodeIcon.InputSearch,
            size: 12,
            color: Color.GREY_500
          }}
        />

        <Container className={cx(css.tabContainer, forBranchesOnly && css.branchesOnly)}>
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
                    activeGitRefType={gitRefType}
                    gitRefType={GitRefType.BRANCH}
                    onCreateBranch={onCreateBranch}
                    onSelect={branch => onSelect(branch, GitRefType.BRANCH)}
                    repoMetadata={repoMetadata}
                    query={query}
                    disableBranchCreation={disableBranchCreation}
                    disableViewAllBranches={disableViewAllBranches}
                    setLoading={setLoading}
                  />
                )
              },
              {
                id: GitRefType.TAG,
                title: getString('tags'),
                panel: (
                  <GitRefList
                    gitRef={gitRef.replace(REFS_TAGS_PREFIX, '')}
                    activeGitRefType={gitRefType}
                    gitRefType={GitRefType.TAG}
                    onCreateBranch={onCreateBranch}
                    onSelect={branch => onSelect(branch, GitRefType.TAG)}
                    repoMetadata={repoMetadata}
                    query={query}
                    disableBranchCreation={disableBranchCreation}
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
  ) : (
    <></>
  )
}

interface GitRefListProps extends Omit<PopoverContentProps, 'onQuery'> {
  activeGitRefType: GitRefType
  query: string
  setLoading: React.Dispatch<React.SetStateAction<boolean>>
}

function GitRefList({
  gitRef,
  gitRefType,
  activeGitRefType,
  repoMetadata,
  query,
  onSelect,
  onCreateBranch = noop,
  disableBranchCreation,
  disableViewAllBranches,
  setLoading
}: GitRefListProps) {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const { data, error, loading } = useGet<{ name: string }[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/${gitRefType === GitRefType.BRANCH ? 'branches' : 'tags'}`,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      page: 1,
      sort: 'date',
      order: 'desc',
      include_commit: false,
      query
    }
  })

  useEffect(() => {
    setLoading(loading)
  }, [setLoading, loading])
  const history = useHistory()
  return (
    <Container>
      {!!error && (
        <Container flex={{ align: 'center-center' }} padding="large">
          {!!error && <Text font={{ variation: FontVariation.FORM_MESSAGE_DANGER }}>{getErrorMessage(error)}</Text>}
        </Container>
      )}

      {!!data?.length && (
        <Container className={css.listContainer} padding={{ top: 'small', bottom: 'small' }}>
          <Menu>
            {data.map(({ name }) => {
              const isItemSelected = name === gitRef && activeGitRefType === gitRefType
              return (
                <MenuItem
                  key={name}
                  text={name}
                  labelElement={isItemSelected ? <BPIcon icon="tick" /> : undefined}
                  disabled={isItemSelected}
                  onClick={() => onSelect(name as string, gitRefType)}
                />
              )
            })}
          </Menu>
        </Container>
      )}

      {data?.length === 0 && (
        <Container flex={{ align: 'center-center' }} padding="medium">
          {(gitRefType === GitRefType.BRANCH &&
            ((disableBranchCreation && (
              <Text padding={{ top: 'small' }}>
                <String stringID="branchNotFound" tagName="span" vars={{ branch: query }} useRichText />
              </Text>
            )) || (
              <Button
                padding={'xsmall'}
                text={
                  <>
                    <String
                      stringID={
                        activeGitRefType === GitRefType.BRANCH ? 'createBranchFromBranch' : 'createBranchFromTag'
                      }
                      tagName="span"
                      className={css.newBtnText}
                      vars={{ newBranch: query, target: gitRef }}
                      useRichText
                    />
                  </>
                }
                iconProps={{ size: 22 }}
                icon={CodeIcon.BranchSmall}
                variation={ButtonVariation.SECONDARY}
                onClick={() => onCreateBranch()}
                className={cx(Classes.POPOVER_DISMISS, css.newBranchOption)}
              />
            ))) || (
            <Text padding={{ top: 'small' }}>
              <String stringID="tagNotFound" tagName="span" vars={{ tag: query }} useRichText />
            </Text>
          )}
        </Container>
      )}

      {!disableViewAllBranches && gitRefType === GitRefType.BRANCH && (
        <Container border={{ top: true }} flex={{ align: 'center-center' }} padding={{ top: 'xsmall' }}>
          <Button
            variation={ButtonVariation.LINK}
            text={getString('viewAllBranches')}
            onClick={() => history.push(routes.toCODEBranches({ repoPath: repoMetadata.path as string }))}
          />
        </Container>
      )}
    </Container>
  )
}
