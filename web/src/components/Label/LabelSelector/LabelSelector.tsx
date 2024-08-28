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

import React, { useEffect, useRef, useState } from 'react'
import {
  Button,
  ButtonProps,
  ButtonSize,
  ButtonVariation,
  Container,
  Layout,
  Tag,
  Text,
  TextInput,
  useToaster
} from '@harnessio/uicore'
import cx from 'classnames'
import { Menu, MenuItem, PopoverPosition } from '@blueprintjs/core'
import { useMutate } from 'restful-react'
import { Render } from 'react-jsx-match'
import { isEmpty } from 'lodash-es'
import { useAppContext } from 'AppContext'
import type {
  RepoRepositoryOutput,
  TypesLabelAssignment,
  TypesLabelValueInfo,
  TypesPullReq,
  TypesScopesLabels
} from 'services/code'
import { useStrings } from 'framework/strings'
import { ButtonRoleProps, ColorName, LabelType, getErrorMessage, permissionProps } from 'utils/Utils'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { getConfig } from 'services/config'
import { Label, LabelTitle } from '../Label'
import css from './LabelSelector.module.scss'

export interface LabelSelectorProps {
  allLabelsData: TypesScopesLabels | null
  refetchLabels: () => void
  refetchlabelsList: () => void
  repoMetadata: RepoRepositoryOutput
  pullRequestMetadata: TypesPullReq
  setQuery: React.Dispatch<React.SetStateAction<string>>
  query: string
  labelListLoading: boolean
  refetchActivities: () => void
}

export interface LabelSelectProps extends Omit<ButtonProps, 'onSelect'> {
  onSelectLabel: (label: TypesLabelAssignment) => void
  onSelectValue: (labelId: number, valueId: number, labelKey: string, valueKey: string) => void
  menuState?: LabelsMenuState
  currentLabel: TypesLabelAssignment
  handleValueRemove?: () => void
  addNewValue?: () => void
  allLabelsData: TypesScopesLabels | null
  query: string
  setQuery: React.Dispatch<React.SetStateAction<string>>
  menuItemIndex: number
  setMenuItemIndex: React.Dispatch<React.SetStateAction<number>>
  labelListLoading: boolean
}

enum LabelsMenuState {
  LABELS = 'labels',
  VALUES = 'label_values'
}

