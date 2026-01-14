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
export const getTruncatedValue = (fullPath: string): string => {
  if (!fullPath) return ''
  const parts = fullPath.split('/')
  return parts.length > 0 ? parts[parts.length - 1] : fullPath
}

export const downloadYaml = (yamlContent: string | undefined, fileName: string, onError?: () => void): void => {
  if (!yamlContent) {
    if (onError) {
      onError()
    }
    return
  }

  const blob = new Blob([yamlContent], { type: 'text/yaml' })
  const a = document.createElement('a')
  const url = URL.createObjectURL(blob)
  a.href = url
  a.download = fileName
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
