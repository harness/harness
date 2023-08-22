import React from 'react'
import { Text } from '@harnessio/uicore'
import { IconName, HarnessIcons } from '@harnessio/icons'
import type { Spacing, PaddingProps } from '@harnessio/design-system'
import { Falsy, Match, Render, Truthy } from 'react-jsx-match'
import css from './TabTitleWithCount.module.scss'

export const TabTitleWithCount: React.FC<{
  icon: IconName
  title: string
  count?: number
  countElement?: React.ReactNode
  padding?: Spacing | PaddingProps
  iconSize?: number
}> = ({ icon, title, count, padding, countElement, iconSize = 16 }) => {
  // Icon inside a tab got overriden-and-looked-bad styles from UICore
  // on hover. Use icon directly instead
  const TabIcon: React.ElementType = HarnessIcons[icon]

  return (
    <Text className={css.tabTitle} padding={padding} tag="div">
      <TabIcon width={iconSize} height={iconSize} />
      {title}
      <Match expr={countElement}>
        <Truthy>{countElement}</Truthy>
        <Falsy>
          <Render when={count}>
            <Text inline className={css.count}>
              {count}
            </Text>
          </Render>
        </Falsy>
      </Match>
    </Text>
  )
}

export const tabContainerCSS = css