export const LabelSelector: React.FC<LabelSelectorProps> = ({
  allLabelsData,
  refetchLabels,
  pullRequestMetadata,
  repoMetadata,
  refetchlabelsList,
  query,
  setQuery,
  labelListLoading,
  refetchActivities,
  ...props
}) => {
  const [popoverDialogOpen, setPopoverDialogOpen] = useState<boolean>(false)
  const [menuState, setMenuState] = useState<LabelsMenuState>(LabelsMenuState.LABELS)
  const [menuItemIndex, setMenuItemIndex] = useState<number>(0)
  const [currentLabel, setCurrentLabel] = useState<TypesLabelAssignment>({ key: '', id: -1 })
  const { getString } = useStrings()

  const { showError, showSuccess } = useToaster()
  const { mutate: updatePRLabels } = useMutate({
    verb: 'PUT',
    base: getConfig('code/api/v1'),
    path: `/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata.number}/labels`
  })

  const space = useGetSpaceParam()
  const { hooks, standalone } = useAppContext()
  const permPushResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY',
        resourceIdentifier: repoMetadata?.identifier as string
      },
      permissions: ['code_repo_push']
    },
    [space]
  )

  return (
    <Button
      className={css.addLabelBtn}
      text={<span className={css.prefix}>{getString('add')}</span>}
      variation={ButtonVariation.TERTIARY}
      minimal
      disabled={labelListLoading}
      size={ButtonSize.SMALL}
      tooltip={
        <PopoverContent
          onSelectLabel={label => {
            setCurrentLabel(label)
            if (label.values?.length || label.type === LabelType.DYNAMIC) {
              setMenuState(LabelsMenuState.VALUES)
              setMenuItemIndex(0)
            } else {
              try {
                updatePRLabels({
                  label_id: label.id
                })
                  .then(() => {
                    refetchLabels()
                    setQuery('')
                    setPopoverDialogOpen(false)
                    showSuccess(`Applied '${label.key}' label`)
                    refetchActivities()
                  })
                  .catch(error => showError(getErrorMessage(error)))
              } catch (exception) {
                showError(getErrorMessage(exception))
              }
            }
          }}
          onSelectValue={(labelId, valueId, labelKey, valueKey) => {
            setMenuState(LabelsMenuState.VALUES)
            setMenuItemIndex(0)
            try {
              updatePRLabels({
                label_id: labelId,
                value_id: valueId
              })
                .then(() => {
                  refetchLabels()
                  setMenuState(LabelsMenuState.LABELS)
                  setMenuItemIndex(0)
                  setCurrentLabel({ key: '', id: -1 })
                  setPopoverDialogOpen(false)
                  showSuccess(`Applied '${labelKey}:${valueKey}' label`)
                  refetchActivities()
                })
                .catch(error => showError(getErrorMessage(error)))
            } catch (exception) {
              showError(getErrorMessage(exception))
            }
          }}
          allLabelsData={allLabelsData}
          menuState={menuState}
          currentLabel={currentLabel}
          addNewValue={() => {
            try {
              updatePRLabels({
                label_id: currentLabel.id,
                value: query
              })
                .then(() => {
                  showSuccess(`Updated ${currentLabel.key} with ${query}`)
                  refetchLabels()
                  refetchlabelsList()
                  setMenuState(LabelsMenuState.LABELS)
                  setMenuItemIndex(0)
                  setCurrentLabel({ key: '', id: -1 })
                  setPopoverDialogOpen(false)
                  setQuery('')
                  refetchActivities()
                })
                .catch(error => showError(getErrorMessage(error)))
            } catch (exception) {
              showError(getErrorMessage(exception))
            }
          }}
          query={query}
          setQuery={setQuery}
          handleValueRemove={() => {
            setMenuState(LabelsMenuState.LABELS)
            setMenuItemIndex(0)
            setCurrentLabel({ key: '', id: -1 })
          }}
          menuItemIndex={menuItemIndex}
          setMenuItemIndex={setMenuItemIndex}
          labelListLoading={labelListLoading}
        />
      }
      tooltipProps={{
        interactionKind: 'click',
        usePortal: true,
        position: PopoverPosition.BOTTOM_RIGHT,
        popoverClassName: css.popover,
        isOpen: popoverDialogOpen,
        onClose: () => {
          setMenuState(LabelsMenuState.LABELS)
          setMenuItemIndex(0)
          setCurrentLabel({ key: '', id: -1 })
          setQuery('')
        },
        onInteraction: nxtState => setPopoverDialogOpen(nxtState)
      }}
      tabIndex={0}
      {...props}
      {...permissionProps(permPushResult, standalone)}
    />
  )
}

