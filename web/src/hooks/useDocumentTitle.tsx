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

import { useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import type { CODEProps } from 'RouteDefinitions'

type Title = string | string[]

export function useDocumentTitle(title: Title) {
  const { getString } = useStrings()
  const { standalone } = useAppContext()
  const { repoName } = useParams<CODEProps>()

  useEffect(() => {
    const _title = document.title
    const titleArray = Array.isArray(title) ? [...title] : [title]
    if (repoName) {
      titleArray.push(repoName)
    }

    document.title = [...titleArray, standalone ? getString('gitness') : getString('exportSpace.harness')].join(' | ')

    return () => {
      document.title = _title
    }
  }, [title, getString, repoName, standalone])
}
