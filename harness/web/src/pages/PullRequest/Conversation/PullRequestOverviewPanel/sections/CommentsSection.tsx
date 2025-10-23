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
import { Color } from '@harnessio/design-system'
import { Button, ButtonSize, ButtonVariation, Container, Layout, SelectOption, Text } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import type { RepoRepositoryOutput, TypesPullReq, TypesViolation } from 'services/code'
import { PRCommentFilterType } from 'utils/Utils'
import Success from '../../../../../icons/code-success.svg?url'
import Fail from '../../../../../icons/code-fail.svg?url'
import css from '../PullRequestOverviewPanel.module.scss'
interface MergeSectionProps {
  repoMetadata: RepoRepositoryOutput
  pullReqMetadata: TypesPullReq
  resolvedCommentArr?: TypesViolation
  requiresCommentApproval: boolean
  setActivityFilter: (val: SelectOption) => void
}
const CommentsSection = (props: MergeSectionProps) => {
  const { resolvedCommentArr, requiresCommentApproval, setActivityFilter } = props
  const resolvedComments = requiresCommentApproval && !resolvedCommentArr?.params ? true : false
  const { getString } = useStrings()

  return (
    <Container flex={{ justifyContent: 'space-between' }}>
      <Layout.Horizontal flex={{ align: 'center-center' }}>
        {resolvedComments ? (
          <img alt={getString('success')} width={27} height={27} src={Success} />
        ) : (
          <img alt={getString('failed')} width={27} height={27} src={Fail} />
        )}

        {resolvedComments ? (
          <Text color={Color.GREEN_800} className={css.sectionTitle} padding={{ left: 'medium' }}>
            {getString('allCommentsResolved')}
          </Text>
        ) : (
          <Layout.Vertical>
            <Text color={Color.RED_700} className={css.sectionTitle} padding={{ left: 'medium', bottom: 'xsmall' }}>
              {getString('unrsolvedComment')}
            </Text>
            <Text color={Color.GREY_450} className={css.sectionSubheader} padding={{ left: 'medium' }}>
              {getString('resolveComments', { n: resolvedCommentArr?.params })}
            </Text>
          </Layout.Vertical>
        )}
      </Layout.Horizontal>
      {!resolvedComments ? (
        <Button
          className={cx(css.blueText, css.buttonPadding)}
          variation={ButtonVariation.LINK}
          size={ButtonSize.SMALL}
          text={getString('view')}
          iconProps={{ size: 10, margin: { left: 'xsmall' } }}
          onClick={() => {
            setActivityFilter({
              label: getString('unrsolvedComment'),
              value: PRCommentFilterType.UNRESOLVED_COMMENTS
            })
            setTimeout(() => {
              document.querySelectorAll('.bp3-input[value="Active"]')[0]?.scrollIntoView({ behavior: 'smooth' })
            }, 0)
          }}
        />
      ) : null}
    </Container>
  )
}

export default CommentsSection
