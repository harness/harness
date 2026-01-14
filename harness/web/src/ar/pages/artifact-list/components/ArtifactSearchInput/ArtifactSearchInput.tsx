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

import React from 'react'
import AISearchInput from '@ar/components/AISearchInput/AISearchInput'
import { ArtifactListAIOptions } from './constants'

interface ArtifactSearchInputProps {
  searchTerm: string
  onChange: (val: string) => void
  placeholder?: string
}

export default function ArtifactSearchInput(props: ArtifactSearchInputProps) {
  return <AISearchInput {...props} options={ArtifactListAIOptions} />
}
