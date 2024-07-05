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

import React, { useState } from 'react'
import { Button, ButtonVariation, Dialog, Layout, Text, Container, TextInput, useToaster } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color } from '@harnessio/design-system'
import { Divider } from '@blueprintjs/core'
import { useHistory } from 'react-router-dom'
import { useModalHook } from 'hooks/useModalHook'
import { useStrings } from 'framework/strings'
import { useDeleteRepository } from 'services/code'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { getErrorMessage } from 'utils/Utils'

const useDeleteRepoModal = () => {
  const { repoMetadata } = useGetRepositoryMetadata()
  const space = useGetSpaceParam()

  const { getString } = useStrings()
  const { routes } = useAppContext()

  const { mutate: deleteRepo, loading } = useDeleteRepository({})
  const { showSuccess, showError } = useToaster()
  const history = useHistory()

  const [showConfirmPage, setShowConfirmPage] = useState(false)
  const [deleteConfirmString, setDeleteConfirmString] = useState('')

  const [openModal, hideModal] = useModalHook(() => {
    const onClose = () => {
      setShowConfirmPage(false)
      setDeleteConfirmString('')
      hideModal()
    }
    return (
      <Dialog isOpen enforceFocus={false} onClose={onClose} title={getString('repoDelete.title')}>
        <Layout.Vertical flex={{ justifyContent: 'center' }}>
          <Icon name="nav-project" size={32} />
          <Text font={{ size: 'large' }} color="grey900" padding={{ top: 'small', bottom: 'medium' }}>
            {repoMetadata?.identifier}
          </Text>
        </Layout.Vertical>
        <Divider />
        {!showConfirmPage ? (
          <>
            <Container
              intent="warning"
              background="yellow100"
              border={{
                color: 'orange500'
              }}
              margin={{ top: 'small' }}>
              <Text
                icon="warning-outline"
                iconProps={{ size: 16, margin: { right: 'small' } }}
                padding={{ left: 'large', right: 'large', top: 'small', bottom: 'small' }}
                color={Color.WARNING}>
                {getString('spaceSetting.deleteWarning')}
              </Text>
            </Container>
            <Text padding={{ top: 'small' }} color="grey500">
              {getString('repoDelete.deleteConfirm1', {
                repo: repoMetadata?.identifier
              })}
            </Text>
            <Button
              variation={ButtonVariation.TERTIARY}
              margin={{ top: 'large' }}
              onClick={() => {
                setShowConfirmPage(true)
              }}
              width="100%">
              {getString('repoDelete.deleteConfirmButton1')}
            </Button>
          </>
        ) : (
          <>
            <Text padding={{ top: 'small', bottom: 'small' }} color="grey500">
              {getString('repoDelete.deleteConfirm2', {
                repo: repoMetadata?.identifier
              })}
            </Text>
            <TextInput
              placeholder={repoMetadata?.identifier}
              value={deleteConfirmString}
              onInput={e => {
                setDeleteConfirmString(e.currentTarget.value)
              }}
            />
            <Button
              variation={ButtonVariation.SECONDARY}
              intent="danger"
              loading
              disabled={deleteConfirmString !== repoMetadata?.identifier || loading}
              margin={{ top: 'small' }}
              onClick={async () => {
                try {
                  // this isn't implemented in the backend yet
                  await deleteRepo(`${repoMetadata?.path as string}/+/`)
                  setShowConfirmPage(false)
                  setDeleteConfirmString('')
                  hideModal()
                  history.push(routes.toCODERepositories({ space }))
                  showSuccess(getString('repoDelete.deleteToastSuccess'))
                } catch (e) {
                  showError(getErrorMessage(e))
                }
              }}
              width="100%"
              text={getString('repoDelete.deleteConfirmButton2')}></Button>
          </>
        )}
      </Dialog>
    )
  }, [showConfirmPage, deleteConfirmString, loading, repoMetadata])

  return {
    openModal,
    hideModal
  }
}

export default useDeleteRepoModal
