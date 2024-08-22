/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

export function encode(arg: unknown): string | undefined {
  if (typeof arg !== 'undefined') return btoa(encodeURIComponent(JSON.stringify(arg)))
}

export function decode<T = unknown>(arg: string): T {
  try {
    return JSON.parse(decodeURIComponent(atob(arg)))
  } catch (e) {
    return arg as T
  }
}

export default class SecureStorage {
  public static exceptions: string[] = []
  public static sessionExceptions: string[] = []

  public static set(key: string, value: unknown): void {
    const str = encode(value)
    if (str) localStorage.setItem(key, str)
  }

  public static get<T = unknown>(key: string): T | undefined {
    const str = localStorage.getItem(key)
    if (str) return decode<T>(str)
  }

  public static registerCleanupException(key: string): void {
    SecureStorage.exceptions.push(key)
  }

  public static registerCleanupSessionException(key: string): void {
    SecureStorage.sessionExceptions.push(key)
  }

  public static clear(): void {
    // clear localStorage, except fields to persist across user-sessions:
    const storage: [string, string][] = SecureStorage.exceptions.map(key => [key, localStorage.getItem(key) || ''])
    const sessionKeys: [string, string][] = SecureStorage.sessionExceptions.map(key => [
      key,
      sessionStorage.getItem(key) || ''
    ])

    localStorage.clear()

    /* adding this to clear sessionStorage on logout - because at harness we want user session to end once the user is logged out,
       so we are clearing this from our end - because by default sessionStorage behavior doesn't care for login/logout events */
    sessionStorage.clear()

    storage.forEach(([key, val]) => {
      localStorage.setItem(key, val)
    })

    sessionKeys.forEach(([key, val]) => {
      sessionStorage.setItem(key, val)
    })
  }
}
