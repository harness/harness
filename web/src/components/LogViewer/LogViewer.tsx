import React, { useEffect, useRef } from 'react'
import { Container } from '@harnessio/uicore'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { SearchAddon } from 'xterm-addon-search'
import 'xterm/css/xterm.css'
import { useEventListener } from 'hooks/useEventListener'

export type TermRefs = { term: Terminal; fitAddon: FitAddon }

export interface LogViewerProps {
  search?: string
  content: string
  termRefs?: React.MutableRefObject<TermRefs | undefined>
  autoHeight?: boolean
}

const LogTerminal: React.FC<LogViewerProps> = ({ content, termRefs, autoHeight }) => {
  const ref = useRef<HTMLDivElement | null>(null)
  const term = useRef<TermRefs>()

  useEffect(() => {
    if (!term.current) {
      const _term = new Terminal({
        allowTransparency: true,
        disableStdin: true,
        tabStopWidth: 2,
        scrollOnUserInput: false,
        smoothScrollDuration: 0,
        scrollback: 10000
      })

      const searchAddon = new SearchAddon()
      const fitAddon = new FitAddon()

      _term.loadAddon(searchAddon)
      _term.loadAddon(fitAddon)

      _term.open(ref?.current as HTMLDivElement)

      fitAddon.fit()
      searchAddon.activate(_term)

      // disable cursor
      _term.write('\x1b[?25l')

      term.current = { term: _term, fitAddon }

      if (termRefs) {
        termRefs.current = term.current
      }
    }

    return () => {
      if (term.current) {
        if (termRefs) {
          termRefs.current = undefined
        }
        setTimeout(() => term.current?.term.dispose(), 1000)
      }
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    const lines = content.split(/\r?\n/)

    lines.forEach(line => term.current?.term?.writeln(line))

    if (autoHeight) {
      term.current?.term?.resize(term.current?.term?.cols, lines.length + 1)
    }

    setTimeout(() => {
      term.current?.term.scrollToTop()
    }, 0)

    return () => {
      term.current?.term?.clear()
    }
  }, [content, autoHeight])

  useEventListener('resize', () => term.current?.fitAddon?.fit())

  return <Container ref={ref} width="100%" height={autoHeight ? 'auto' : '100%'} />
}

export const LogViewer = React.memo(LogTerminal)
