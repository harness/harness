import React from 'react'
import { render } from '@testing-library/react'
import { renderHook } from '@testing-library/react-hooks'

import { String, useStrings } from '../String'
import { StringsContext } from '../StringsContext'

const value = {
  data: {
    a: { b: 'Test Value 1' },
    harness: 'Harness',
    test: '{{ $.a.b }} in template',
    test2: '{{ $.test }} again'
  }
}
describe('String tests', () => {
  test('renders strings with simple id', () => {
    const { container } = render(
      <StringsContext.Provider value={value as any}>
        <String stringID={'harness' as any} />
      </StringsContext.Provider>
    )

    expect(container).toMatchInlineSnapshot(`
      <div>
        <span>
          Harness
        </span>
      </div>
    `)
  })

  test('renders error when key not found', () => {
    const { container } = render(
      <StringsContext.Provider value={value as any}>
        <String stringID={'harnes' as any} />
      </StringsContext.Provider>
    )

    expect(container).toMatchInlineSnapshot(`
      <div>
        <span>
          No valid template with id "harnes" found in any namespace
        </span>
      </div>
    `)
  })

  test('renders strings with nested value', () => {
    const { container } = render(
      <StringsContext.Provider value={value as any}>
        <String stringID={'a.b' as any} />
      </StringsContext.Provider>
    )

    expect(container).toMatchInlineSnapshot(`
      <div>
        <span>
          Test Value 1
        </span>
      </div>
    `)
  })

  test('renders strings with self reference values', () => {
    const { container } = render(
      <StringsContext.Provider value={value as any}>
        <String stringID={'test' as any} />
      </StringsContext.Provider>
    )

    expect(container).toMatchInlineSnapshot(`
      <div>
        <span>
          Test Value 1 in template
        </span>
      </div>
    `)
  })

  test('self reference only works for one level', () => {
    const { container } = render(
      <StringsContext.Provider value={value as any}>
        <String stringID={'test2' as any} />
      </StringsContext.Provider>
    )

    expect(container).toMatchInlineSnapshot(`
      <div>
        <span>
          {{ $.a.b }} in template again
        </span>
      </div>
    `)
  })
})

describe('useString tests', () => {
  describe('getString', () => {
    test('works with simple id', () => {
      const wrapper = ({ children }: React.PropsWithChildren<unknown>): React.ReactElement => (
        <StringsContext.Provider value={value as any}>{children}</StringsContext.Provider>
      )
      const { result } = renderHook(() => useStrings(), { wrapper })

      expect(result.current.getString('harness' as any)).toMatchInlineSnapshot(`"Harness"`)
    })

    test('works with nested values', () => {
      const wrapper = ({ children }: React.PropsWithChildren<unknown>): React.ReactElement => (
        <StringsContext.Provider value={value as any}>{children}</StringsContext.Provider>
      )
      const { result } = renderHook(() => useStrings(), { wrapper })

      expect(result.current.getString('a.b' as any)).toMatchInlineSnapshot(`"Test Value 1"`)
    })

    test('works with self reference values', () => {
      const wrapper = ({ children }: React.PropsWithChildren<unknown>): React.ReactElement => (
        <StringsContext.Provider value={value as any}>{children}</StringsContext.Provider>
      )
      const { result } = renderHook(() => useStrings(), { wrapper })

      expect(result.current.getString('test' as any)).toMatchInlineSnapshot(`"Test Value 1 in template"`)
    })

    test('self reference works foor only one level', () => {
      const wrapper = ({ children }: React.PropsWithChildren<unknown>): React.ReactElement => (
        <StringsContext.Provider value={value as any}>{children}</StringsContext.Provider>
      )
      const { result } = renderHook(() => useStrings(), { wrapper })

      expect(result.current.getString('test2' as any)).toMatchInlineSnapshot(`"{{ $.a.b }} in template again"`)
    })

    test('throws when key not found', () => {
      const wrapper = ({ children }: React.PropsWithChildren<unknown>): React.ReactElement => (
        <StringsContext.Provider value={value as any}>{children}</StringsContext.Provider>
      )
      const { result } = renderHook(() => useStrings(), { wrapper })

      expect(() => result.current.getString('harnes' as any)).toThrowError(
        'No valid template with id "harnes" found in any namespace'
      )
    })
  })

  test('Works with custom getString', () => {
    const { container } = render(
      <StringsContext.Provider value={{ ...value, getString: (key: string) => key } as any}>
        <String stringID={'harness.foo.bar.baz' as any} />
      </StringsContext.Provider>
    )

    expect(container).toMatchInlineSnapshot(`
      <div>
        <span>
          harness.foo.bar.baz
        </span>
      </div>
    `)
  })
})
