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

import React, { useMemo, useState } from 'react'
import {
  Button,
  ButtonVariation,
  Container,
  Dialog,
  Formik,
  FormikForm,
  FormInput,
  Layout,
  Select,
  SelectOption,
  Text,
  useToaster
} from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { Menu } from '@blueprintjs/core'
import cx from 'classnames'
import type { MutateRequestOptions } from 'restful-react/dist/Mutate'

import type { EnumMergeMethod, OpenapiMergePullReq, TypesPullReq, TypesRuleViolations } from 'services/code'
import { useStrings } from 'framework/strings'

import { getErrorMessage, PRMergeOption } from 'utils/Utils'

import { useGetPullRequestInfo } from 'pages/PullRequest/useGetPullRequestInfo'
import css from './PullRequestActionsBox.module.scss'

interface MergeSideDialogBoxProps {
  sideDialogOpen: boolean
  setSideDialogOpen: (value: React.SetStateAction<boolean>) => void
  mergeOption: PRMergeOption
  allowedStrats: string[]
  mergeOptions: PRMergeOption[]
  setMergeOption: (val: PRMergeOption) => void
  mergeable: boolean
  ruleViolation: boolean
  bypass: boolean
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
      | undefined // eslint-disable-next-line @typescript-eslint/no-explicit-any
  ) => Promise<any>
  setPrMerged: React.Dispatch<React.SetStateAction<boolean>>
  pullReqMetadata: TypesPullReq
  onPRStateChanged: () => void
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
}

const MergeSideDialogBox = (props: MergeSideDialogBoxProps) => {
  const {
    bypass,
    ruleViolation,
    mergeable,
    sideDialogOpen,
    setSideDialogOpen,
    mergeOption,
    allowedStrats,
    mergeOptions,
    setMergeOption,
    mergePR,
    setPrMerged,
    pullReqMetadata,
    onPRStateChanged,
    setRuleViolationArr
  } = props
  const { getString } = useStrings()
  const { showError } = useToaster()
  const { pullReqCommits } = useGetPullRequestInfo()
  const [mergeStrat, setMergeStrat] = useState(mergeOption.value)

  const initialValues = useMemo(() => {
    let messageString = ''
    if (pullReqCommits?.commits) {
      pullReqCommits?.commits.map(commit => {
        messageString = `* ${commit.title}\n   ${commit.message}\n`
      })
    }
    return {
      commitMessage: messageString
    }
  }, [pullReqCommits])
  return (
    <Dialog
      className={css.dialogContainer}
      isOpen={sideDialogOpen}
      onClose={() => {
        setSideDialogOpen(false)
      }}>
      <Formik
        initialValues={initialValues}
        formName="MergeDialog"
        enableReinitialize={true}
        validateOnChange
        validateOnBlur
        onSubmit={(data: { commitMessage: string }) => {
          const payload: OpenapiMergePullReq = {
            method: mergeStrat as EnumMergeMethod,
            source_sha: pullReqMetadata?.source_sha,
            bypass_rules: bypass,
            dry_run: false,
            message: data.commitMessage
          }
          mergePR(payload)
            .then(() => {
              setPrMerged(true)
              onPRStateChanged()
              setRuleViolationArr(undefined)
            })
            .catch(exception => showError(getErrorMessage(exception)))
        }}>
        {() => (
          <FormikForm>
            <Container className={css.dialogContentContainer}>
              <Layout.Vertical className={css.content} flex={{ alignItems: 'start', justifyContent: 'space-between' }}>
                <Container padding="small" className={css.widthContent}>
                  <Text padding={{ bottom: 'medium' }} font={{ variation: FontVariation.H4 }}>
                    {getString('pr.mergePR')}
                  </Text>
                  <Select
                    className={css.mergeStratText}
                    name={'mergeStrategy'}
                    value={mergeOption}
                    key={'mergeStrat'}
                    itemRenderer={(item: SelectOption): React.ReactElement => {
                      const mergeCheck = allowedStrats !== undefined && allowedStrats.includes(item.value as string)
                      return (
                        <>
                          {item.value !== 'close' && (
                            <Menu.Item
                              className={css.mergeItemContainer}
                              key={item.value as string}
                              disabled={!mergeCheck}
                              title={item.label}
                              onClick={() => {
                                setMergeStrat(item.value as string)
                                setMergeOption(item as PRMergeOption)
                              }}
                              text={
                                <Layout.Horizontal flex={{ distribution: 'space-between' }}>
                                  <Text lineClamp={1}>{item.label}</Text>
                                </Layout.Horizontal>
                              }
                            />
                          )}
                        </>
                      )
                    }}
                    items={mergeOptions}
                    onChange={(e: SelectOption) => {
                      setMergeStrat(e.value as string)
                    }}
                  />

                  {(mergeStrat === 'squash' || mergeStrat === 'squash') && (
                    <Layout.Vertical padding={{ top: 'small' }}>
                      <Text padding={{ bottom: 'small' }} font={{ variation: FontVariation.FORM_LABEL }}>
                        {getString('commitMessage')}
                      </Text>
                      <Container>
                        <FormInput.TextArea placeholder={getString('writeDownCommit')} name={'commitMessage'} />
                      </Container>
                    </Layout.Vertical>
                  )}
                </Container>

                <Container>
                  <Layout.Horizontal>
                    <Container
                      className={cx({
                        [css.btnWrapper]: mergeOption.method !== 'close',
                        [css.hasError]: mergeable === false,
                        [css.hasRuleViolated]: ruleViolation,
                        [css.bypass]: bypass
                      })}>
                      <Button
                        color={Color.GREEN_800}
                        className={cx({
                          [css.secondaryButton]: mergeOption.method === 'close' || mergeable === false
                        })}
                        variation={ButtonVariation.PRIMARY}
                        text={getString('pr.mergePR')}
                        type="submit"></Button>
                    </Container>
                    <Container padding={{ left: 'small' }}>
                      <Button
                        text={getString('cancel')}
                        variation={ButtonVariation.TERTIARY}
                        onClick={() => {
                          setSideDialogOpen(false)
                        }}></Button>
                    </Container>
                  </Layout.Horizontal>
                </Container>
              </Layout.Vertical>
            </Container>
          </FormikForm>
        )}
      </Formik>
    </Dialog>
  )
}

export default MergeSideDialogBox
