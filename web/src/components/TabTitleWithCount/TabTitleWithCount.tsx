import React from 'react'
import { Text, IconName, HarnessIcons, Spacing, PaddingProps } from '@harness/uicore'
import { Render } from 'react-jsx-match'
import css from './TabTitleWithCount.module.scss'

export const TabTitleWithCount: React.FC<{
  icon: IconName
  title: string
  count?: number
  padding?: Spacing | PaddingProps
}> = ({ icon, title, count, padding }) => {
  // Icon inside a tab got overriden-and-looked-bad styles from UICore
  // on hover. Use icon directly instead
  const TabIcon: React.ElementType = HarnessIcons[icon]

  return (
    <Text className={css.tabTitle} padding={padding}>
      <TabIcon width={16} height={16} />
      {title}
      <Render when={count}>
        <Text inline className={css.count}>
          {count}
        </Text>
      </Render>
    </Text>
  )
}

export const tabContainerCSS = css
