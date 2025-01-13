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
import {
  Container,
  Button,
  ButtonVariation,
  Dialog,
  Layout,
  Text,
  useToaster,
  StringSubstitute
} from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useUpdateRepository } from 'services/code'
import { useModalHook } from 'hooks/useModalHook'
import { useStrings } from 'framework/strings'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { getErrorMessage } from 'utils/Utils'
import css from '../../RepositorySettings.module.scss'

interface ArchiveRepoModalProps {
  refetch: () => void
}

const useArchiveRepoModal = ({ refetch: refetchMetaData }: ArchiveRepoModalProps) => {
  const { repoMetadata } = useGetRepositoryMetadata()
  const { getString } = useStrings()
  const { mutate: archiveRepository, loading: archivingRepository } = useUpdateRepository({
    repo_ref: `${repoMetadata?.path as string}/+`
  })
  const { showSuccess, showError } = useToaster()
  const [openModal, hideModal] = useModalHook(() => {
    const onClose = () => {
      hideModal()
    }
    return (
      <Dialog
        className={css.dialogContainer}
        isOpen
        enforceFocus={false}
        onClose={onClose}
        title={
          <Text font={{ variation: FontVariation.H4 }}>
            {repoMetadata?.archived === true
              ? getString('repoArchive.titleUnarchive')
              : getString('repoArchive.titleArchive')}
          </Text>
        }>
        <Layout.Vertical spacing="xlarge">
          <Container
            intent="warning"
            background="yellow100"
            border={{
              color: 'orange500'
            }}
            margin={{ top: 'medium', bottom: 'medium' }}>
            <Text
              icon="warning-outline"
              iconProps={{ size: 16, margin: { right: 'small' } }}
              padding={{ left: 'large', right: 'large', top: 'small', bottom: 'small' }}
              color={Color.WARNING}>
              {repoMetadata?.archived === true
                ? getString('repoArchive.unarchiveWarning')
                : getString('repoArchive.archiveWarning')}
            </Text>
          </Container>
          <Layout.Horizontal className={css.buttonContainer}>
            <Button
              type="submit"
              variation={ButtonVariation.PRIMARY}
              margin={{ right: 'small' }}
              onClick={async () => {
                try {
                  await archiveRepository({ state: repoMetadata?.archived ? 0 : 4 })
                  showSuccess(getString('repoUpdate'))
                  hideModal()
                  refetchMetaData()
                } catch (exception) {
                  showError(getErrorMessage(exception), 0, 'failed to archive the repository')
                }
                refetchMetaData()
              }}>
              <StringSubstitute
                str={getString('repoArchive.confirmButton')}
                vars={{
                  archiveVerb: <span className={css.text}>{repoMetadata?.archived ? 'unarchive' : 'archive'}</span>
                }}
              />
            </Button>
            <Button
              text={getString('cancel')}
              variation={ButtonVariation.TERTIARY}
              onClick={() => {
                hideModal()
              }}
            />
          </Layout.Horizontal>
        </Layout.Vertical>
      </Dialog>
    )
  }, [archivingRepository, repoMetadata])

  return {
    openModal,
    hideModal
  }
}

export default useArchiveRepoModal
