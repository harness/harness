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
import React from 'react'
import cx from 'classnames'
import { Button, ButtonVariation, Container, Layout, StringSubstitute, Text, useToaster } from '@harnessio/uicore'
import type { MutateMethod } from 'restful-react'
import { Icon } from '@harnessio/icons'
import { useStrings } from 'framework/strings'
import { CodeIcon } from 'utils/GitUtils'
import { getErrorMessage } from 'utils/Utils'
import type { TypesBranch } from 'services/code'
import css from '../PullRequestOverviewPanel.module.scss'

interface DeleteBranchSectionProps {
  sourceSha: string
  sourceBranch: TypesBranch | null
  deleteBranch: MutateMethod<
    any,
    any,
    {
      bypass_rules: boolean
      dry_run_rules: boolean
      commit_sha: string
    },
    unknown
  >
  setShowDeleteBranchButton: React.Dispatch<React.SetStateAction<boolean>>
  setIsSourceBranchDeleted: React.Dispatch<React.SetStateAction<boolean>>
}

const DeleteBranchSection = ({
  sourceSha,
  sourceBranch,
  deleteBranch,
  setShowDeleteBranchButton,
  setIsSourceBranchDeleted
}: DeleteBranchSectionProps) => {
  const { getString } = useStrings()
  const { showSuccess, showError } = useToaster()

  return (
    <Container className={cx(css.deleteBranchSectionContainer, css.borderRadius)} padding={{ right: 'xlarge' }}>
      <Layout.Horizontal flex={{ justifyContent: 'space-between' }}>
        <Text flex={{ alignItems: 'center' }}>
          <StringSubstitute
            str={getString('pr.closedPrBranchDelete')}
            vars={{
              source: (
                <Container padding={{ left: 'small', right: 'small' }}>
                  <strong className={cx(css.boldText, css.branchContainer)}>
                    <Icon name={CodeIcon.Branch} size={16} />
                    <Text className={cx(css.boldText, css.widthContainer)} lineClamp={1}>
                      {sourceBranch?.name}
                    </Text>
                  </strong>
                </Container>
              )
            }}
          />
        </Text>
        <Button
          text={getString('deleteBranch')}
          variation={ButtonVariation.SECONDARY}
          onClick={() => {
            deleteBranch({}, { queryParams: { bypass_rules: true, dry_run_rules: false, commit_sha: sourceSha } })
              .then(() => {
                setIsSourceBranchDeleted(true)
                setShowDeleteBranchButton(false)
                showSuccess(
                  <StringSubstitute
                    str={getString('branchDeleted')}
                    vars={{
                      branch: sourceBranch?.name
                    }}
                  />,
                  5000
                )
              })
              .catch(err => showError(getErrorMessage(err)))
          }}
        />
      </Layout.Horizontal>
    </Container>
  )
}

export default DeleteBranchSection
