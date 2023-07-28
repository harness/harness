import React, { useState } from 'react'
import { Button, ButtonVariation, Dialog, Icon, Layout, Text, Container, Color, TextInput } from '@harness/uicore'
import { useModalHook } from '@harness/use-modal'

import { Divider } from '@blueprintjs/core'
import { useStrings } from 'framework/strings'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
// import { useDeleteSpace } from 'services/code'

const useDeleteSpaceModal = () => {
  const space = useGetSpaceParam()
  const { getString } = useStrings()

  // this isn't implemented in the backend yet
  // const { mutate: deleteSpace, loading } = useDeleteSpace({})
  // temporary loading state until backend is implemented
  const loading = false

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
              onClick={() => {
                // this isn't implemented in the backend yet
                // deleteSpace(encodeURIComponent(space))
                setShowConfirmPage(false)
                setDeleteConfirmString('')
                hideModal()
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
