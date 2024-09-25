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

import React, { useEffect } from 'react'
import cx from 'classnames'
import { Button, ButtonSize, ButtonVariation, Container, Layout, Tag, Text } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { useGet } from 'restful-react'
import { Menu, Spinner } from '@blueprintjs/core'
import { Icon } from '@harnessio/icons'
import { isEmpty } from 'lodash-es'
import { ColorName, LabelType, getColorsObj, getScopeData, getScopeIcon } from 'utils/Utils'
import type { RepoRepositoryOutput, TypesLabelValue } from 'services/code'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { getConfig } from 'services/config'
import css from './Label.module.scss'

interface Label {
  name: string
  scope?: number
  label_color?: ColorName
}

interface LabelTitleProps extends Label {
  labelType?: LabelType
  value_count?: number
}

interface LabelProps extends Label {
  label_value?: {
    name?: string
    color?: ColorName
  }
  className?: string
  removeLabelBtn?: boolean
  handleRemoveClick?: () => void
  onClick?: () => void
  disableRemoveBtnTooltip?: boolean
}

export const Label: React.FC<LabelProps> = props => {
  const {
    name,
    scope,
    label_value: { name: valueName, color: valueColor } = {},
    label_color,
    className,
    removeLabelBtn,
    handleRemoveClick,
    onClick,
    disableRemoveBtnTooltip = false
  } = props
  const { getString } = useStrings()
  const { standalone } = useAppContext()
  const scopeIcon = getScopeIcon(scope, standalone)
  if (valueName) {
    const colorObj = getColorsObj(valueColor ?? label_color ?? ColorName.Blue)

    return (
      <Tag
        onClick={e => {
          if (onClick) {
            e.preventDefault()
            e.stopPropagation()
            onClick()
          }
        }}
        className={cx(css.labelTag, className, { [css.removeBtnTag]: removeLabelBtn })}>
        <Layout.Horizontal flex={{ alignItems: 'center' }}>
          <Container
            style={{
              border: `1px solid ${colorObj.stroke}`
            }}
            className={css.labelKey}>
            {scopeIcon && (
              <Icon
                name={scopeIcon}
                size={12}
                style={{
                  color: `${colorObj.text}`
                }}
              />
            )}
            <Text
              style={{
                color: `${colorObj.text}`
              }}
              font={{ variation: FontVariation.SMALL_SEMI }}
              lineClamp={1}>
              {name}
            </Text>
          </Container>
          <Layout.Horizontal
            className={css.labelValue}
            style={{
              backgroundColor: `${colorObj.backgroundWithoutStroke}`
            }}
            flex={{ alignItems: 'center' }}>
            <Text
              style={{
                color: `${colorObj.text}`,
                backgroundColor: `${colorObj.backgroundWithoutStroke}`
              }}
              lineClamp={1}
              className={css.labelValueTxt}
              font={{ variation: FontVariation.SMALL_SEMI }}>
              {valueName}
            </Text>
            {removeLabelBtn && (
              <Button
                variation={ButtonVariation.ICON}
                minimal
                icon="main-close"
                role="close"
                color={colorObj.backgroundWithoutStroke}
                iconProps={{ size: 8 }}
                size={ButtonSize.SMALL}
                onClick={() => {
                  if (handleRemoveClick && disableRemoveBtnTooltip) handleRemoveClick()
                }}
                tooltip={
                  <Menu style={{ minWidth: 'unset' }}>
                    <Menu.Item
                      text={getString('labels.removeLabel')}
                      key={getString('labels.removeLabel')}
                      className={cx(css.danger, css.isDark)}
                      onClick={handleRemoveClick}
                    />
                  </Menu>
                }
                tooltipProps={{ disabled: disableRemoveBtnTooltip, interactionKind: 'click', isDark: true }}
              />
            )}
          </Layout.Horizontal>
        </Layout.Horizontal>
      </Tag>
    )
  } else {
    const colorObj = getColorsObj(label_color ?? ColorName.Blue)
    return (
      <Tag
        onClick={e => {
          if (onClick) {
            e.preventDefault()
            e.stopPropagation()
            onClick()
          }
        }}
        className={cx(css.labelTag, className, { [css.removeBtnTag]: removeLabelBtn })}>
        <Layout.Horizontal
          className={css.standaloneKey}
          flex={{ alignItems: 'center' }}
          style={{
            color: `${colorObj.text}`,
            border: `1px solid ${colorObj.stroke}`
          }}>
          {scopeIcon && (
            <Icon
              name={scopeIcon}
              size={12}
              style={{
                color: `${colorObj.text}`
              }}
            />
          )}
          <Text
            style={{
              color: `${colorObj.text}`
            }}
            lineClamp={1}
            font={{ variation: FontVariation.SMALL_SEMI }}>
            {name}
          </Text>
          {removeLabelBtn && (
            <Button
              variation={ButtonVariation.ICON}
              minimal
              icon="main-close"
              role="close"
              color={colorObj.backgroundWithoutStroke}
              iconProps={{ size: 8 }}
              size={ButtonSize.SMALL}
              onClick={() => {
                if (handleRemoveClick && disableRemoveBtnTooltip) handleRemoveClick()
              }}
              tooltip={
                <Menu style={{ minWidth: 'unset' }}>
                  <Menu.Item
                    text={getString('labels.removeLabel')}
                    key={getString('labels.removeLabel')}
                    className={cx(css.danger, css.isDark)}
                    onClick={handleRemoveClick}
                  />
                </Menu>
              }
              tooltipProps={{ disabled: disableRemoveBtnTooltip, interactionKind: 'click', isDark: true }}
            />
          )}
        </Layout.Horizontal>
      </Tag>
    )
  }
}

