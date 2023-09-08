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
          <Layout.Horizontal key={index} spacing={'medium'} className={css.logLayout}>
            <Text className={css.lineNumber}>{log.pos}</Text>
            <Text className={css.log}>{log.out}</Text>
            <FlexExpander />
            <Text>{log.time}s</Text>
          </Layout.Horizontal>
        )
      })}
    </>
  )
}

export default ConsoleLogs
