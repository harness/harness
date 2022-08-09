import '@testing-library/jest-dom'
import { setAutoFreeze, enableMapSet } from 'immer'
import { noop } from 'lodash-es'

// set up Immer
setAutoFreeze(false)
enableMapSet()

process.env.TZ = 'UTC'

document.createRange = () => ({
  setStart: () => {},
  setEnd: () => {},
  commonAncestorContainer: {
    nodeName: 'BODY',
    ownerDocument: document,
  },
})
window.HTMLElement.prototype.scrollIntoView = jest.fn()
window.scrollTo = jest.fn()

window.fetch = jest.fn((url, options) => {
  fail(`A fetch is being made to url '${url}' with options:
${JSON.stringify(options, null, 2)}
Please mock this call.`)
  throw new Error()
})

Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: jest.fn().mockImplementation((query) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: jest.fn(), // Deprecated
    removeListener: jest.fn(), // Deprecated
    addEventListener: jest.fn(),
    removeEventListener: jest.fn(),
    dispatchEvent: jest.fn(),
  })),
})

jest.mock('react-timeago', () => () => 'dummy date')
