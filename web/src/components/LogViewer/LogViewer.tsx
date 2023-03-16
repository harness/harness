import React, { useEffect, useRef } from 'react'
import { Container } from '@harness/uicore'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { CanvasAddon } from 'xterm-addon-canvas'
import { SearchAddon } from 'xterm-addon-search'
import { WebLinksAddon } from 'xterm-addon-web-links'
import 'xterm/css/xterm.css'

const DEFAULT_SCROLLBACK_LINES = 1000

export type TermRefs = { term: Terminal; fitAddon: FitAddon } | undefined

export interface LogViewerProps {
  /** Search text */
  searchText?: string

  /** Number of scrollback lines */
  scrollbackLines?: number

  /** Log content as string. Note that we can support streaming easily if backend has it */
  content: string

  termRefs?: React.MutableRefObject<TermRefs>
}

export const LogViewer: React.FC<LogViewerProps> = ({ scrollbackLines, content, termRefs }) => {
  const ref = useRef<HTMLDivElement | null>(null)
  const lines = content.split(/\r?\n/)
  const term = useRef<Terminal>()

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

      _term.clear()
      _term.open(ref?.current as HTMLDivElement)

      fitAddon.fit()
      searchAddon.activate(_term)

      _term.write('\x1b[?25l') // disable cursor

      lines.forEach((line, _index) => _term.writeln(line))

      term.current = _term
      if (termRefs) {
        termRefs.current = { term: _term, fitAddon }
      }
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  return <Container ref={ref} width="100%" height="100%" />
}
