import React, { useCallback, useEffect, useState } from 'react'
import { Container } from '@harnessio/uicore'
import cx from 'classnames'
import { GitspaceStatus } from 'cde-gitness/constants'
import { lineElement } from 'components/LogViewer/LogViewer'
import { useScheduleJob } from 'hooks/useScheduleJob'
import { useAppContext } from 'AppContext'
import LogStreaming from './LogStreaming'
import css from './Logger.module.scss'

interface LoggerProps {
  stepNameLogKeyMap?: Map<string, string>
  expanded?: boolean
  logKey: string
  state: string
  value: string
  isStreaming: boolean
  localRef: any
  setIsBottom: (val: boolean) => void
}

const Logger: React.FC<LoggerProps> = ({ expanded, logKey, value, state, isStreaming, localRef, setIsBottom }) => {
  const logKeyList: string[] = [logKey]
  const { hooks } = useAppContext()
  const [startStreaming, setStartStreaming] = useState(false)
  const { getBlobData, blobDataCur } = hooks?.useLogsContent(logKeyList)

  const sendStreamLogToRenderer = useScheduleJob({
    handler: useCallback((blocks: string[]) => {
      const logContainer = localRef.current as HTMLDivElement

      if (logContainer) {
        const fragment = new DocumentFragment()

        blocks.forEach((block: string) => {
          const blockData = JSON.parse(block)
          const linePos = blockData.pos + 1
          const localDate = new Date(blockData.time)
          const formattedDate = localDate.toLocaleString()

          fragment.appendChild(
            lineElement(`${linePos}  ${blockData.level}  ${formattedDate.replace(',', '')}  ${blockData.out}`)
          )
        })

        logContainer?.appendChild(fragment)

        const scrollParent = logContainer?.parentElement as HTMLDivElement
        scrollParent.scrollTop = scrollParent?.scrollHeight
        setIsBottom(true)
      }
    }, []),
    isStreaming: true
  })

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const onMessageStreaming = (e: any) => {
    if (e.data) {
      sendStreamLogToRenderer(e.data || '')
    }
  }

  const onError = (e: any) => {
    // eslint-disable-next-line no-console
    console.log(e)
  }

  const getLogData = async () => {
    await getBlobData(logKeyList)
  }

  useEffect(() => {
    if (expanded && (state === GitspaceStatus.RUNNING || state === GitspaceStatus.STOPPED)) {
      // Fetch from blob
      getLogData()
    } else if (expanded && state !== GitspaceStatus.RUNNING && state !== GitspaceStatus.STOPPED) {
      if (isStreaming) {
        setStartStreaming(true)
      } else {
        setStartStreaming(false)
      }
    }
  }, [state, isStreaming, expanded])

  useEffect(() => {
    if (blobDataCur && (state === GitspaceStatus.RUNNING || state === GitspaceStatus.STOPPED)) {
      const logData = JSON.parse(blobDataCur)?.map((logs: { level: string; time: string }) => {
        return JSON.stringify(logs)
      })
      sendStreamLogToRenderer(logData || '')
    }
  }, [blobDataCur])

  return (
    <>
      {startStreaming ? (
        <LogStreaming logKeyList={logKeyList} onMessageStreaming={onMessageStreaming} onError={onError} />
      ) : null}
      <Container key={`harnesslog_${value}`} ref={localRef} className={cx(css.main, css.stepLogContainer)} />
    </>
  )
}

export default Logger
