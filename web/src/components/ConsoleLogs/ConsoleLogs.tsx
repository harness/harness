import { Layout, Text } from '@harnessio/uicore'
import React, { FC } from 'react'
import css from './ConsoleLogs.module.scss'

// currently a string - should be an array of strings in future
interface ConsoleLogsProps {
  logs: string
}

interface log {
  pos: number
  out: string
  time: number
}

const convertStringToLogArray = (logs: string): log[] => {
  const logStrings = logs.split('\n').map(log => {
    return JSON.parse(log)
  })

  return logStrings
}

const ConsoleLogs: FC<ConsoleLogsProps> = ({ logs }) => {
  const logArray = convertStringToLogArray(logs)
  return logArray.map((log, index) => {
    return (
      <Layout.Horizontal key={index} spacing={'medium'} className={css.logLayout}>
        <Text className={css.lineNumber}>{log.pos}</Text>
        <Text className={css.log}>{log.out}</Text>
      </Layout.Horizontal>
    )
  })
}

export default ConsoleLogs
