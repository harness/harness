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

import React, { FC, PropsWithChildren, useState } from 'react'
import classNames from 'classnames'
import { defaultTo } from 'lodash-es'
import copy from 'clipboard-copy'
import { Link } from 'react-router-dom'
import type { TableExpandedToggleProps } from 'react-table'
import { Button, ButtonProps, ButtonVariation, Layout, Text } from '@harnessio/uicore'
import type { IconName, IconProps } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'

import { killEvent } from '@ar/common/utils'
import { useStrings } from '@ar/frameworks/strings/String'
import type { RepositoryConfigType } from '@ar/common/types'
import { TimeAgoPopover } from '@ar/components/TimeAgoPopover/TimeAgoPopover'
import RepositoryLocationBadge from '@ar/components/Badge/RepositoryLocationBadge'

import { DefaultIconProps } from './constants'
import { handleToggleExpandableRow } from './utils'
import CommandBlock from '../CommandBlock/CommandBlock'
import { NonProdTag, ProdTag } from '../Tag/Tags'

import css from './TableCells.module.scss'

interface CommonCellProps {
  value: number | string
}

interface CountCellProps extends CommonCellProps {
  icon?: IconName
  iconProps?: Omit<IconProps, 'name'>
}

const LastModifiedCell = ({ value }: CommonCellProps): JSX.Element => {
  return <TimeAgoPopover time={Number(value)} color={Color.GREY_900} />
}

export const UrlCell = ({ value }: CommonCellProps): JSX.Element => {
  const { getString } = useStrings()
  return <Text color={Color.GREY_900}>{defaultTo(value, getString('na'))}</Text>
}

interface CopyUrlCellProps {
  value: string
}

export const CopyUrlCell: FC<PropsWithChildren<CopyUrlCellProps>> = ({ value, children }): JSX.Element => {
  const { getString } = useStrings()
  const [openTooltip, setOpenTooltip] = useState(false)
  const showCopySuccess = () => {
    setOpenTooltip(true)
    setTimeout(() => {
      setOpenTooltip(false)
    }, 1000)
  }
  return (
    <Button
      className={classNames(css.copyButton, css.copyUrlBtn)}
      intent="primary"
      minimal
      icon="link"
      variation={ButtonVariation.LINK}
      iconProps={{ size: 12, className: css.copyUrlIcon }}
      onClick={evt => {
        killEvent(evt)
        copy(value)
        showCopySuccess()
      }}
      tooltip={getString('copied')}
      tooltipProps={{ isOpen: openTooltip, isDark: true }}>
      {children}
    </Button>
  )
}

interface CopyTextCellProps {
  value: string
  icon?: ButtonProps['rightIcon']
  iconProps?: ButtonProps['iconProps']
}

export const CopyTextCell: FC<PropsWithChildren<CopyTextCellProps>> = ({
  value,
  icon,
  iconProps,
  children
}): JSX.Element => {
  const { getString } = useStrings()
  const [openTooltip, setOpenTooltip] = useState(false)
  const showCopySuccess = () => {
    setOpenTooltip(true)
    setTimeout(() => {
      setOpenTooltip(false)
    }, 1000)
  }
  return (
    <Button
      className={css.copyButton}
      intent="primary"
      minimal
      variation={ButtonVariation.LINK}
      rightIcon={defaultTo(icon, 'code-copy')}
      iconProps={iconProps}
      onClick={evt => {
        killEvent(evt)
        copy(value)
        showCopySuccess()
      }}
      tooltip={getString('copied')}
      tooltipProps={{ isOpen: openTooltip, isDark: true }}>
      {children}
    </Button>
  )
}

interface RepositoryLocationBadgeProps {
  value: RepositoryConfigType
}

export const RepositoryLocationBadgeCell = ({ value }: RepositoryLocationBadgeProps): JSX.Element => {
  return <RepositoryLocationBadge type={value} />
}

export const SizeCell = ({ value }: CommonCellProps): JSX.Element => {
  const { getString } = useStrings()
  return <Text color={Color.GREY_900}>{defaultTo(value, getString('na'))}</Text>
}

