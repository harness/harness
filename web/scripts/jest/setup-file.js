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
