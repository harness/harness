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

import React, { useCallback, useEffect, useState } from 'react'
import { Container, Text, Layout, Button, ButtonVariation, ButtonSize, TextInput, useToaster } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { useMutate } from 'restful-react'
import { Match, Truthy, Else } from 'react-jsx-match'
import { compact, isEmpty } from 'lodash-es'
import { useStrings } from 'framework/strings'
import { ButtonRoleProps, getErrorMessage } from 'utils/Utils'
import type { GitInfoProps } from 'utils/GitUtils'
import type { TypesPullReq } from 'services/code'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import { useDocumentTitle } from 'hooks/useDocumentTitle'
import css from './PullRequest.module.scss'

interface PullRequestTitleProps extends TypesPullReq, Pick<GitInfoProps, 'repoMetadata'> {
  edit: boolean
  currentRef: string
  setEdit: React.Dispatch<React.SetStateAction<boolean>>
  onSaveDone?: (newTitle: string) => Promise<boolean>
  onAddDescriptionClick: () => void
}

export const PullRequestTitle: React.FC<PullRequestTitleProps> = ({
  repoMetadata,
  title,
  number,
  description,
  currentRef,
  target_branch,
  edit,
  setEdit,
  onAddDescriptionClick
}) => {
  const [original, setOriginal] = useState(title)
  const [val, setVal] = useState(title)
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()
  const { mutate: updatePRTitle } = useMutate({
    verb: 'PATCH',
    path: `/api/v1/repos/${repoMetadata?.path}/+/pullreq/${number}`
  })
  const { mutate: updateTargetBranch } = useMutate({
    verb: 'PUT',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${number}/target-branch`
  })
  const submitChange = useCallback(() => {
    const titleChanged = title !== val
    const targetBranchChanged = target_branch !== currentRef

    if (!titleChanged && !targetBranchChanged) {
      return
    }

    const promises = []

    if (titleChanged) {
      promises.push(
        updatePRTitle({
          title: val,
          description
        })
          .then(() => ({ success: true, type: 'title' }))
          .catch(exception => {
            showError(getErrorMessage(exception), 1000)
            return { success: false, type: 'title' }
          })
      )
    }

    if (targetBranchChanged) {
      promises.push(
        updateTargetBranch({ branch_name: currentRef })
          .then(() => ({ success: true, type: 'branch' }))
          .catch(exception => {
            showError(getErrorMessage(exception), 1000)
            return { success: false, type: 'branch' }
          })
      )
    }

    Promise.all(promises).then(results => {
      setEdit(false)
      if (titleChanged && results.some(r => r.type === 'title' && r.success)) {
        setOriginal(val)
      }

      const successful = results.filter(result => result.success)
      const titleSuccess = successful.some(r => r.type === 'title')
      const branchSuccess = successful.some(r => r.type === 'branch')

      if (titleSuccess && branchSuccess) {
        showSuccess(getString('pr.titleAndBranchUpdated', { branch: currentRef }), 3000)
      } else if (titleSuccess) {
        showSuccess(getString('pr.titleUpdated'), 3000)
      } else if (branchSuccess) {
        showSuccess(getString('pr.targetBranchUpdated', { branch: currentRef }), 3000)
      }
    })
  }, [description, val, title, target_branch, currentRef, updatePRTitle, updateTargetBranch, showError, showSuccess])

  useEffect(() => {
    setOriginal(title)

    // make sure to update editor if it's not open
    if (!edit) {
      setVal(title)
    }
  }, [title, edit])

  useDocumentTitle(compact([original, `(#${number})`]).join(' '))

  return (
    <Layout.Horizontal spacing="small" className={css.prTitle}>
      <Match expr={edit}>
        <Truthy>
          <Container>
            <Layout.Horizontal spacing="small">
              <TextInput
                wrapperClassName={css.input}
                value={val}
                onFocus={event => event.target.select()}
                onInput={event => setVal(event.currentTarget.value)}
                autoFocus
                onKeyDown={event => {
                  switch (event.key) {
                    case 'Enter':
                      submitChange()
                      break
                    case 'Escape': // does not work, maybe TextInput cancels ESC?
                      setEdit(false)
                      break
                  }
                }}
              />
              <Button
                variation={ButtonVariation.PRIMARY}
                text={getString('save')}
                size={ButtonSize.MEDIUM}
                disabled={isEmpty(val?.trim()) || !(title !== val || target_branch !== currentRef)}
                onClick={submitChange}
              />
              <Button
                variation={ButtonVariation.TERTIARY}
                text={getString('cancel')}
                size={ButtonSize.MEDIUM}
                onClick={() => {
                  setEdit(false)
                  setVal(title)
                }}
              />
            </Layout.Horizontal>
          </Container>
        </Truthy>
        <Else>
          <>
            <Text
              tag="h1"
              className={css.titleText}
              font={{ variation: FontVariation.H4 }}
              lineClamp={1}
              tooltipProps={{ portalClassName: css.popover }}>
              {original} <span className={css.prNumber}>#{number}</span>
            </Text>
            <Button
              variation={ButtonVariation.ICON}
              tooltip={getString('edit')}
              tooltipProps={{ isDark: true, position: 'right' }}
              size={ButtonSize.SMALL}
              icon="code-edit"
              className={css.btn}
              onClick={() => setEdit(true)}
            />
            {!(description || '').trim().length && (
              <>
                <PipeSeparator height={10} />
                <a className={css.addDesc} {...ButtonRoleProps} onClick={onAddDescriptionClick}>
                  &nbsp;{getString('pr.addDescription')}
                </a>
              </>
            )}
          </>
        </Else>
      </Match>
    </Layout.Horizontal>
  )
}
