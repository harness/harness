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

import { Intent } from '@blueprintjs/core'
import { getErrorInfoFromErrorObject, useToaster } from '@harnessio/uicore'

import { useGetSpaceRef, useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { TypesLabel, useDeleteSpaceLabel } from 'services/code'

interface useDeleteLabelModalProps {
  data: TypesLabel
  onSuccess: () => void
}
export default function useDeleteLabelModal(props: useDeleteLabelModalProps) {
  const { data, onSuccess } = props
  const { getString } = useStrings()
  const { showSuccess, showError, clear } = useToaster()
  const { useConfirmationDialog } = useParentHooks()
  const spaceRef = useGetSpaceRef()

  const { mutate: deleteLabel } = useDeleteSpaceLabel({
    space_ref: spaceRef
  })

  const handleDeleteLabel = async (isConfirmed: boolean): Promise<void> => {
    if (isConfirmed) {
      try {
        await deleteLabel(data.key || '')
        clear()
        showSuccess(getString('labelsList.deleteLabelModal.labelDeleted'))
        onSuccess()
      } catch (e: any) {
        showError(getErrorInfoFromErrorObject(e, true))
      }
    }
  }

  const { openDialog } = useConfirmationDialog({
    titleText: getString('labelsList.deleteLabelModal.title'),
    contentText: getString('labelsList.deleteLabelModal.contentText'),
    confirmButtonText: getString('delete'),
    cancelButtonText: getString('cancel'),
    intent: Intent.DANGER,
    onCloseDialog: handleDeleteLabel
  })

  return { triggerDelete: openDialog }
}
