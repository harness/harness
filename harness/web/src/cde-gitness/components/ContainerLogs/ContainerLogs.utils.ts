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

export function parseLogString(logString: string) {
  if (!logString) {
    return ''
  }
  const logEntries = logString.trim().split('\n')
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const parsedLogs: any = []
  logEntries.forEach((entry, lineIndex) => {
    // Parse the entry as JSON
    const jsonEntry = JSON.parse(entry)
    // Apply the regex to the 'out' field
    const parts = (jsonEntry?.message).match(/time="([^"]+)" level=([^ ]+) msg="([^"]+)"(.*)/)
    if (parts) {
      const [, time, level, message, details, out] = parts
      const detailParts = details.trim().split(' ')
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const detailDict: any = {}
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      detailParts.forEach((part: any) => {
        if (part.includes('=')) {
          const [key, value] = part.split('=')
          detailDict[key.trim()] = value.trim()
        }
      })
      parsedLogs.push({ time, level, message, out, details: detailDict, pos: jsonEntry.pos, logLevel: jsonEntry.level })
    } else {
      parsedLogs.push({
        time: jsonEntry.time,
        level: jsonEntry.level,
        message: jsonEntry?.message,
        pos: lineIndex,
        logLevel: jsonEntry.level
      })
    }
  })

  return parsedLogs
}
