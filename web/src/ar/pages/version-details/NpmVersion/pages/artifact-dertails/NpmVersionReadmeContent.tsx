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
import { Container } from '@harnessio/uicore'
import ReadmeFileContent from '@ar/pages/version-details/components/ReadmeFileContent/ReadmeFileContent'
import { MOCK_README_CONTENT } from './mockData'

export default function NpmVersionReadmeContent() {
  return (
    <Container>
      <ReadmeFileContent source={MOCK_README_CONTENT} />
    </Container>
  )
}
