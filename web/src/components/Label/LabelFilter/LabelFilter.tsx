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

import React, { useEffect, useState } from 'react'
import {
  Container,
  Layout,
  FlexExpander,
  DropDown,
  ButtonVariation,
  Button,
  Text,
  SelectOption,
  useToaster,
  stringSubstitute,
  TextInput
} from '@harnessio/uicore'
import cx from 'classnames'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { Menu, MenuItem, PopoverInteractionKind, PopoverPosition, Spinner } from '@blueprintjs/core'
import { isEmpty, noop } from 'lodash-es'
import { getConfig, getUsingFetch } from 'services/config'
import type { EnumLabelColor, RepoRepositoryOutput, TypesLabel, TypesLabelValue } from 'services/code'
import {
  ColorName,
  LIST_FETCHING_LIMIT,
  LabelFilterObj,
  LabelFilterType,
  getErrorMessage,
  getScopeData
} from 'utils/Utils'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { Label, LabelTitle } from '../Label'
import css from './LabelFilter.module.scss'

interface LabelFilterProps {
  labelFilterOption?: LabelFilterObj[]
  setLabelFilterOption: React.Dispatch<React.SetStateAction<LabelFilterObj[] | undefined>>
  onPullRequestLabelFilterChanged: (labelFilter: LabelFilterObj[]) => void
  bearerToken: string
  repoMetadata: RepoRepositoryOutput
  spaceRef: string
}

enum utilFilterType {
  LABEL = 'label',
  VALUE = 'value',
  FOR_VALUE = 'for_value'
}

const mapToSelectOptions = (items: TypesLabelValue[] | TypesLabel[] = []) =>
  items.map(item => ({
    label: JSON.stringify(item),
    value: String(item?.id)
  })) as SelectOption[]

