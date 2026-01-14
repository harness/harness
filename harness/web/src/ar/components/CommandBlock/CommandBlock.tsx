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
import cx from 'classnames'
import { Color, FontVariation } from '@harnessio/design-system'
import { Layout, Text, Button, ButtonVariation } from '@harnessio/uicore'
import type { ClientSetupStepCommand } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings/String'
import CopyButton from '@ar/components/CopyButton/CopyButton'

import css from './CommandBlock.module.scss'

interface DownloadFileProps {
  downloadFileExtension?: string
  downloadFileName?: string
}

interface CommandBlockProps {
  commandSnippet: string | ClientSetupStepCommand[]
  allowCopy?: boolean
  ignoreWhiteSpaces?: boolean
  allowDownload?: boolean
  downloadFileProps?: DownloadFileProps
  copyButtonText?: string
  darkmode?: boolean
  onCopy?: (evt: React.MouseEvent<Element, MouseEvent>) => void
  lineClamp?: number
  noWrap?: boolean
}
enum DownloadFile {
  DEFAULT_NAME = 'commandBlock',
  DEFAULT_TYPE = 'txt'
}

const combineCommands = (list: ClientSetupStepCommand[]): string => {
  return list
    .map(cmd => {
      return cmd.label || cmd.value
    })
    .join('\n')
}

const getCommands = (commandSnippet: string | ClientSetupStepCommand[]): ClientSetupStepCommand[] => {
  if (typeof commandSnippet === 'string') {
    return [
      {
        label: '',
        value: commandSnippet
      }
    ]
  } else {
    return commandSnippet
  }
}

const CommandBlock: React.FC<CommandBlockProps> = ({
  commandSnippet,
  allowCopy,
  ignoreWhiteSpaces = true,
  noWrap = false,
  allowDownload = false,
  downloadFileProps,
  copyButtonText,
  darkmode,
  lineClamp,
  onCopy
}): JSX.Element => {
  const downloadFileDefaultName = downloadFileProps?.downloadFileName || DownloadFile.DEFAULT_NAME
  const downloadeFileDefaultExtension =
    (downloadFileProps && downloadFileProps.downloadFileExtension) || DownloadFile.DEFAULT_TYPE
  const linkRef = React.useRef<HTMLAnchorElement>(null)
  const commands = getCommands(commandSnippet).filter(cmd => cmd.label || cmd.value)

  const { getString } = useStrings()
  const onDownload = (): void => {
    const content = new Blob([combineCommands(commands) as BlobPart], { type: 'data:text/plain;charset=utf-8' })
    if (linkRef?.current) {
      linkRef.current.href = window.URL.createObjectURL(content)
      linkRef.current.download = `${downloadFileDefaultName}.${downloadeFileDefaultExtension}`
      linkRef.current.click()
    }
  }
  return (
    <Layout.Horizontal
      flex={{ justifyContent: 'space-between', alignItems: 'center' }}
      spacing="medium"
      className={cx(css.commandBlock, { [css.darkmode]: darkmode })}>
      <Layout.Vertical flex={{ alignItems: 'flex-start' }} className={css.commandGroup}>
        {commands.map((command, i) => {
          const { label: cmdLabel = '', value: cmdValue = '' } = command
          return (
            <Layout.Horizontal
              key={i}
              flex={{ justifyContent: 'space-between', alignItems: 'flex-start' }}
              className={css.commandRow}>
              <Text
                color={darkmode ? Color.WHITE : undefined}
                className={cx([
                  css.commandText,
                  {
                    [css.ignoreWhiteSpaces]: !ignoreWhiteSpaces,
                    [css.noWrap]: noWrap
                  }
                ])}
                font={{ variation: FontVariation.YAML }}
                lineClamp={lineClamp}>
                {cmdLabel || cmdValue}
              </Text>
              {allowCopy && cmdValue && (
                <CopyButton
                  textToCopy={cmdValue}
                  text={copyButtonText}
                  onCopySuccess={evt => {
                    onCopy?.(evt)
                  }}
                  primaryBtn={darkmode}
                  className={cx({ [css.copyButtonHover]: darkmode })}
                />
              )}
            </Layout.Horizontal>
          )
        })}
      </Layout.Vertical>

      {allowDownload && (
        <>
          <Button
            className={css.downloadBtn}
            variation={ButtonVariation.LINK}
            text={getString('download')}
            onClick={event => {
              event.stopPropagation()
              onDownload()
            }}
          />
          <a className={css.downloadAnchor} ref={linkRef} target={'_blank'} />
        </>
      )}
    </Layout.Horizontal>
  )
}

export default CommandBlock
