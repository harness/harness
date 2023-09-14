import React, { useEffect, useMemo, useRef } from 'react'
import { Container } from '@harnessio/uicore'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { CanvasAddon } from 'xterm-addon-canvas'
import { SearchAddon } from 'xterm-addon-search'
import { WebLinksAddon } from 'xterm-addon-web-links'
import 'xterm/css/xterm.css'
import { useEventListener } from 'hooks/useEventListener'

const DEFAULT_SCROLLBACK_LINES = 100000

export type TermRefs = { term: Terminal; fitAddon: FitAddon } | undefined

export interface LogViewerProps {
  /** Search text */
  searchText?: string

  /** Number of scrollback lines */
  scrollbackLines?: number

  /** Log content as string */
  content: string

  termRefs?: React.MutableRefObject<TermRefs>

  autoHeight?: boolean
}

export const LogViewer: React.FC<LogViewerProps> = ({ scrollbackLines, content, termRefs, autoHeight }) => {
  const ref = useRef<HTMLDivElement | null>(null)
  const lines = useMemo(() => content.split(/\r?\n/), [content])
  const term = useRef<{ term: Terminal; fitAddon: FitAddon }>()

  useEffect(() => {
    if (!term.current) {
      const _term = new Terminal({
        cursorBlink: true,
        cursorStyle: 'block',
        allowTransparency: true,
        disableStdin: true,
        scrollback: scrollbackLines || DEFAULT_SCROLLBACK_LINES,
        theme: {
          background: 'transparent'
        }
      })

      const searchAddon = new SearchAddon()
      const fitAddon = new FitAddon()
      const webLinksAddon = new WebLinksAddon()

      _term.loadAddon(searchAddon)
      _term.loadAddon(fitAddon)
      _term.loadAddon(webLinksAddon)
      _term.loadAddon(new CanvasAddon())

      _term.open(ref?.current as HTMLDivElement)

      fitAddon.fit()
      searchAddon.activate(_term)

      _term.write('\x1b[?25l') // disable cursor

      term.current = { term: _term, fitAddon }

      if (termRefs) {
        termRefs.current = term.current
      }
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    term.current?.term?.clear()
    lines.forEach(line => term.current?.term?.writeln(line))

    if (autoHeight) {
      term.current?.term?.resize(term.current?.term?.cols, lines.length + 1)
    }

    return () => {
      term.current?.term?.clear()
    }
  }, [lines, autoHeight])

  useEventListener('resize', () => {
    term.current?.fitAddon?.fit()
  })

  return <Container ref={ref} width="100%" height={autoHeight ? 'auto' : '100%'} />
}
