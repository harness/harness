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
import { Button, ButtonVariation, Dialog, Layout, Text, useToaster } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useMutate } from 'restful-react'
import { useModalHook } from 'hooks/useModalHook'
import { String, useStrings } from 'framework/strings'
import { getErrorMessage, ACCESS_MODES } from 'utils/Utils'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'

interface DefaultBranchModalProps {
  currentGitRef: string
  setDefaultBranch: any
  refetch: () => void
}

const useDefaultBranchModal = ({
  currentGitRef,
  setDefaultBranch,
  refetch: refetchMetaData
}: DefaultBranchModalProps) => {
  const { repoMetadata } = useGetRepositoryMetadata()
  const { getString } = useStrings()
  const { mutate: changeDefaultBranch, loading: changingDefaultBranch } = useMutate({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata?.path}/+/default-branch`
  })
  const { showSuccess, showError } = useToaster()
  const defaultBranchPoints = getString('confirmDefaultBranch.points')?.split(',')
  const [openModal, hideModal] = useModalHook(() => {
    const onClose = () => {
      hideModal()
    }
    return (
      <Dialog
        isOpen
        enforceFocus={false}
        onClose={onClose}
        title={getString('confirmDefaultBranch.title')}
        style={{ width: 'auto' }}>
        <>
          <Text padding={{ bottom: 'small' }} color="black">
            <String stringID="confirmDefaultBranch.message" vars={{ currentGitRef }} useRichText />
          </Text>
          <ul style={{ margin: '0.05rem', padding: '1rem 1rem 1rem 1.5rem' }}>
            {defaultBranchPoints?.map((point: string, idx: number) => {
              return (
                <li key={idx}>
                  <Text color={Color.GREY_500} style={{ paddingBottom: '0.5rem', fontSize: '14px' }}>
                    {point}
                  </Text>
                </li>
              )
            })}
          </ul>
          <Layout.Horizontal margin={{ top: 'small' }}>
            <Button
              variation={ButtonVariation.PRIMARY}
              margin={{ right: 'small' }}
              onClick={async () => {
                try {
                  await changeDefaultBranch({ name: currentGitRef })
                  showSuccess(`changed default branch to ${currentGitRef}`, 5000)
                  setDefaultBranch(ACCESS_MODES.VIEW)
                  refetchMetaData()
                  hideModal()
                } catch (exception) {
                  showError(getErrorMessage(exception), 0, 'failed to change default branch')
                }
                refetchMetaData()
              }}>
              {getString('confirm')}
            </Button>
            <Button
              text={getString('cancel')}
              variation={ButtonVariation.TERTIARY}
              onClick={() => {
                hideModal()
              }}
            />
          </Layout.Horizontal>
        </>
      </Dialog>
    )
  }, [changingDefaultBranch, repoMetadata, currentGitRef])

  return {
    openModal,
    hideModal
  }
}

export default useDefaultBranchModal
