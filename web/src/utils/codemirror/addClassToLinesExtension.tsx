import { ViewPlugin, ViewUpdate, type EditorView, Decoration } from '@codemirror/view'
import { RangeSetBuilder, Compartment, Extension } from '@codemirror/state'

export const addClassToLinesExtension: AddClassToLinesExtension = (lines = [], className) => {
  const highlightLineDecoration = Decoration.line({
    attributes: { class: className }
  })

  function updateDecorations(view: EditorView) {
    const builder = new RangeSetBuilder<Decoration>()

    for (const { from, to } of view.visibleRanges) {
      for (let pos = from; pos <= to; ) {
        const line = view.state.doc.lineAt(pos)

        if (lines.includes(line.number)) {
          builder.add(line.from, line.from, highlightLineDecoration)
        }
        pos = line.to + 1
      }
    }

    return builder.finish()
  }

  class Plugin {
    decorations = Decoration.none

    constructor(view: EditorView) {
      this.decorations = updateDecorations(view)
    }

    update(update: ViewUpdate) {
      if (update.docChanged || update.viewportChanged) {
        this.decorations = updateDecorations(update.view)
      }
    }
  }

  const config = new Compartment()

  const update: AddClassToLinesReturnType[1] = (_lines, view) => {
    lines = _lines

    view?.dispatch({
      effects: config.reconfigure(
        ViewPlugin.fromClass(Plugin, {
          decorations: v => v.decorations
        })
      )
    })
  }

  return [
    config.of(
      ViewPlugin.fromClass(Plugin, {
        decorations: v => v.decorations
      })
    ),
    update
  ]
}

type AddClassToLinesReturnType = [Extension, (lines: number[], view?: EditorView) => void]

type AddClassToLinesExtension = (lines: number[], className: string) => AddClassToLinesReturnType
