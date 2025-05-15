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

import React, { useEffect, useState } from 'react'
import { useMutate } from 'restful-react'
import { Container, Layout, FlexExpander, ButtonVariation, Text, SplitButton } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color } from '@harnessio/design-system'
import { Menu, PopoverPosition, Icon as BIcon } from '@blueprintjs/core'
import type { RepoMergeCheck } from 'services/code'
import { useStrings } from 'framework/strings'
import { normalizeGitRef, type GitInfoProps } from 'utils/GitUtils'
import { BranchTagSelect } from 'components/BranchTagSelect/BranchTagSelect'
import { getErrorMessage, permissionProps } from 'utils/Utils'
import { UserPreference, useUserPreference } from 'hooks/useUserPreference'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import css from './CompareContentHeader.module.scss'

interface CompareContentHeaderProps extends Pick<GitInfoProps, 'repoMetadata'> {
  loading?: boolean
  targetGitRef: string
  onTargetGitRefChanged: (gitRef: string) => void
  sourceGitRef: string
  onSourceGitRefChanged: (gitRef: string) => void
  mergeable?: boolean
  onCreatePullRequestClick: (creationType: PRCreationType) => void
}

export function CompareContentHeader({
  loading,
  repoMetadata,
  targetGitRef,
  onTargetGitRefChanged,
  sourceGitRef,
  onSourceGitRefChanged,
  onCreatePullRequestClick
}: CompareContentHeaderProps) {
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const {
    hooks: { usePermissionTranslate },
    standalone
  } = useAppContext()
  const permPushResult = usePermissionTranslate(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY',
        resourceIdentifier: repoMetadata?.identifier as string
      },
      permissions: ['code_repo_push']
    },
    [space]
  )
  const [createOption, setCreateOption] = useUserPreference<PRCreationOption>(
    UserPreference.PULL_REQUEST_CREATION_OPTION,
    prCreationOptions[0]
  )

  return (
    <Container className={css.main} padding="xlarge">
      <Layout.Horizontal spacing="medium">
        <Icon name="code-branch" size={20} />
        <BranchTagSelect
          repoMetadata={repoMetadata}
          disableBranchCreation
          disableViewAllBranches
          gitRef={targetGitRef}
          onSelect={onTargetGitRefChanged}
          labelPrefix={getString('prefixBase')}
          placeHolder={getString('selectBranchPlaceHolder')}
          className={css.branchTagDropdown}
          popoverClassname={css.popoverClassname}
        />
        <Icon name="arrow-left" size={14} />
        <BranchTagSelect
          repoMetadata={repoMetadata}
          disableBranchCreation
          disableViewAllBranches
          gitRef={sourceGitRef}
          onSelect={onSourceGitRefChanged}
          labelPrefix={getString('prefixCompare')}
          placeHolder={getString('selectBranchPlaceHolder')}
          className={css.branchTagDropdown}
          popoverClassname={css.popoverClassname}
        />
        {!!targetGitRef && !!sourceGitRef && (
          <MergeableLabel repoMetadata={repoMetadata} targetGitRef={targetGitRef} sourceGitRef={sourceGitRef} />
        )}
        <FlexExpander />
        <SplitButton
          loading={loading}
          disabled={loading}
          text={createOption.title}
          variation={ButtonVariation.PRIMARY}
          popoverProps={{
            interactionKind: 'click',
            usePortal: true,
            popoverClassName: css.popover,
            position: PopoverPosition.BOTTOM_RIGHT,
            transitionDuration: 1000
          }}
          {...permissionProps(permPushResult, standalone)}
          onClick={() => {
            onCreatePullRequestClick(createOption.type)
          }}>
          {prCreationOptions.map(option => {
            return (
              <Menu.Item
                key={option.type}
                className={css.menuItem}
                text={
                  <>
                    <BIcon icon={createOption.type === option.type ? 'tick' : 'blank'} />
                    <strong>{option.title}</strong>
                    <p>{option.desc}</p>
                  </>
                }
                onClick={() => setCreateOption(option)}
              />
            )
          })}
        </SplitButton>
      </Layout.Horizontal>
    </Container>
  )
}

const MergeableLabel: React.FC<Pick<CompareContentHeaderProps, 'repoMetadata' | 'targetGitRef' | 'sourceGitRef'>> = ({
  repoMetadata,
  targetGitRef,
  sourceGitRef
}) => {
  const {
    mutate: mergeCheck,
    loading,
    error
  } = useMutate<RepoMergeCheck>({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata.path}/+/merge-check/${normalizeGitRef(targetGitRef)}..${normalizeGitRef(
      sourceGitRef
    )}`
  })
  const [mergeable, setMergable] = useState<boolean | undefined>()
  const color = mergeable ? Color.GREEN_700 : mergeable === false ? Color.RED_500 : undefined
  const { getString } = useStrings()

  useEffect(() => {
    if (targetGitRef && sourceGitRef) {
      mergeCheck({})
        .then(response => {
          setMergable(response.mergeable)
        })
        .catch(err => {
          getErrorMessage(err)
        })
    }
  }, [targetGitRef, sourceGitRef, mergeCheck])

  useEffect

  return (
    <Text
      className={css.mergeText}
      icon={loading ? 'steps-spinner' : mergeable === true ? 'command-artifact-check' : 'cross'}
      iconProps={{ color, margin: { right: 'xsmall' } }}
      color={color}>
      {loading ? '' : error ? getErrorMessage(error) : getString(mergeable ? 'pr.ableToMerge' : 'pr.cantMerge')}
    </Text>
  )
}

export enum PRCreationType {
  NORMAL = 'normal',
  DRAFT = 'draft'
}

interface PRCreationOption {
  type: PRCreationType
  title: string
  desc: string
}

const prCreationOptions: PRCreationOption[] = [
  {
    type: PRCreationType.NORMAL,
    title: 'Create pull request',
    desc: 'Open a pull request that is ready for review'
  },
  {
    type: PRCreationType.DRAFT,
    title: 'Create draft pull request',
    desc: 'Does not request code reviews and cannot be merged'
  }
]