export const CountCell = ({ value, icon, iconProps }: CountCellProps): JSX.Element => {
  const _iconProps = defaultTo(iconProps, DefaultIconProps)
  return (
    <Text color={Color.GREY_900} rightIcon={icon} rightIconProps={_iconProps}>
      {defaultTo(value, 0)}
    </Text>
  )
}

export const TextCell = ({ value }: CommonCellProps): JSX.Element => {
  const { getString } = useStrings()
  return (
    <Text color={Color.GREY_900} lineClamp={1}>
      {defaultTo(value, getString('na'))}
    </Text>
  )
}

export interface ToggleAccordionCellProps {
  expandedRows: Set<string>
  setExpandedRows: React.Dispatch<React.SetStateAction<Set<string>>>
  value: string
  initialIsExpanded: boolean
  getToggleRowExpandedProps: (props?: Partial<TableExpandedToggleProps>) => TableExpandedToggleProps
  onToggleRowExpanded: (val: boolean) => void
}

const ToggleAccordionCell = (props: ToggleAccordionCellProps): JSX.Element => {
  const { expandedRows, setExpandedRows, value } = props
  const [isExpanded, setIsExpanded] = React.useState<boolean>(props.initialIsExpanded)

  React.useEffect(() => {
    if (value) {
      const isRowExpanded = expandedRows.has(value)
      setIsExpanded(isRowExpanded)
      props.onToggleRowExpanded(isRowExpanded)
    }
  }, [value, expandedRows, props.onToggleRowExpanded])

  const toggleRow = (evt: React.MouseEvent<Element, MouseEvent>): void => {
    killEvent(evt)
    setExpandedRows(handleToggleExpandableRow(value))
  }

  return (
    <Layout.Horizontal>
      <Button
        {...props.getToggleRowExpandedProps()}
        onClick={toggleRow}
        color={Color.GREY_600}
        icon={isExpanded ? 'chevron-up' : 'chevron-down'}
        variation={ButtonVariation.ICON}
        iconProps={{ size: 19 }}
        className={css.toggleAccordion}
      />
    </Layout.Horizontal>
  )
}

interface LinkCellProps {
  label: string
  subLabel?: string
  prefix?: React.ReactElement
  postfix?: React.ReactElement
  linkTo: string
}

const LinkCell = (props: LinkCellProps): JSX.Element => {
  const { prefix, postfix, label, linkTo, subLabel } = props
  return (
    <Layout.Horizontal
      className={css.nameCellContainer}
      flex={{ justifyContent: 'flex-start', alignItems: 'flex-start' }}>
      {prefix}
      <Layout.Vertical>
        <Link to={linkTo}>
          <Text color={Color.PRIMARY_7} lineClamp={1}>
            {label}
          </Text>
        </Link>
        {subLabel && <Text lineClamp={1}>{subLabel}</Text>}
      </Layout.Vertical>
      {postfix}
    </Layout.Horizontal>
  )
}

interface DeploymentsCellProps {
  prodCount?: number
  nonProdCount?: number
}

export const DeploymentsCell = ({ prodCount, nonProdCount }: DeploymentsCellProps) => {
  return (
    <Layout.Horizontal spacing="xsmall">
      <Layout.Horizontal spacing="small">
        <Text font={{ variation: FontVariation.BODY }}>{defaultTo(prodCount, 0)}</Text>
        <ProdTag />
      </Layout.Horizontal>
      <Layout.Horizontal spacing="small">
        <Text font={{ variation: FontVariation.BODY }}>{defaultTo(nonProdCount, 0)}</Text>
        <NonProdTag />
      </Layout.Horizontal>
    </Layout.Horizontal>
  )
}

export const PullCommandCell = ({ value }: CommonCellProps) => {
  const { getString } = useStrings()
  if (!value) return <>{getString('na')}</>
  return <CommandBlock noWrap commandSnippet={value as string} allowCopy onCopy={killEvent} />
}

export default {
  UrlCell,
  SizeCell,
  CountCell,
  LinkCell,
  TextCell,
  CopyUrlCell,
  CopyTextCell,
  DeploymentsCell,
  PullCommandCell,
  LastModifiedCell,
  ToggleAccordionCell,
  RepositoryLocationBadgeCell
}
