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
import { Container, NoDataCard } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import noExecutionImage from '../../pages/RepositoriesListing/no-repo.svg?url'
import css from './NoExecutionsCard.module.scss'

interface NoResultCardProps {
  showWhen?: () => boolean
}

export const NoExecutionsCard: React.FC<NoResultCardProps> = ({ showWhen = () => true }) => {
  const { getString } = useStrings()

  if (!showWhen()) {
    return null
  }

  return (
    <Container className={css.main}>
      <NoDataCard image={noExecutionImage} message={getString('executions.noData')} />
    </Container>
  )
}
