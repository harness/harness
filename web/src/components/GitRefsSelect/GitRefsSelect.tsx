import React from 'react'
import { Container, Layout, Icon, Button, ButtonVariation } from '@harness/uicore'
import { Popover, PopoverInteractionKind, Position } from '@blueprintjs/core'

export function GitRefsSelect() {
  const popoverContent = (
    <Container padding="large">
      <h5>Popover Title</h5>
      <p>...</p>
      <button className="bp3-button bp3-popover-dismiss">Close popover</button>
    </Container>
  )
  return (
    <Popover
      content={popoverContent}
      interactionKind={PopoverInteractionKind.CLICK}
      minimal
      usePortal
      // isOpen={this.state.isOpen}
      // onInteraction={state => this.handleInteraction(state)}
      position={Position.BOTTOM_LEFT}>
      <Button icon="git-branch" iconProps={{ size: 13 }} variation={ButtonVariation.TERTIARY}>
        <Layout.Horizontal spacing="small">
          <span>With right button</span>
          <Icon name="main-chevron-down" size={8} />
        </Layout.Horizontal>
      </Button>
    </Popover>
  )
}

// TODO: Optimize branch fetching when first fetch return less than request LIMIT
