/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
