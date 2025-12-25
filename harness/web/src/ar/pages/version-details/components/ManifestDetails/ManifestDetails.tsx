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
import { FontVariation } from '@harnessio/design-system'
import { Card, Container, Layout, Text } from '@harnessio/uicore'
import { useStrings } from '@ar/frameworks/strings/String'
import { prettifyManifestJSON } from '@ar/pages/version-details/utils'
import CommandBlock from '@ar/components/CommandBlock/CommandBlock'

interface ManifestDetailsProps {
  manifest: string
  className?: string
}

export default function ManifestDetails(props: ManifestDetailsProps) {
  const { manifest, className } = props
  const { getString } = useStrings()
  return (
    <Card className={className}>
      <Layout.Vertical spacing="large">
        <Text font={{ variation: FontVariation.CARD_TITLE }}>
          {getString('versionDetails.artifactDetails.tabs.manifest')}
        </Text>
        <Container>
          {manifest && (
            <CommandBlock ignoreWhiteSpaces={false} commandSnippet={prettifyManifestJSON(manifest)} allowCopy />
          )}
        </Container>
      </Layout.Vertical>
    </Card>
  )
}
