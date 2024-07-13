import { useEffect, useState } from 'react'

export interface LogData {
  pos: number
  out: string
  time: number
}

export function parseLog(log: string): LogData[] {
  const logLines = log.trim().split('\n\n')
  const parsedData: LogData[] = []

  logLines.forEach(line => {
    const dataMatch = line.match(/data: (.+)/)

    if (dataMatch && dataMatch[1] !== 'eof') {
      const eventData: LogData = JSON.parse(dataMatch[1])

      parsedData.push(eventData)
    }
  })

  return parsedData
}

export const useGetLogStream = ({ response }: { response: any }) => {
  const [data, setData] = useState('')

  useEffect(() => {
    const fetchStreamData = async () => {
      const reader = response?.body?.getReader()
      const decoder = new TextDecoder()
      let done = false

      while (!done) {
        /* eslint-disable no-await-in-loop */
        const { value, done: streamDone } = (await reader?.read()) || {}
        done = streamDone
        const chunk = decoder.decode(value)
        setData(prevData => prevData + chunk)
      }
    }

    try {
      if (response && !response.body.locked) {
        fetchStreamData()
      }
    } catch (error) {
      //
    }
  }, [response])

  return { data: parseLog(data) }
}
