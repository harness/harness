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
import { Container, Layout, Button, ButtonVariation } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
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