const PopoverContent: React.FC<LabelSelectProps> = ({
  onSelectLabel,
  onSelectValue,
  menuState,
  currentLabel,
  handleValueRemove,
  allLabelsData,
  addNewValue,
  query,
  setQuery,
  menuItemIndex,
  setMenuItemIndex,
  labelListLoading
}) => {
  const inputRef = useRef<HTMLInputElement | null>()
  const { getString } = useStrings()
  const filteredLabelValues = (valueQuery: string) => {
    if (!valueQuery) return currentLabel?.values // If no query, return all names
    const lowerCaseQuery = valueQuery.toLowerCase()
    return currentLabel?.values?.filter(label => label.value?.toLowerCase().includes(lowerCaseQuery))
  }

  const labelsValueList = filteredLabelValues(query)
  const labelsList = allLabelsData?.label_data ?? []

  useEffect(() => {
    if (menuState === LabelsMenuState.LABELS && menuItemIndex > 0) {
      const previousLabel = labelsList[menuItemIndex - 1]
      if (previousLabel && previousLabel.key && previousLabel.id) {
        document
          .getElementById(previousLabel.key + previousLabel.id)
          ?.scrollIntoView({ behavior: 'auto', block: 'center' })
      }
    } else if (menuState === LabelsMenuState.VALUES && menuItemIndex > 0) {
      const previousValue = labelsValueList?.[menuItemIndex - 1]
      if (previousValue && previousValue.value && previousValue.id) {
        const elementId = previousValue.value + previousValue.id
        document.getElementById(elementId)?.scrollIntoView({ behavior: 'auto', block: 'center' })
      }
    }
  }, [menuItemIndex, menuState, labelsList, labelsValueList])

  // const handleKeyDownLabels: React.KeyboardEventHandler<HTMLInputElement> = e => {
  //   if (labelsList && labelsList.length !== 0) {
  //     e.preventDefault()
  //     switch (e.key) {
  //       case 'ArrowDown':
  //         setMenuItemIndex((index: number) => {
  //           return index + 1 > labelsList.length ? 1 : index + 1
  //         })
  //         break
  //       case 'ArrowUp':
  //         setMenuItemIndex((index: number) => {
  //           return index - 1 > 0 ? index - 1 : labelsList.length
  //         })
  //         break
  //       case 'Enter':
  //         if (labelsList[menuItemIndex - 1]) {
  //           onSelectLabel(labelsList[menuItemIndex - 1])
  //           setQuery('')
  //         }
  //         break
  //       default:
  //         break
  //     }
  //   }
  // }

  const handleKeyDownValue: React.KeyboardEventHandler<HTMLInputElement> = e => {
    if (e.key === 'Backspace' && !query && currentLabel) {
      setQuery('')
      handleValueRemove && handleValueRemove()
    }
    // } else if (labelsValueList && labelsValueList.length !== 0) {
    //   switch (e.key) {
    //     case 'ArrowDown':
    //       setMenuItemIndex((index: number) => {
    //         return index + 1 > labelsValueList.length ? 1 : index + 1
    //       })
    //       break
    //     case 'ArrowUp':
    //       setMenuItemIndex((index: number) => {
    //         return index - 1 > 0 ? index - 1 : labelsValueList.length
    //       })
    //       break
    //     case 'Enter':
    //       onSelectValue(
    //         currentLabel.id ?? -1,
    //         labelsValueList[menuItemIndex - 1].id ?? -1,
    //         currentLabel.key ?? '',
    //         labelsValueList[menuItemIndex - 1].value ?? ''
    //       )
    //       setQuery('')
    //       break
    //     default:
    //       break
    //   }
    // }
  }

  return (
    <Container padding="small" className={css.main}>
      <Layout.Vertical className={css.layout}>
        {menuState === LabelsMenuState.LABELS ? (
          <TextInput
            className={css.input}
            wrapperClassName={css.inputBox}
            value={query}
            inputRef={ref => (inputRef.current = ref)}
            autoFocus
            placeholder={getString('labels.findALabel')}
            onInput={e => {
              const _value = e.currentTarget.value || ''
              setQuery(_value)
            }}
            rightElement={query ? 'code-close' : undefined}
            rightElementProps={{
              onClick: () => setQuery(''),
              className: css.closeBtn,
              size: 20
            }}
            // onKeyDown={handleKeyDownLabels}
          />
        ) : (
          currentLabel &&
          handleValueRemove && (
            <Layout.Horizontal flex={{ alignItems: 'center' }} className={css.labelSearch}>
              <Container className={css.labelCtn}>
                <Label
                  name={currentLabel.key as string}
                  label_color={currentLabel.color as ColorName}
                  scope={currentLabel.scope}
                />
              </Container>
              <TextInput
                className={css.input}
                onKeyDown={handleKeyDownValue}
                wrapperClassName={css.inputBox}
                value={query}
                inputRef={ref => (inputRef.current = ref)}
                defaultValue={query}
                autoFocus
                placeholder={
                  currentLabel.type === LabelType.STATIC
                    ? getString('labels.findaValue')
                    : !isEmpty(currentLabel.values)
                    ? getString('labels.findOrAdd')
                    : getString('labels.addaValue')
                }
                onInput={e => {
                  const _value = e.currentTarget.value || ''
                  setQuery(_value)
                }}
                rightElement={query || currentLabel?.key ? 'code-close' : undefined}
                rightElementProps={{
                  onClick: () => {
                    setQuery('')
                    handleValueRemove()
                  },
                  className: css.closeBtn,
                  size: 20
                }}
              />
            </Layout.Horizontal>
          )
        )}

        <Container className={cx(css.menuContainer)}>
          <LabelList
            onSelectLabel={onSelectLabel}
            onSelectValue={onSelectValue}
            query={query}
            setQuery={setQuery}
            menuState={menuState}
            currentLabel={currentLabel}
            allLabelsData={labelsList}
            menuItemIndex={menuItemIndex}
            addNewValue={addNewValue}
            setMenuItemIndex={setMenuItemIndex}
            labelListLoading={labelListLoading}
          />
        </Container>
      </Layout.Vertical>
    </Container>
  )
}

