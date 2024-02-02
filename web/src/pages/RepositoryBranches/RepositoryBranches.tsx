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
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { voidFn, getErrorMessage } from 'utils/Utils'
import { RepositoryBranchesContent } from './RepositoryBranchesContent/RepositoryBranchesContent'
import css from './RepositoryBranches.module.scss'

export default function RepositoryBranches() {
  const { getString } = useStrings()
  const { repoMetadata, error, refetch } = useGetRepositoryMetadata()

  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('branches')}
        dataTooltipId="repositoryBranches"
      />
      <PageBody error={getErrorMessage(error)} retryOnError={voidFn(refetch)}>
        <RepositoryBranchesContent repoMetadata={repoMetadata} />
      </PageBody>
    </Container>
  )
}
