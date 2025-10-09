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
import { Container, Layout, Text, FlexExpander } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { ButtonGroup, Button as BButton, Classes } from '@blueprintjs/core'
import cx from 'classnames'
import { useStrings } from 'framework/strings'
import { ButtonRoleProps } from 'utils/Utils'
import { ViewStyle } from 'components/DiffViewer/DiffViewerUtils'

interface DiffViewConfigurationProps {
  viewStyle: ViewStyle
  lineBreaks: boolean
  setViewStyle: (val: ViewStyle) => void
  setLineBreaks: (val: boolean) => void
}

export const DiffViewConfiguration: React.FC<DiffViewConfigurationProps> = ({
  viewStyle,
  setViewStyle
  // lineBreaks,
  // setLineBreaks
}) => {
  const { getString } = useStrings()

  return (
    <Text
      icon="cog"
      rightIcon="caret-down"
      tag="span"
      tooltip={
        <Container padding="large">
          <Layout.Vertical spacing="small">
            <Container width={250}>
              <Layout.Horizontal spacing="xsmall" flex={{ alignItems: 'center' }}>
                <Text font={{ variation: FontVariation.SMALL_BOLD }}>{getString('pr.diffView')}</Text>
                <FlexExpander />
                <ButtonGroup>
                  <BButton
                    className={cx(Classes.POPOVER_DISMISS, viewStyle === ViewStyle.SIDE_BY_SIDE ? Classes.ACTIVE : '')}
                    onClick={() => {
                      setViewStyle(ViewStyle.SIDE_BY_SIDE)
                      window.scroll({ top: 0 })
                    }}>
                    {getString('pr.split')}
                  </BButton>
                  <BButton
                    className={cx(Classes.POPOVER_DISMISS, viewStyle === ViewStyle.LINE_BY_LINE ? Classes.ACTIVE : '')}
                    onClick={() => {
                      setViewStyle(ViewStyle.LINE_BY_LINE)
                      window.scroll({ top: 0 })
                    }}>
                    {getString('pr.unified')}
                  </BButton>
                </ButtonGroup>
              </Layout.Horizontal>
            </Container>
            {/* 
            // TODO: Line break barely works. Disable until we find a complete solution for it
            // https://harness.atlassian.net/browse/CODE-1452
            // [css.enableDiffLineBreaks]: lineBreaks && viewStyle === ViewStyle.SIDE_BY_SIDE
            <Container>
              <Layout.Horizontal spacing="xsmall" flex={{ alignItems: 'center' }}>
                <Text font={{ variation: FontVariation.SMALL_BOLD }}>{getString('lineBreaks')}</Text>
                <FlexExpander />
                <ButtonGroup>
                  <BButton
                    className={cx(Classes.POPOVER_DISMISS, lineBreaks ? Classes.ACTIVE : '')}
                    onClick={() => setLineBreaks(true)}>
                    {getString('on')}
                  </BButton>
                  <BButton
                    className={cx(Classes.POPOVER_DISMISS, !lineBreaks ? Classes.ACTIVE : '')}
                    onClick={() => setLineBreaks(false)}>
                    {getString('off')}
                  </BButton>
                </ButtonGroup>
              </Layout.Horizontal>
            </Container> */}
          </Layout.Vertical>
        </Container>
      }
      tooltipProps={{ interactionKind: 'click' }}
      iconProps={{ size: 14, padding: { right: 3 } }}
      rightIconProps={{ size: 13, padding: { left: 0 } }}
      padding={{ left: 'small' }}
      {...ButtonRoleProps}
    />
  )
}
