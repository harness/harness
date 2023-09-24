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
import { Container, PageError } from '@harnessio/uicore'
import { getErrorMessage } from 'utils/Utils'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'

interface TabContentWrapperProps {
  className?: string
  loading?: boolean
  error?: Unknown
  onRetry: () => void
}

export const TabContentWrapper: React.FC<TabContentWrapperProps> = ({
  className,
  loading,
  error,
  onRetry,
  children
}) => {
  return (
    <Container className={className} padding="xlarge">
      <LoadingSpinner visible={loading} withBorder={true} />
      {error && <PageError message={getErrorMessage(error)} onClick={onRetry} />}
      {!error && children}
    </Container>
  )
}
