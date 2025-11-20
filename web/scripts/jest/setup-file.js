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

// https://stackoverflow.com/questions/71704077/errors-when-updating-testing-library-user-event-to-v-14
// document.createRange = () => ({
//   setStart: () => {},
//   setEnd: () => {},
//   commonAncestorContainer: {
//     nodeName: 'BODY',
//     ownerDocument: document,
//   },
// })

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
  value: jest.fn().mockImplementation(query => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: jest.fn(), // Deprecated
    removeListener: jest.fn(), // Deprecated
    addEventListener: jest.fn(),
    removeEventListener: jest.fn(),
    dispatchEvent: jest.fn()
  }))
})

jest.mock('react-timeago', () => () => 'dummy date')

class MockIntersectionObserver {
  constructor() {
    this.root = null
    this.rootMargin = ''
    this.thresholds = []
    this.disconnect = () => null
    this.observe = () => null
    this.takeRecords = () => []
    this.unobserve = () => null
  }
}

Object.defineProperty(window, 'IntersectionObserver', {
  writable: true,
  configurable: true,
  value: MockIntersectionObserver
})

Object.defineProperty(global, 'IntersectionObserver', {
  writable: true,
  configurable: true,
  value: MockIntersectionObserver
})

Object.defineProperty(window, 'getApiBaseUrl', {
  writable: true,
  configurable: true,
  value: jest.fn().mockImplementation(query => {
    return '/'
  })
})

// TODO: add tests for v2
jest.mock('@harnessio/react-har-service-v2-client', () => ({
  useListRegistriesQuery: jest.fn(),
  useListPackagesQuery: jest.fn(),
  useListArtifactsQuery: jest.fn(),
  useGetArtifactMetadataQuery: jest.fn(),
  useUpdateMetadataMutation: jest.fn(),
  useGetMetadataKeysQuery: jest.fn(),
  useGetMetadataValuesQuery: jest.fn()
}))
