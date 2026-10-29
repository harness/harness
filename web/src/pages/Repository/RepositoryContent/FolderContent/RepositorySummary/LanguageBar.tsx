/*
 * Copyright 2024 Harness, Inc.
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

import React, { useMemo } from 'react'
import { Layout, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import { DEFAULT_LANGUAGE_COLOR, getLanguageColor } from './languageColors'
import css from './LanguageBar.module.scss'

export interface LanguageStat {
  name: string
  percentage: number
}

interface LanguageBarProps {
  languages: LanguageStat[]
}

const MAX_LEGEND_ITEMS = 6

const formatPercentage = (pct: number): string => {
  if (pct > 0 && pct < 0.1) return '< 0.1'
  return pct.toFixed(1)
}

const LanguageBar: React.FC<LanguageBarProps> = ({ languages }) => {
  const { getString } = useStrings()

  const sorted = useMemo(() => [...languages].sort((a, b) => b.percentage - a.percentage), [languages])

  if (!sorted.length) {
    return null
  }

  const legendItems = sorted.slice(0, MAX_LEGEND_ITEMS)
  const otherPercent = sorted.slice(MAX_LEGEND_ITEMS).reduce((sum, lang) => sum + lang.percentage, 0)

  return (
    <Layout.Vertical spacing="small">
      <Text font={{ variation: FontVariation.BODY2_SEMI }} color={Color.BLACK}>
        {getString('languages')}
      </Text>

      <div className={css.bar}>
        {sorted.map(lang => (
          <div
            key={lang.name}
            className={css.segment}
            title={`${lang.name} ${formatPercentage(lang.percentage)}%`}
            style={{
              backgroundColor: getLanguageColor(lang.name),
              width: `clamp(3px, ${lang.percentage}%, 100%)`
            }}
          />
        ))}
      </div>

      <Layout.Horizontal className={css.legend}>
        {legendItems.map(lang => (
          <Layout.Horizontal key={lang.name} className={css.legendItem} spacing="xsmall">
            <div className={css.dot} style={{ backgroundColor: getLanguageColor(lang.name) }} />
            <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_700}>
              {lang.name}
            </Text>
            <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_450}>
              {formatPercentage(lang.percentage)}%
            </Text>
          </Layout.Horizontal>
        ))}
        {otherPercent > 0 && (
          <Layout.Horizontal className={css.legendItem} spacing="xsmall">
            <div className={css.dot} style={{ backgroundColor: DEFAULT_LANGUAGE_COLOR }} />
            <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_700}>
              {getString('other')}
            </Text>
            <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_450}>
              {formatPercentage(otherPercent)}%
            </Text>
          </Layout.Horizontal>
        )}
      </Layout.Horizontal>
    </Layout.Vertical>
  )
}

export default LanguageBar
