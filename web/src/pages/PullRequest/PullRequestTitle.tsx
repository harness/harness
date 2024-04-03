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
import { compact } from 'lodash-es'
import { useStrings } from 'framework/strings'
import { ButtonRoleProps, getErrorMessage } from 'utils/Utils'
import type { GitInfoProps } from 'utils/GitUtils'
import type { TypesPullReq } from 'services/code'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import { useDocumentTitle } from 'hooks/useDocumentTitle'
import css from './PullRequest.module.scss'

interface PullRequestTitleProps extends TypesPullReq, Pick<GitInfoProps, 'repoMetadata'> {
  onSaveDone?: (newTitle: string) => Promise<boolean>
  onAddDescriptionClick: () => void
}

export const PullRequestTitle: React.FC<PullRequestTitleProps> = ({
  repoMetadata,
  title,
  number,
  description,
  onAddDescriptionClick
}) => {
  const [original, setOriginal] = useState(title)
  const [val, setVal] = useState(title)
  const [edit, setEdit] = useState(false)
  const { getString } = useStrings()
  const { showError } = useToaster()
  const { mutate } = useMutate({
    verb: 'PATCH',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${number}`
  })
  const submitChange = useCallback(() => {
    mutate({
      title: val,
      description
    })
      .then(() => {
        setEdit(false)
        setOriginal(val)
      })
      .catch(exception => showError(getErrorMessage(exception), 0))
  }, [description, val, mutate, showError])

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
                disabled={(val || '').trim().length === 0 || title === val}
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
