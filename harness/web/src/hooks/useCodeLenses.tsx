import { useEffect } from 'react'
import * as monaco from 'monaco-editor'
import { Range } from 'monaco-editor'
import type { editor, languages, IRange } from 'monaco-editor'
import { noop, pick } from 'lodash-es'
import { ILanguageFeaturesService } from 'monaco-editor/esm/vs/editor/common/services/languageFeatures.js'
import { OutlineModel } from 'monaco-editor/esm/vs/editor/contrib/documentSymbols/browser/outlineModel.js'
import { StandaloneServices } from 'monaco-editor/esm/vs/editor/standalone/browser/standaloneServices.js'

export interface CodeLensCommand {
  title: string
  onClick: (arg: { path: string[]; range: monaco.IRange }, ...args: unknown[]) => void
  args?: unknown[]
}

interface CommandArg extends Pick<CodeLensCommand, 'onClick' | 'args'> {
  range: monaco.IRange
  symbols: monaco.languages.DocumentSymbol[]
}

export interface CodeLensConfig
  extends Partial<Pick<monaco.languages.DocumentSymbol, 'name' | 'containerName' | 'kind'>> {
  commands: CodeLensCommand[]
}

export const getDocumentSymbols = async (model: editor.ITextModel): Promise<languages.DocumentSymbol[]> => {
  const { documentSymbolProvider } = StandaloneServices.get(ILanguageFeaturesService)
  const outline = await OutlineModel.create(documentSymbolProvider, model)
  return outline.asListOfDocumentSymbols()
}

export const getPathFromRange = (range: IRange, symbols: languages.DocumentSymbol[]): string[] => {
  const path: string[] = []
  for (const symbol of symbols) {
    if (!Range.containsRange(symbol.range, range)) continue
    path.push(symbol.name)
    if (Range.equalsRange(symbol.range, range)) break
    if (!symbol.children) continue
    path.push(...getPathFromRange(range, symbol.children))
  }
  return path
}

export type UseCodeLenses = (arg: {
  editor?: editor.IStandaloneCodeEditor | null
  codeLensConfigs?: CodeLensConfig[]
}) => void

export const useCodeLenses: UseCodeLenses = ({ editor, codeLensConfigs }): void => {
  useEffect(() => {
    if (!codeLensConfigs) return

    const commandId = editor?.addCommand(0, (_, { range, symbols, onClick, args }: CommandArg) => {
      const path = getPathFromRange(range, symbols)
      onClick({ path, range }, ...(args ? args : []))
    })

    if (!commandId) return

    const disposable = monaco.languages.registerCodeLensProvider('yaml', {
      provideCodeLenses: async model => {
        const symbols = await getDocumentSymbols(model)
        const lenses = symbols.reduce<monaco.languages.CodeLens[]>((acc, symbol) => {
          const codeLensConfig = codeLensConfigs.find(config => {
            const configSymbolProps = pick(config, ['name', 'containerName', 'kind'])

            return (Object.keys(configSymbolProps) as ('name' | 'containerName' | 'kind')[]).every(
              key => symbol[key] === config[key]
            )
          })

          if (!codeLensConfig) return acc

          const { range } = symbol

          acc.push(
            ...codeLensConfig.commands.map(({ title, onClick, args }) => {
              const commandArg: CommandArg = {
                range,
                symbols,
                onClick,
                args
              }

              return {
                range,
                command: {
                  id: commandId,
                  title,
                  arguments: [commandArg]
                }
              }
            })
          )

          return acc
        }, [])

        return {
          lenses,
          dispose: noop
        }
      }
    })

    return disposable.dispose
  }, [codeLensConfigs, editor])
}
