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
import { Layout, Text } from '@harnessio/uicore'
import { Menu, MenuItem } from '@blueprintjs/core'
import { Code } from 'iconoir-react'
import { IDEType } from 'cde-gitness/constants'
import vsCodeWebIcon from 'cde-gitness/assests/vsCodeWeb.svg?url'
import vscodeIcon from 'cde-gitness/assests/VSCode.svg?url'
import { useStrings } from 'framework/strings'
import { CDECustomDropdown } from '../CDECustomDropdown/CDECustomDropdown'

export const CDEIDESelect = ({
  onChange,
  selectedIde
}: {
  onChange: (field: string, value: any) => void
  selectedIde?: string
}) => {
  const { getString } = useStrings()

  const ideOptions = [
    {
      label: getString('cde.ide.desktop'),
      value: IDEType.VSCODE,
      img: vscodeIcon
    },
    {
      label: getString('cde.ide.browser'),
      value: IDEType.VSCODEWEB,
      img: vsCodeWebIcon
    }
  ]

  const { label, img } = ideOptions.find(item => item.value === selectedIde) || {}

  return (
    <CDECustomDropdown
      leftElement={
        <Layout.Horizontal>
          <Code height={20} width={20} style={{ marginRight: '8px', alignItems: 'center' }} />
          <Layout.Vertical spacing="small">
            <Text>IDE</Text>
            <Text font="small">Your Gitspace will open in the selected IDE to code</Text>
          </Layout.Vertical>
        </Layout.Horizontal>
      }
      label={
        <Layout.Horizontal width="100%" flex={{ alignItems: 'center', justifyContent: 'space-between' }}>
          <img height={16} width={16} src={img} />
          <Text>{label}</Text>
        </Layout.Horizontal>
      }
      menu={
        <Menu>
          {ideOptions.map(item => {
            return (
              <MenuItem
                key={item.value}
                text={
                  <Layout.Horizontal width="90%" flex={{ alignItems: 'center', justifyContent: 'space-between' }}>
                    <img height={16} width={16} src={item.img} />
                    <Text>{item.label}</Text>
                  </Layout.Horizontal>
                }
                onClick={() => {
                  onChange('ide', item.value)
                }}
              />
            )
          })}
        </Menu>
      }
    />
  )

  // return (
  //   <Layout.Horizontal className={css.main}>
  //     <Container width="70%">
  //       <Layout.Vertical>
  //         <Text icon="code">IDE</Text>
  //         <Text margin={{ left: 'large' }} font="small">
  //           Your Gitspace will open in the selected IDE to code
  //         </Text>
  //       </Layout.Vertical>
  //     </Container>
  //     <Container width="30%">
  //       <Button
  //         height="45px"
  //         width="100%"
  //         className={css.button}
  //         text={
  //           <Layout.Horizontal width="90%" flex={{ alignItems: 'center', justifyContent: 'space-between' }}>
  //             <img height={16} width={16} src={img} />
  //             <Text>{label}</Text>
  //           </Layout.Horizontal>
  //         }
  //         rightIcon={'chevron-down'}
  //         variation={ButtonVariation.TERTIARY}
  //         iconProps={{ size: 14 }}
  //         tooltipProps={{
  //           fill: true,
  //           interactionKind: PopoverInteractionKind.CLICK,
  //           position: PopoverPosition.BOTTOM_LEFT,
  //           popoverClassName: css.popover
  //         }}
  //         tooltip={
  //           <Container className={css.listContainer} width="100%">
  //             <Menu>
  //               {ideOptions.map(item => {
  //                 return (
  //                   <MenuItem
  //                     key={item.value}
  //                     text={
  //                       <Layout.Horizontal>
  //                         <img height={16} width={16} src={item.img} />
  //                         <Text>{item.label}</Text>
  //                       </Layout.Horizontal>
  //                     }
  //                     onClick={() => {
  //                       onChange('ide', item.value)
  //                     }}
  //                   />
  //                 )
  //               })}
  //             </Menu>
  //           </Container>
  //         }
  //       />
  //     </Container>
  //   </Layout.Horizontal>
  // )
}
