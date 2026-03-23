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
import { getErrorInfoFromErrorObject, ModalDialog, useToaster } from '@harnessio/uicore'
import { type ErrorV3, useCopyResourcesV3Mutation } from '@harnessio/react-har-service-client'
import { useAppStore, useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { queryClient } from '@ar/utils/queryClient'
import type { RepositoryConfigType } from '@ar/common/types'

import { CloneVersionModalContent } from './CloneVersionModalContent'

interface UseCloneVersionModalProps {
  repoKey: string
  artifactKey: string
  versionKey: string
  /** Source version UUID – required for v3 copy API */
  versionUuid?: string
  registryType?: RepositoryConfigType
  /** Package type of the source registry (e.g. NPM, DOCKER) – used to filter target registries */
  packageType?: string
  onSuccess?: () => void
}

export default function useCloneVersionModal(props: UseCloneVersionModalProps) {
  const { repoKey, artifactKey, versionKey, versionUuid, registryType, packageType, onSuccess } = props
  const { getString } = useStrings()
  const { showSuccess, showError, clear } = useToaster()
  const { scope } = useAppStore()
  const { useModalHook } = useParentHooks()

  const { mutateAsync: copyResourcesV3, isLoading: loading } = useCopyResourcesV3Mutation()

  const handleClone = async (target: {
    organization: string
    project: string
    registry: string
    registryUuid: string
  }): Promise<void> => {
    if (!versionUuid) return
    try {
      await copyResourcesV3({
        queryParams: {
          account_identifier: scope?.accountId ?? ''
        },
        body: {
          source: { versionId: versionUuid },
          target: { registryId: target.registryUuid }
        }
      })
      clear()
      showSuccess(getString('versionList.messages.copyVersionSuccess'))
      onSuccess?.()
      hideModal()
      queryClient.invalidateQueries(['GetAllHarnessArtifacts'])
      queryClient.invalidateQueries(['ListVersions'])
    } catch (e: any) {
      clear()
      const err = e as ErrorV3
      const message =
        err?.error?.message ||
        getErrorInfoFromErrorObject(e, true) ||
        getString('versionList.messages.copyVersionFailed')
      showError(message)
    }
  }

  const [showModal, hideModal] = useModalHook(
    () => (
      <ModalDialog
        isOpen
        enforceFocus={false}
        canEscapeKeyClose
        canOutsideClickClose
        onClose={hideModal}
        title={getString('versionDetails.copyVersionModal.title')}
        isCloseButtonShown
        showOverlay={loading}
        height={750}
        width={600}>
        <CloneVersionModalContent
          registryName={repoKey}
          packageName={artifactKey}
          version={versionKey}
          registryType={registryType}
          packageType={packageType}
          onSubmit={handleClone}
          onClose={hideModal}
          disabled={loading}
        />
      </ModalDialog>
    ),
    [repoKey, artifactKey, versionKey, versionUuid, registryType, packageType, loading, onSuccess]
  )

  return { openCloneVersionModal: showModal }
}