interface LabelListProps extends Omit<LabelSelectProps, 'allLabelsData'> {
  query: string
  setQuery: React.Dispatch<React.SetStateAction<string>>
  setLoading?: React.Dispatch<React.SetStateAction<boolean>>
  menuItemIndex: number
  allLabelsData: TypesLabelAssignment[]
}

const LabelList = ({
  onSelectLabel,
  onSelectValue,
  query,
  setQuery,
  menuState,
  currentLabel,
  allLabelsData: labelsList,
  menuItemIndex,
  addNewValue,
  labelListLoading
}: LabelListProps) => {
  const { getString } = useStrings()
  if (menuState === LabelsMenuState.LABELS) {
    if (labelsList.length) {
      return (
        <Menu className={css.labelMenu}>
          {labelsList?.map((label, index: number) => {
            return (
              <MenuItem
                key={(label.key as string) + label.id}
                id={(label.key as string) + label.id}
                className={cx(css.menuItem, {
                  [css.selected]: index === menuItemIndex - 1
                })}
                text={
                  <LabelTitle
                    name={label.key as string}
                    value_count={label.values?.length}
                    label_color={label.color as ColorName}
                    scope={label.scope}
                    labelType={label.type as LabelType}
                  />
                }
                onClick={e => {
                  e.preventDefault()
                  e.stopPropagation()
                  onSelectLabel(label)
                  setQuery('')
                }}
                {...ButtonRoleProps}
              />
            )
          })}
        </Menu>
      )
    } else {
      return (
        <Container flex={{ align: 'center-center' }} padding="large">
          {!labelListLoading && (
            <Text className={css.noWrapText} flex padding={{ top: 'small' }}>
              <span>
                {query && <Tag> {query} </Tag>} {getString('labels.labelNotFound')}
              </span>
            </Text>
          )}
        </Container>
      )
    }
  } else {
    const filteredLabelValues = (filterQuery: string) => {
      if (!filterQuery) return currentLabel?.values // If no query, return all names
      const lowerCaseQuery = filterQuery.toLowerCase()
      return currentLabel?.values?.filter(label => label.value?.toLowerCase().includes(lowerCaseQuery))
    }
    const matchFound = (userQuery: string, list?: TypesLabelValueInfo[]) => {
      const res = list ? list.map(ele => ele.value?.toLowerCase()).includes(userQuery.toLowerCase()) : false
      return res
    }
    const labelsValueList = filteredLabelValues(query)
    return (
      <Menu className={css.labelMenu}>
        <Render when={labelsValueList && currentLabel}>
          {labelsValueList?.map(({ value, id, color }, index: number) => (
            <MenuItem
              key={((value as string) + id) as string}
              id={((value as string) + id) as string}
              className={cx(css.menuItem, {
                [css.selected]: index === menuItemIndex - 1
              })}
              text={
                <Label
                  name={currentLabel.key as string}
                  label_color={currentLabel.color as ColorName}
                  label_value={{ name: value as string, color: color as ColorName }}
                  scope={currentLabel.scope}
                />
              }
              onClick={e => {
                e.preventDefault()
                e.stopPropagation()
                onSelectValue(currentLabel.id as number, id as number, currentLabel.key as string, value as string)
                setQuery('')
              }}
              {...ButtonRoleProps}
            />
          ))}
        </Render>
        <Render when={currentLabel.type === LabelType.DYNAMIC && !matchFound(query, labelsValueList) && query}>
          <Button
            variation={ButtonVariation.LINK}
            className={css.noWrapText}
            flex
            padding={{ top: 'small', left: 'small' }}
            onClick={() => {
              if (addNewValue) {
                addNewValue()
              }
            }}>
            <span className={css.valueNotFound}>
              {getString('labels.addNewValue')}
              {currentLabel && (
                <Label name={'...'} label_color={currentLabel.color as ColorName} label_value={{ name: query }} />
              )}
            </span>
          </Button>
        </Render>
        <Render when={labelsValueList?.length === 0 && currentLabel?.type === LabelType.STATIC}>
          <Text className={css.noWrapText} flex padding={{ top: 'small', left: 'small' }}>
            <span>
              {currentLabel && query && (
                <Label
                  name={currentLabel?.key as string}
                  label_color={currentLabel.color as ColorName}
                  label_value={{ name: query }}
                  scope={currentLabel.scope}
                />
              )}
              {getString('labels.labelNotFound')}
            </span>
          </Text>
        </Render>
      </Menu>
    )
  }
}
