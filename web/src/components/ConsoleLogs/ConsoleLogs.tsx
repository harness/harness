import { FlexExpander, Layout, Text } from '@harnessio/uicore'
import React, { FC } from 'react'
import type { LivelogLine } from 'services/code'
import css from './ConsoleLogs.module.scss'

interface ConsoleLogsProps {
  logs: LivelogLine[]
}

const ConsoleLogs: FC<ConsoleLogsProps> = ({ logs }) => {
  return (
    <>
      {logs.map((log, index) => {
        return (
          <Layout.Horizontal key={index} spacing={'small'} className={css.logLayout}>
            {typeof log.pos === 'number' && <Text className={css.lineNumber}>{log.pos + 1}</Text>}
            <Text className={css.log}>{log.out}</Text>
            <FlexExpander />
            <Text className={css.time}>{log.time}s</Text>
          </Layout.Horizontal>
        )
      })}
    </>
  )
}

export default ConsoleLogs
