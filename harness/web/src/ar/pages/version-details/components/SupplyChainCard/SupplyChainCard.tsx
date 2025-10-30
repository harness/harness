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
import { Button, ButtonSize, ButtonVariation, Card, Layout, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'

import { killEvent } from '@ar/common/utils'
import { useStrings } from '@ar/frameworks/strings'

import useDownloadSBOM from '../../hooks/useDownloadSBOM'
import SecurityItem from '../SecurityTestsCard/SecurityItem'
import { SecurityTestSatus } from '../SecurityTestsCard/types'

import css from './SupplyChainCard.module.scss'

interface SupplyChainCardProps {
  className?: string
  title?: string
  totalComponents: number
  allowListCount: number
  denyListCount: number
  sbomScore: string | number
  orchestrationId: string
  onClick?: () => void
}

export default function SupplyChainCard(props: SupplyChainCardProps) {
  const { title, totalComponents, className, sbomScore, onClick, orchestrationId } = props
  const { getString } = useStrings()

  const { download, loading } = useDownloadSBOM()

  return (
    <Card data-testid="integration-supply-chain-card" className={className} onClick={onClick}>
      <Layout.Vertical>
        <Text font={{ variation: FontVariation.CARD_TITLE }}>
          {title ?? getString('versionDetails.cards.supplyChain.title')}
        </Text>
        <Layout.Vertical className={css.container}>
          <Layout.Horizontal spacing="small" flex={{ alignItems: 'flex-end', justifyContent: 'flex-start' }}>
            <Text font={{ variation: FontVariation.H2 }}>{totalComponents}</Text>
            <Text color={Color.GREY_500} font={{ variation: FontVariation.SMALL }}>
              {getString('versionDetails.cards.supplyChain.totalComponents')}
            </Text>
          </Layout.Horizontal>
          <Layout.Horizontal spacing="small">
            <SecurityItem
              title={getString('versionDetails.cards.supplyChain.sbomScore')}
              value={sbomScore}
              status={SecurityTestSatus.Green}
              toPrecision={2}
            />
            <Button
              className={css.downloadSlsaBtn}
              size={ButtonSize.SMALL}
              rightIcon="download-manifests"
              variation={ButtonVariation.LINK}
              loading={loading}
              disabled={!orchestrationId}
              onClick={evt => {
                killEvent(evt)
                download(orchestrationId)
              }}>
              {getString('versionDetails.cards.supplyChain.downloadSbom')}
            </Button>
          </Layout.Horizontal>
        </Layout.Vertical>
      </Layout.Vertical>
    </Card>
  )
}