export const LabelTitle: React.FC<LabelTitleProps> = props => {
  const { name, scope, label_color, value_count, labelType } = props
  const { standalone } = useAppContext()
  const colorObj = getColorsObj(label_color ?? ColorName.Blue)
  const scopeIcon = getScopeIcon(scope, standalone)
  if (value_count || (labelType && labelType === LabelType.DYNAMIC)) {
    return (
      <Tag className={css.labelTag}>
        <Layout.Horizontal flex={{ alignItems: 'center' }}>
          <Container
            style={{
              border: `1px solid ${colorObj.stroke}`
            }}
            className={css.labelKey}>
            {scopeIcon && (
              <Icon
                style={{
                  color: `${colorObj.text}`
                }}
                name={scopeIcon}
                size={12}
              />
            )}
            <Text
              style={{
                color: `${colorObj.text}`
              }}
              lineClamp={1}
              font={{ variation: FontVariation.SMALL_SEMI }}>
              {name}
            </Text>
          </Container>
          <Text
            style={{
              color: `${colorObj.text}`,
              backgroundColor: `${colorObj.backgroundWithoutStroke}`
            }}
            className={css.labelValue}
            font={{ variation: FontVariation.SMALL_SEMI }}>
            ... ({value_count ?? 0})
          </Text>
        </Layout.Horizontal>
      </Tag>
    )
  } else {
    return (
      <Tag className={css.labelTag}>
        <Layout.Horizontal
          className={css.standaloneKey}
          flex={{ alignItems: 'center' }}
          style={{
            border: `1px solid ${colorObj.stroke}`
          }}>
          {scopeIcon && (
            <Icon
              style={{
                color: `${colorObj.text}`
              }}
              name={scopeIcon}
              size={12}
            />
          )}
          <Text
            style={{
              color: `${colorObj.text}`
            }}
            lineClamp={1}
            font={{ variation: FontVariation.SMALL_SEMI }}>
            {name}
          </Text>
        </Layout.Horizontal>
      </Tag>
    )
  }
}

export const LabelValuesList: React.FC<{
  name: string
  scope: number
  repoMetadata?: RepoRepositoryOutput
  space?: string
  standalone: boolean
}> = ({ name, scope, repoMetadata, space = '', standalone }) => {
  const { scopeRef } = getScopeData(space as string, scope, standalone)
  const getPath = () =>
    scope === 0
      ? `/repos/${encodeURIComponent(repoMetadata?.path as string)}/labels/${encodeURIComponent(name)}/values`
      : `/spaces/${encodeURIComponent(scopeRef)}/labels/${encodeURIComponent(name)}/values`

  const {
    data: labelValues,
    refetch: refetchLabelValues,
    loading: loadingLabelValues
  } = useGet<TypesLabelValue[]>({
    base: getConfig('code/api/v1'),
    path: getPath(),
    lazy: true
  })

  useEffect(() => {
    refetchLabelValues()
  }, [name, scope, space, repoMetadata])

  return (
    <Layout.Horizontal className={css.valuesList}>
      {!loadingLabelValues && labelValues && !isEmpty(labelValues) ? (
        labelValues.map(value => (
          <Label
            key={`${name}-${value.value}`}
            name={name}
            scope={scope}
            label_value={{ name: value.value, color: value.color as ColorName }}
          />
        ))
      ) : (
        <Spinner size={16} />
      )}
    </Layout.Horizontal>
  )
}

// ToDo : Remove LabelValuesListQuery component when Encoding is handled by BE for Harness
export const LabelValuesListQuery: React.FC<{
  name: string
  scope: number
  repoMetadata?: RepoRepositoryOutput
  space?: string
  standalone: boolean
}> = ({ name, scope, repoMetadata, space = '', standalone }) => {
  const { scopeRef } = getScopeData(space as string, scope, standalone)

  const getPath = () =>
    scope === 0
      ? `/repos/${repoMetadata?.identifier}/labels/${encodeURIComponent(name)}/values`
      : `/labels/${encodeURIComponent(name)}/values`

  const {
    data: labelValues,
    refetch: refetchLabelValues,
    loading: loadingLabelValues
  } = useGet<TypesLabelValue[]>({
    base: getConfig('code/api/v1'),
    path: getPath(),
    queryParams: {
      accountIdentifier: scopeRef?.split('/')[0],
      orgIdentifier: scopeRef?.split('/')[1],
      projectIdentifier: scopeRef?.split('/')[2]
    },
    lazy: true
  })

  useEffect(() => {
    refetchLabelValues()
  }, [name, scope, space, repoMetadata])

  return (
    <Layout.Horizontal className={css.valuesList}>
      {!loadingLabelValues && labelValues && !isEmpty(labelValues) ? (
        labelValues.map(value => (
          <Label
            key={`${name}-${value.value}`}
            name={name}
            scope={scope}
            label_value={{ name: value.value, color: value.color as ColorName }}
          />
        ))
      ) : (
        <Spinner size={16} />
      )}
    </Layout.Horizontal>
  )
}
