import React from 'react'
import { MenuItem } from '@blueprintjs/core'
import { Layout, Text } from '@harnessio/uicore'
import css from './IDEDropdownSection.module.scss'

interface DropdownSectionProps {
  heading: string
  options: any[]
  value: string | undefined
  onChange: (field: string, value: any) => void
}

export const CustomIDESection = ({ heading, options, value, onChange }: DropdownSectionProps) => {
  return (
    <>
      <Text className={css.menuHeading}>{heading}</Text>
      {options.map((item: any) => {
        return (
          <MenuItem
            key={item.value}
            active={item.value === value}
            text={
              <Layout.Horizontal width="90%" spacing="medium" flex={{ alignItems: 'center', justifyContent: 'start' }}>
                <img height={16} width={16} src={item.icon} />
                <Text className={css.menuLabel}>{item.label}</Text>
              </Layout.Horizontal>
            }
            onClick={() => onChange('ide', item.value)}
          />
        )
      })}
    </>
  )
}
