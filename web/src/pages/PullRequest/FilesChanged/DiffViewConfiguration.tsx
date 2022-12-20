import React from 'react'
import { Container, Layout, Text, FontVariation } from '@harness/uicore'
import { ButtonGroup, Button as BButton, Classes } from '@blueprintjs/core'
import cx from 'classnames'
import 'highlight.js/styles/github.css'
import 'diff2html/bundles/css/diff2html.min.css'
import { useStrings } from 'framework/strings'
import { ButtonRoleProps } from 'utils/Utils'
import { ViewStyle } from 'components/DiffViewer/DiffViewerUtils'

export const DiffViewConfiguration: React.FC<{ viewStyle: ViewStyle; setViewStyle: (val: ViewStyle) => void }> = ({
  viewStyle,
  setViewStyle
}) => {
  const { getString } = useStrings()

  return (
    <Text
      icon="cog"
      rightIcon="caret-down"
      tooltip={
        <Container padding="large">
          <Layout.Horizontal spacing="xsmall" flex={{ alignItems: 'center' }}>
            <Text width={100} font={{ variation: FontVariation.SMALL_BOLD }}>
              {getString('pr.diffView')}
            </Text>
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
      }
      tooltipProps={{ interactionKind: 'click' }}
      iconProps={{ size: 14, padding: { right: 3 } }}
      rightIconProps={{ size: 13, padding: { left: 0 } }}
      padding={{ left: 'small' }}
      {...ButtonRoleProps}
    />
  )
}
