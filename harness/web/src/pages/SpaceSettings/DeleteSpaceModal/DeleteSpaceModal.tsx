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
import { useAppContext } from 'AppContext'
import { useModalHook } from 'hooks/useModalHook'
import { useStrings } from 'framework/strings'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useDeleteSpace } from 'services/code'
import { getErrorMessage } from 'utils/Utils'

const useDeleteSpaceModal = () => {
  const space = useGetSpaceParam()
  const { getString } = useStrings()
  const { routes } = useAppContext()

  // this isn't implemented in the backend yet
  const { mutate: deleteSpace, loading } = useDeleteSpace({})
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
      <Dialog isOpen enforceFocus={false} onClose={onClose} title={getString('deleteSpace')}>
        <Layout.Vertical flex={{ justifyContent: 'center' }}>
          <Icon name="nav-project" size={32} />
          <Text font={{ size: 'large' }} color="grey900" padding={{ top: 'small', bottom: 'medium' }}>
            {space}
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
              {getString('spaceSetting.deleteConfirm1', {
                space
              })}
            </Text>
            <Button
              variation={ButtonVariation.TERTIARY}
              margin={{ top: 'large' }}
              onClick={() => {
                setShowConfirmPage(true)
              }}
              width="100%">
              {getString('spaceSetting.deleteConfirmButton1')}
            </Button>
          </>
        ) : (
          <>
            <Text padding={{ top: 'small', bottom: 'small' }} color="grey500">
              {getString('spaceSetting.deleteConfirm2', {
                space
              })}
            </Text>
            <TextInput
              placeholder={space}
              value={deleteConfirmString}
              onInput={e => {
                setDeleteConfirmString(e.currentTarget.value)
              }}
            />
            <Button
              variation={ButtonVariation.SECONDARY}
              intent="danger"
              loading
              disabled={deleteConfirmString !== space || loading}
              margin={{ top: 'small' }}
              onClick={async () => {
                try {
                  // this isn't implemented in the backend yet
                  await deleteSpace(encodeURIComponent(space))
                  setShowConfirmPage(false)
                  setDeleteConfirmString('')
                  hideModal()
                  history.push(routes.toCODEHome())
                  showSuccess(getString('spaceSetting.deleteToastSuccess'))
                } catch (e) {
                  showError(getErrorMessage(e))
                }
              }}
              width="100%"
              text={getString('spaceSetting.deleteConfirmButton2')}></Button>
          </>
        )}
      </Dialog>
    )
  }, [showConfirmPage, deleteConfirmString, loading])

  return {
    openModal,
    hideModal
  }
}

export default useDeleteSpaceModal