export const LabelFilter = (props: LabelFilterProps) => {
  const {
    labelFilterOption,
    setLabelFilterOption,
    onPullRequestLabelFilterChanged,
    bearerToken,
    repoMetadata,
    spaceRef
  } = props
  const { showError } = useToaster()
  const { standalone, routingId } = useAppContext()
  const [loadingLabels, setLoadingLabels] = useState(false)
  const [loadingLabelValues, setLoadingLabelValues] = useState(false)
  const [labelValues, setLabelValues] = useState<SelectOption[]>()
  const [labelQuery, setLabelQuery] = useState<string>('')
  const [highlightItem, setHighlightItem] = useState('')
  const [isVisible, setIsVisible] = useState(true)
  const [valueQuery, setValueQuery] = useState('')
  const [labelItems, setLabelItems] = useState<SelectOption[]>()
  const { getString } = useStrings()

  const getDropdownLabels = async (currentFilterOption?: LabelFilterObj[]) => {
    try {
      const fetchedLabels: TypesLabel[] = await getUsingFetch(
        getConfig('code/api/v1'),
        `/repos/${repoMetadata?.path}/+/labels`,
        bearerToken,
        {
          queryParams: {
            page: 1,
            limit: LIST_FETCHING_LIMIT,
            inherited: true,
            query: labelQuery?.trim(),
            accountIdentifier: routingId
          }
        }
      )
      const updateLabelsList = mapToSelectOptions(fetchedLabels)
      const labelForTop = mapToSelectOptions(currentFilterOption?.map(({ labelObj }) => labelObj))
      const mergedArray = [...labelForTop, ...updateLabelsList]
      return Array.from(new Map(mergedArray.map(item => [item.value, item])).values())
    } catch (error) {
      showError(getErrorMessage(error))
    }
  }

  useEffect(() => {
    setLoadingLabels(true)
    getDropdownLabels(labelFilterOption)
      .then(res => {
        setLabelItems(res)
      })
      .finally(() => {
        setLoadingLabels(false)
      })
      .catch(error => showError(getErrorMessage(error)))
  }, [labelFilterOption, labelQuery])

  const getLabelValuesPromise = async (key: string, scope: number): Promise<SelectOption[]> => {
    setLoadingLabelValues(true)
    const { scopeRef } = getScopeData(spaceRef, scope, standalone)
    const getPath = () =>
      scope === 0
        ? `/repos/${encodeURIComponent(repoMetadata?.path as string)}/labels/${encodeURIComponent(key)}/values`
        : `/spaces/${encodeURIComponent(scopeRef)}/labels/${encodeURIComponent(key)}/values`

    try {
      const fetchedValues: TypesLabelValue[] = await getUsingFetch(getConfig('code/api/v1'), getPath(), bearerToken, {
        queryParams: { accountIdentifier: routingId }
      })
      const updatedValuesList = mapToSelectOptions(fetchedValues)
      setLoadingLabelValues(false)
      return updatedValuesList
    } catch (error) {
      setLoadingLabelValues(false)
      showError(getErrorMessage(error))
      return []
    }
  }

  // ToDo : Remove getLabelValuesPromiseQuery component when Encoding is handled by BE for Harness
  const getLabelValuesPromiseQuery = async (key: string, scope: number): Promise<SelectOption[]> => {
    setLoadingLabelValues(true)
    const { scopeRef } = getScopeData(spaceRef, scope, standalone)
    const getPath = () =>
      scope === 0
        ? `/repos/${repoMetadata?.identifier}/labels/${encodeURIComponent(key)}/values`
        : `/labels/${encodeURIComponent(key)}/values`

    try {
      const fetchedValues: TypesLabelValue[] = await getUsingFetch(getConfig('code/api/v1'), getPath(), bearerToken, {
        queryParams: {
          accountIdentifier: scopeRef?.split('/')[0],
          orgIdentifier: scopeRef?.split('/')[1],
          projectIdentifier: scopeRef?.split('/')[2]
        }
      })
      const updatedValuesList = mapToSelectOptions(fetchedValues)
      setLoadingLabelValues(false)
      return updatedValuesList
    } catch (error) {
      setLoadingLabelValues(false)
      showError(getErrorMessage(error))
      return []
    }
  }

  const containsFilter = (filterObjArr: LabelFilterObj[], currentObj: any, type: utilFilterType) => {
    let res = false
    if (filterObjArr && filterObjArr.length === 0) return res
    else if (type === utilFilterType.LABEL) {
      res = filterObjArr.some(
        filterObj =>
          filterObj.labelId === currentObj.id &&
          filterObj.valueId === undefined &&
          filterObj.type === LabelFilterType.LABEL
      )
    } else if (type === utilFilterType.VALUE) {
      const labelId = currentObj?.valueId === -1 ? currentObj.labelId : currentObj.label_id
      const valueId = currentObj?.valueId === -1 ? currentObj.valueId : currentObj.id
      res = filterObjArr.some(
        filterObj =>
          filterObj.labelId === labelId && filterObj.valueId === valueId && filterObj.type === LabelFilterType.VALUE
      )
    } else if (type === utilFilterType.FOR_VALUE) {
      res = filterObjArr.some(
        filterObj =>
          filterObj.labelId === currentObj.id &&
          filterObj.valueId !== undefined &&
          filterObj.type === LabelFilterType.VALUE
      )
    }
    return res
  }

  const replaceValueFilter = (filterObjArr: LabelFilterObj[], currentObj: any) => {
    const updateFilterObjArr = filterObjArr.map(filterObj => {
      if (filterObj.labelId === currentObj.label_id && filterObj.type === LabelFilterType.VALUE) {
        return { ...filterObj, valueId: currentObj.id, valueObj: currentObj }
      }
      return filterObj
    })
    onPullRequestLabelFilterChanged([...updateFilterObjArr])
    setLabelFilterOption([...updateFilterObjArr])
  }

  const removeValueFromFilter = (filterObjArr: LabelFilterObj[], currentObj: any) => {
    const updateFilterObjArr = filterObjArr.filter(filterObj => {
      if (!(filterObj.labelId === currentObj.label_id && filterObj.type === LabelFilterType.VALUE)) {
        return filterObj
      }
    })
    onPullRequestLabelFilterChanged(updateFilterObjArr)
    setLabelFilterOption(updateFilterObjArr)
  }

  const removeLabelFromFilter = (filterObjArr: LabelFilterObj[], currentObj: any) => {
    const updateFilterObjArr = filterObjArr.filter(filterObj => {
      if (!(filterObj.labelId === currentObj.id && filterObj.type === LabelFilterType.LABEL)) {
        return filterObj
      }
    })
    onPullRequestLabelFilterChanged(updateFilterObjArr)
    setLabelFilterOption(updateFilterObjArr)
  }

  return (
    <DropDown
      value={
        labelFilterOption && !isEmpty(labelFilterOption)
          ? (labelFilterOption[labelFilterOption.length - 1].labelId as unknown as string)
          : (labelFilterOption?.length as unknown as string)
      }
      items={labelItems}
      disabled={loadingLabels}
      onChange={noop}
      popoverClassName={css.labelDropdownPopover}
      icon={!isEmpty(labelFilterOption) ? undefined : 'code-tag'}
      iconProps={{ size: 16 }}
      placeholder={getString('labels.filterByLabels')}
      resetOnClose
      resetOnSelect
      resetOnQuery
      query={labelQuery}
      onQueryChange={newQuery => {
        setLabelQuery(newQuery)
      }}
      itemRenderer={(item, { handleClick }) => {
        const itemObj = JSON.parse(item.label)
        const offsetValue = labelFilterOption && containsFilter(labelFilterOption, itemObj, utilFilterType.FOR_VALUE)
        const offsetLabel = labelFilterOption && containsFilter(labelFilterOption, itemObj, utilFilterType.LABEL)
        const anyValueObj = {
          labelId: itemObj.id as number,
          type: LabelFilterType.VALUE,
          valueId: -1,
          labelObj: itemObj,
          valueObj: {
            id: -1,
            color: itemObj.color as EnumLabelColor,
            label_id: itemObj.id,
            value: getString('labels.anyValueOption')
          }
        }

        const filteredLabelValues = (filterQuery: string) => {
          if (!filterQuery) {
            return labelValues
          }
          const lowerCaseQuery = filterQuery.toLowerCase()
          return labelValues?.filter((value: any) => {
            const valueObj = JSON.parse(value.label)
            return valueObj.value?.toLowerCase().includes(lowerCaseQuery)
          })
        }
        const labelsValueList = filteredLabelValues(valueQuery)

        return (
          <Container
            onMouseEnter={() => (item.label !== highlightItem ? setIsVisible(false) : setIsVisible(true))}
            className={cx(css.labelCtn, { [css.highlight]: highlightItem === item.label })}>
            {itemObj.value_count ? (
              <Button
                className={css.labelBtn}
                text={
                  labelFilterOption?.length ? (
                    <Layout.Horizontal
                      className={css.offsetcheck}
                      spacing={'small'}
                      flex={{ alignItems: 'center', justifyContent: 'space-between' }}
                      width={'100%'}>
                      <Icon name={'tick'} size={16} style={{ opacity: offsetValue ? 1 : 0 }} />
                      <FlexExpander />
                      <LabelTitle
                        name={itemObj?.key as string}
                        value_count={itemObj.value_count}
                        label_color={itemObj.color as ColorName}
                        scope={itemObj.scope}
                      />
                    </Layout.Horizontal>
                  ) : (
                    <LabelTitle
                      name={itemObj?.key as string}
                      value_count={itemObj.value_count}
                      label_color={itemObj.color as ColorName}
                      scope={itemObj.scope}
                    />
                  )
                }
                rightIcon={'chevron-right'}
                iconProps={{ size: 16 }}
                variation={ButtonVariation.LINK}
                onClick={e => {
                  e.preventDefault()
                  setIsVisible(true)
                  setValueQuery('')
                  setHighlightItem(item.label as string)
                  // ToDo : Remove this check once BE has support for encoding
                  if (standalone) {
                    getLabelValuesPromise(itemObj.key, itemObj.scope)
                      .then(res => setLabelValues(res))
                      .catch(err => {
                        showError(getErrorMessage(err))
                      })
                  } else {
                    getLabelValuesPromiseQuery(itemObj.key, itemObj.scope)
                      .then(res => setLabelValues(res))
                      .catch(err => {
                        showError(getErrorMessage(err))
                      })
                  }
                }}
                tooltip={
                  labelsValueList && !loadingLabelValues ? (
                    <Menu key={itemObj.id} className={css.childBox}>
                      <TextInput
                        className={css.input}
                        wrapperClassName={css.inputBox}
                        value={valueQuery}
                        autoFocus
                        placeholder={getString('labels.findaValue')}
                        onInput={e => {
                          const _value = e.currentTarget.value || ''
                          setValueQuery(_value)
                        }}
                        leftIcon={'thinner-search'}
                        leftIconProps={{
                          name: 'thinner-search',
                          size: 12,
                          color: Color.GREY_500
                        }}
                        rightElement={valueQuery ? 'main-close' : undefined}
                        rightElementProps={{
                          onClick: () => setValueQuery(''),
                          className: css.closeBtn,
                          size: 8,
                          color: Color.GREY_300
                        }}
                      />
                      <MenuItem
                        key={itemObj.key + getString('labels.anyValue')}
                        onClick={event => {
                          if (offsetValue) {
                            if (containsFilter(labelFilterOption, anyValueObj, utilFilterType.VALUE)) {
                              removeValueFromFilter(labelFilterOption, anyValueObj.valueObj)
                            } else {
                              replaceValueFilter(labelFilterOption, anyValueObj.valueObj)
                            }
                          } else if (labelFilterOption) {
                            onPullRequestLabelFilterChanged([...labelFilterOption, anyValueObj])
                            setLabelFilterOption([...labelFilterOption, anyValueObj])
                          }

                          handleClick(event)
                        }}
                        className={cx(css.menuItem)}
                        text={
                          offsetValue ? (
                            <Layout.Horizontal
                              className={css.offsetcheck}
                              spacing={'small'}
                              flex={{ alignItems: 'center', justifyContent: 'flex-start' }}
                              width={'100%'}>
                              <Icon
                                name={'tick'}
                                size={16}
                                color={Color.PRIMARY_7}
                                style={{
                                  opacity: containsFilter(labelFilterOption, anyValueObj, utilFilterType.VALUE) ? 1 : 0
                                }}
                              />
                              <Label
                                key={itemObj.key + (anyValueObj.valueObj.value as string)}
                                name={itemObj.key}
                                label_value={{
                                  name: anyValueObj.valueObj.value,
                                  color: anyValueObj.valueObj.color as ColorName
                                }}
                                scope={itemObj.scope}
                              />
                            </Layout.Horizontal>
                          ) : (
                            <Label
                              key={itemObj.key + (anyValueObj.valueObj.value as string)}
                              name={itemObj.key}
                              label_value={{
                                name: anyValueObj.valueObj.value,
                                color: anyValueObj.valueObj.color as ColorName
                              }}
                              scope={itemObj.scope}
                            />
                          )
                        }
                      />

                      {labelsValueList.map(value => {
                        const valueObj = JSON.parse(value.label)
                        const currentMarkedValue = labelFilterOption
                          ? containsFilter(labelFilterOption, valueObj, utilFilterType.VALUE)
                          : {}
                        const updatedValueFilterOption = labelFilterOption
                          ? [
                              ...labelFilterOption,
                              {
                                labelId: valueObj.label_id,
                                valueId: valueObj.id,
                                type: LabelFilterType.VALUE,
                                labelObj: itemObj,
                                valueObj: valueObj
                              }
                            ]
                          : []
                        return (
                          <MenuItem
                            key={itemObj.key + (value.value as string) + 'menu'}
                            onClick={event => {
                              if (offsetValue) {
                                if (currentMarkedValue) {
                                  removeValueFromFilter(labelFilterOption, valueObj)
                                } else {
                                  replaceValueFilter(labelFilterOption, valueObj)
                                }
                              } else {
                                onPullRequestLabelFilterChanged(updatedValueFilterOption)
                                setLabelFilterOption(updatedValueFilterOption)
                              }

                              handleClick(event)
                            }}
                            className={cx(css.menuItem)}
                            text={
                              offsetValue ? (
                                <Layout.Horizontal
                                  className={css.offsetcheck}
                                  spacing={'small'}
                                  flex={{ alignItems: 'center', justifyContent: 'flex-start' }}
                                  width={'100%'}>
                                  <Icon
                                    name={'tick'}
                                    size={16}
                                    color={Color.PRIMARY_7}
                                    style={{ opacity: currentMarkedValue ? 1 : 0 }}
                                  />
                                  <Label
                                    key={itemObj.key + (value.value as string)}
                                    name={itemObj.key}
                                    label_value={{ name: valueObj.value, color: valueObj.color as ColorName }}
                                    scope={itemObj.scope}
                                  />
                                </Layout.Horizontal>
                              ) : (
                                <Label
                                  key={itemObj.key + (value.value as string)}
                                  name={itemObj.key}
                                  label_value={{ name: valueObj.value, color: valueObj.color as ColorName }}
                                  scope={itemObj.scope}
                                />
                              )
                            }
                          />
                        )
                      })}
                    </Menu>
                  ) : (
                    <Menu className={css.menuItem} style={{ justifyContent: 'center' }}>
                      <Spinner size={20} />
                    </Menu>
                  )
                }
                tooltipProps={{
                  interactionKind: PopoverInteractionKind.CLICK,
                  position: PopoverPosition.RIGHT,
                  popoverClassName: cx(css.popover, { [css.hide]: !isVisible }),
                  modifiers: { preventOverflow: { boundariesElement: 'viewport' } }
                }}
              />
            ) : (
              <Container
                onClick={event => {
                  handleClick(event)
                  const updatedLabelFilterOption = Array.isArray(labelFilterOption)
                    ? [
                        ...labelFilterOption,
                        {
                          labelId: itemObj.id,
                          valueId: undefined,
                          type: LabelFilterType.LABEL,
                          labelObj: itemObj,
                          valueObj: undefined
                        }
                      ]
                    : ([] as LabelFilterObj[] | undefined)
                  if (offsetLabel) removeLabelFromFilter(labelFilterOption, itemObj)
                  else {
                    onPullRequestLabelFilterChanged(
                      updatedLabelFilterOption ? [...updatedLabelFilterOption] : ([] as LabelFilterObj[])
                    )
                    setLabelFilterOption(
                      updatedLabelFilterOption ? [...updatedLabelFilterOption] : ([] as LabelFilterObj[])
                    )
                  }
                }}>
                <Button
                  className={css.labelBtn}
                  text={
                    labelFilterOption?.length ? (
                      <Layout.Horizontal
                        className={css.offsetcheck}
                        spacing={'small'}
                        flex={{ alignItems: 'center', justifyContent: 'space-between' }}
                        width={'100%'}>
                        <Icon name={'tick'} size={16} style={{ opacity: offsetLabel ? 1 : 0 }} />
                        <FlexExpander />
                        <LabelTitle
                          name={itemObj?.key as string}
                          value_count={itemObj.value_count}
                          label_color={itemObj.color as ColorName}
                          scope={itemObj.scope}
                        />
                      </Layout.Horizontal>
                    ) : (
                      <Layout.Horizontal>
                        <LabelTitle
                          name={itemObj?.key as string}
                          value_count={itemObj.value_count}
                          label_color={itemObj.color as ColorName}
                          scope={itemObj.scope}
                        />
                      </Layout.Horizontal>
                    )
                  }
                  variation={ButtonVariation.LINK}
                />
              </Container>
            )}
          </Container>
        )
      }}
      getCustomLabel={() => {
        return (
          <Layout.Horizontal spacing="small" flex={{ alignItems: 'center' }}>
            <Text className={css.counter}>{labelFilterOption?.length}</Text>
            <Text lineClamp={1} color={Color.GREY_900} font={{ variation: FontVariation.BODY }}>
              {
                stringSubstitute(getString('labels.labelsApplied'), {
                  labelCount: labelFilterOption?.length
                }) as string
              }
            </Text>
          </Layout.Horizontal>
        )
      }}
    />
  )
}
