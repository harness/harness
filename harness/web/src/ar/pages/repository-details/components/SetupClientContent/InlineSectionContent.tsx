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
import { Color, FontVariation } from '@harnessio/design-system'
import { Container, Layout, Text } from '@harnessio/uicore'
import type { ClientSetupSection, ClientSetupStepConfig } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import CommandBlock from '@ar/components/CommandBlock/CommandBlock'

import { ClientSetupStepTypeEnum } from './types'
import GenerateTokenStep from './GenerateTokenStep'

import css from './SetupClientContent.module.scss'

interface InlineSectionContentProps {
  section: ClientSetupSection & ClientSetupStepConfig
}

export default function InlineSectionContent(props: InlineSectionContentProps) {
  const { section } = props
  const { getString } = useStrings()
  return (
    <Layout.Vertical spacing="medium">
      <Layout.Vertical>
        <Text font={{ variation: FontVariation.CARD_TITLE }}>{section.header}</Text>
        <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_600}>
          {section.secHeader}
        </Text>
      </Layout.Vertical>
      {section.steps?.map((step, stepIndex) => {
        if (step.type === ClientSetupStepTypeEnum.GenerateToken) {
          return <GenerateTokenStep key={stepIndex} step={step} stepIndex={stepIndex} />
        }
        return (
          <Container className={css.stepGridContainer} key={stepIndex}>
            <Text className={css.label} font={{ variation: FontVariation.SMALL_BOLD }}>
              {getString('repositoryDetails.clientSetup.step', { stepIndex: stepIndex + 1 })}
            </Text>
            <Text font={{ variation: FontVariation.SMALL }}>{step.header}</Text>
            {step.commands && (
              <>
                <div />
                <CommandBlock ignoreWhiteSpaces={false} commandSnippet={step.commands} allowCopy={true} />
              </>
            )}
          </Container>
        )
      })}
    </Layout.Vertical>
  )
}
