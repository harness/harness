/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react'
import { Container, FormInput, FormikForm, Layout, Text, useToaster } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { Formik, FormikProps } from 'formik'
import type { MutateRequestOptions } from 'restful-react/dist/Mutate'
import type { EnumMergeMethod, OpenapiMergePullReq, TypesPullReq, TypesRuleViolations } from 'services/code'
import { useStrings } from 'framework/strings'
import type { PRMergeOption } from 'pages/PullRequest/PullRequestUtils'
import { getErrorMessage, inlineMergeFormValues } from 'utils/Utils'
import { MergeStrategy } from 'utils/GitUtils'
import mergeVideo from '../../../../videos/merge.mp4'
import squashVideo from '../../../../videos/squash.mp4'
import rebaseVideo from '../../../../videos/rebase.mp4'
import fastForward from '../../../../videos/fastForward.mp4'
import css from './PullRequestActionsBox.module.scss'

interface InlineMergeBoxProps {
  inlineMergeRef: React.RefObject<FormikProps<inlineMergeFormValues>>
  mergeOption: PRMergeOption
  initialValues: { commitMessage: string; commitTitle: string }
  onPRStateChanged: () => void
  setShowInlineMergeContainer: (value: React.SetStateAction<boolean>) => void
  mergePR: (
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    data: any,
    mutateRequestOptions?:
      | MutateRequestOptions<
          {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            [key: string]: any
          },
          unknown
        >
      | undefined
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
  ) => Promise<any>
  pullReqMetadata: TypesPullReq
  bypass: boolean
  setRuleViolationArr: React.Dispatch<
    React.SetStateAction<
      | {
          data: {
            rule_violations: TypesRuleViolations[]
          }
        }
      | undefined
    >
  >
  refetchActivities: () => void
  refetchPullReq: () => void
}

const InlineMergeBox = (props: InlineMergeBoxProps) => {
  const {
    mergeOption,
    initialValues,
    setShowInlineMergeContainer,
    mergePR,
    pullReqMetadata,
    bypass,
    onPRStateChanged,
    setRuleViolationArr,
    inlineMergeRef,
    refetchActivities,
    refetchPullReq
  } = props
  const { getString } = useStrings()
  const { showError } = useToaster()

  return (
    <Container className={css.mergeStratContainer} padding={{ top: 'small', bottom: 'large' }}>
      <Container className={css.innerContainer} padding={'medium'} background={Color.PRIMARY_BG}>
        <Layout.Horizontal>
          <Layout.Vertical>
            <Container>
              <Text font={{ variation: FontVariation.CARD_TITLE }}>{mergeOption.title}</Text>
              <Container padding={{ top: 'large' }}>
                <Container>
                  <Container padding={{ right: 'medium' }}>
                    {mergeOption.method === MergeStrategy.REBASE ? (
                      <video height={36} width={148} src={rebaseVideo} autoPlay={true} loop={false} muted={true} />
                    ) : mergeOption.method === MergeStrategy.SQUASH ? (
                      <video height={36} width={148} src={squashVideo} autoPlay={true} loop={false} muted={true} />
                    ) : mergeOption.method === MergeStrategy.FAST_FORWARD ? (
                      <video height={36} width={148} src={fastForward} autoPlay={true} loop={false} muted={true} />
                    ) : (
                      <video height={36} width={148} src={mergeVideo} autoPlay={true} loop={false} muted={true} />
                    )}
                  </Container>
                </Container>
              </Container>
            </Container>
          </Layout.Vertical>
          <Layout.Vertical width={`75%`} padding={{ top: 'large' }}>
            <Container padding={{ left: 'medium' }} className={css.divider}>
              <Formik
                innerRef={inlineMergeRef}
                initialValues={initialValues}
                formName="MergeDialog"
                enableReinitialize={true}
                validateOnChange
                validateOnBlur
                onSubmit={(data: { commitMessage: string; commitTitle: string }) => {
                  const payload: OpenapiMergePullReq = {
                    method: mergeOption.method as EnumMergeMethod,
                    source_sha: pullReqMetadata?.source_sha,
                    bypass_rules: bypass,
                    dry_run: false,
                    message: data.commitMessage
                  }
                  if (mergeOption.method === MergeStrategy.SQUASH || mergeOption.method === MergeStrategy.MERGE) {
                    payload.message = data.commitMessage
                    payload.title = data.commitTitle
                  }

                  mergePR(payload)
                    .then(() => {
                      refetchPullReq()
                      onPRStateChanged()
                      setRuleViolationArr(undefined)
                      setShowInlineMergeContainer(false)
                      refetchActivities()
                    })
                    .catch(exception => showError(getErrorMessage(exception)))
                }}>
                {() => {
                  return (
                    <FormikForm>
                      {(mergeOption.method === MergeStrategy.SQUASH || mergeOption.method === MergeStrategy.MERGE) && (
                        <FormInput.Text name="commitTitle"></FormInput.Text>
                      )}
                      {mergeOption.method !== MergeStrategy.REBASE &&
                        mergeOption.method !== MergeStrategy.FAST_FORWARD && (
                          <FormInput.TextArea
                            placeholder={getString('addOptionalCommitMessage')}
                            name="commitMessage"></FormInput.TextArea>
                        )}
                    </FormikForm>
                  )
                }}
              </Formik>
            </Container>
          </Layout.Vertical>
        </Layout.Horizontal>
      </Container>
    </Container>
  )
}

export default InlineMergeBox
