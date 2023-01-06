import React from 'react'
import { Container, Layout, Text, FontVariation, FlexExpander } from '@harness/uicore'
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
  setViewStyle,
  lineBreaks,
  setLineBreaks
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
            </Container>
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
