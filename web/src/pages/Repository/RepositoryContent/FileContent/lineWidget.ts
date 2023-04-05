import { StateField, Range, Text } from '@codemirror/state'
import { EditorView, Decoration, WidgetType } from '@codemirror/view'

export enum LineWidgetPosition {
  TOP = 'top',
  BOTTOM = 'bottom'
}

export interface LineWidgetSpec {
  lineNumber: number
  position: LineWidgetPosition
}

export type LineWidgetGeneration<T extends LineWidgetSpec> = (args: T) => WidgetType

function buildLineDecorations<T extends LineWidgetSpec = LineWidgetSpec>(
  doc: Text,
  widgetFor: LineWidgetGeneration<T>,
  spec: T[]
) {
  const decorations: Range<Decoration>[] = []

  spec.forEach(_spec => {
    const { lineNumber, position } = _spec
    const lines = doc.lines
    const lineInfo = doc.line(lineNumber)

    if (lineNumber <= lines) {
      const decoration = Decoration.widget({
        widget: widgetFor(_spec),
        block: true,
        side: position === LineWidgetPosition.TOP ? -1 : 1
      })

      decorations.push(decoration.range(position === LineWidgetPosition.TOP ? lineInfo.from : lineInfo.to))
    }
  })

  return Decoration.set(decorations)
}

interface LineWidgetParams<T extends LineWidgetSpec = LineWidgetSpec> {
  spec: T[]
  widgetFor: LineWidgetGeneration<T>
}

export function lineWidget<T extends LineWidgetSpec = LineWidgetSpec>({ spec, widgetFor }: LineWidgetParams<T>) {
  return StateField.define({
    create: state => {
      return buildLineDecorations(state.doc, widgetFor, spec)
    },

    update(decorations, transation) {
      return transation.docChanged
        ? buildLineDecorations(transation.newDoc, widgetFor, spec)
        : decorations.map(transation.changes)
    },

    provide: f => EditorView.decorations.from(f)
  })
}
