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
import { Container, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import css from './RepositoryArchivedBanner.module.scss'

export const RepoArchivedBanner: React.FC<{ isArchived?: boolean; updated?: number }> = ({ isArchived, updated }) => {
  const { getString } = useStrings()

  return (
    <>
      {isArchived && (
        <Container className={css.infoContainer}>
          <Text
            icon="main-issue"
            iconProps={{ size: 16, color: Color.ORANGE_700, margin: { right: 'small' } }}
            color={Color.GREY_700}>
            {getString('repoArchive.infoText', {
              date: updated
                ? new Intl.DateTimeFormat('en-US', {
                    day: '2-digit',
                    month: 'short',
                    year: 'numeric'
                  }).format(new Date(updated))
                : 'N/A'
            })}
          </Text>
        </Container>
      )}
    </>
  )
}
