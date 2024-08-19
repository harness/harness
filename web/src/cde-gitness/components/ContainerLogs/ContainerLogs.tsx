import { Container } from '@harnessio/uicore'
import cx from 'classnames'
import React, { useEffect, useRef } from 'react'
import { lineElement } from 'components/LogViewer/LogViewer'
import { useAppContext } from 'AppContext'
import type { LogData } from '../../hooks/useGetLogStream'
import { parseLogString } from './ContainerLogs.utils'
import css from './ContainerLogs.module.scss'

const ContainerLogs = ({ data }: { data: LogData[] }) => {
  const { standalone } = useAppContext()
  const localRef = useRef<HTMLDivElement | null>()

  useEffect(() => {
    try {
      if (data) {
        const fragment = new DocumentFragment()
        const logContainer = localRef.current as HTMLDivElement
        // Clear the container first
        if (localRef.current) {
          localRef.current.innerHTML = ''
        }
        if (data) {
          const logData = standalone ? data : parseLogString(data as unknown as string)
          logData?.forEach((line: any) => {
            const linePos = line.pos + 1
            const localDate = new Date(line.time)
            // Format date to a more readable format (local time)
            const formattedDate = localDate.toLocaleString()
            fragment.appendChild(
              lineElement(`${linePos} ${formattedDate.replace(',', '')}  ${standalone ? line.out : line.message}`)
            )
          })

          logContainer.appendChild(fragment)
        }

        const scrollParent = logContainer.parentElement as HTMLDivElement
        const autoScroll =
          scrollParent && scrollParent.scrollTop === scrollParent.scrollHeight - scrollParent.offsetHeight

        if (autoScroll || scrollParent.scrollTop === 0) {
          scrollParent.scrollTop = scrollParent.scrollHeight
        }
      }
    } catch (_err) {
      //
    }
  }, [data])

  return (
    <Container className={css.consoleContainer}>
      <Container key={`harnesslog`} ref={localRef} className={cx(css.mainLog, css.stepLogContainer)} />
    </Container>
  )
}

export default ContainerLogs
