import React, { useEffect, useContext, createContext, memo } from 'react'
import type { FC, PropsWithChildren, ElementType } from 'react'
import type { TagProps } from '../svg-icon'
import { TAG } from '../svg-icon'

export interface IconProps {
  size?: string
  color?: string
  strokeWidth?: string
  title?: boolean | string
  className?: string
}

export interface NamedIconProps extends IconProps {
  name: string
}

export const Icon = memo(function NamedIcon(props: NamedIconProps) {
  const ctx = useContext(IconContext)
  const { name, size, color, strokeWidth, title, className } = props
  const Tag = TAG as ElementType<TagProps>
  const label = name.split('/')[0]

  useEffect(() => {
    ctx.renderHook?.(props)
  }, [ctx, props])

  return (
    <Tag
      class={className}
      name={name}
      size={size || ctx.size}
      color={color || ctx.color}
      stroke-width={strokeWidth || ctx.strokeWidth}
      {...(title || ctx.title ? { title: typeof title === 'string' ? title || label : label } : {})}
    />
  )
})

export type IconType = (props: IconProps) => JSX.Element

export interface IconContextProps extends IconProps {
  renderHook?: (props: NamedIconProps) => void
}

const IconContext = createContext<IconContextProps>({})

export const IconContextProvider: FC<PropsWithChildren<IconContextProps>> = ({ children, ...props }) => (
  <IconContext.Provider value={props || {}}>{children}</IconContext.Provider>
)
