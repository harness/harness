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
import { Container, PageBody } from '@harnessio/uicore'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { useAppContext } from 'AppContext'
import { WehookForm } from './WehookForm'

export default function WebhookNew() {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const { repoMetadata, error, loading } = useGetRepositoryMetadata()

  return (
    <Container>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('createAWebhook')}
        dataTooltipId="webhooks"
        extraBreadcrumbLinks={
          repoMetadata && [
            {
              label: getString('webhooks'),
              url: routes.toCODEWebhooks({ repoPath: repoMetadata.path as string })
            }
          ]
        }
      />
      <PageBody error={error}>
        <LoadingSpinner visible={loading} />
        {repoMetadata && <WehookForm repoMetadata={repoMetadata} />}
      </PageBody>
    </Container>
  )
}
