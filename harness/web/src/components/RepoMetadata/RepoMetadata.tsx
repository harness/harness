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
import { Container, Layout, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'

export function RepoMetadata() {
  return (
    <Container width="70%">
      <Layout.Horizontal spacing="large">
        <Text icon="dot" iconProps={{ size: 20, color: Color.BLUE_500 }}>
          Java
        </Text>
        <Text color={Color.GREY_200}>{' | '}</Text>
        <Text icon="git-new-branch">165</Text>
        <Text icon="git-branch-existing">123</Text>
        <Text icon="git-merge">432</Text>
      </Layout.Horizontal>
    </Container>
  )
}
