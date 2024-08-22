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

import React, { useMemo } from 'react'
import { get } from 'lodash-es'
import classNames from 'classnames'
import { Container } from '@harnessio/uicore'
import type { FormikProps } from 'formik'

import SelectList from './SelectList'
import SelectedList from './SelectedList'
import type { RorderSelectOption } from './types'
import css from './ReorderSelect.module.scss'

interface ReorderSelectProps<T> {
  name: string
  formikProps: FormikProps<T>
  className: string
  items: RorderSelectOption[]
  disabled: boolean
  selectListProps?: {
    title?: string | React.ReactNode
    withSearch?: boolean
  }
  selectedListProps?: {
    title?: string | React.ReactNode
    note?: string | React.ReactNode
  }
}

export default function ReorderSelect<T>(props: ReorderSelectProps<T>): JSX.Element {
  const { name, formikProps, className, selectListProps, selectedListProps, items, disabled } = props
  const formValue = useMemo(() => {
    return get(formikProps.values, name) || []
  }, [formikProps, name])

  const itemsMap: { [key: string]: RorderSelectOption } = useMemo(() => {
    return items.reduce(
      (acc, curr) => ({
        ...acc,
        [curr.value]: curr
      }),
      {}
    )
  }, [items])

  const selectedOptions: RorderSelectOption[] = useMemo(() => {
    if (Array.isArray(formValue)) {
      return formValue.map(each => {
        if (itemsMap[each]) {
          return itemsMap[each]
        }
        return {
          value: each,
          label: each,
          disabled: false
        }
      })
    }
    return []
  }, [formValue, itemsMap])

  const handleAddOption = (option: RorderSelectOption): void => {
    if (Array.isArray(formValue)) {
      formikProps.setFieldValue(name, [...formValue, option.value])
    } else {
      formikProps.setFieldValue(name, [option.value])
    }
  }

  const handleReorderList = (options: RorderSelectOption[]): void => {
    formikProps.setFieldValue(
      name,
      options.map(each => each.value)
    )
  }

  const handleRemoveItem = (option: RorderSelectOption): void => {
    const newSelectedOptions = selectedOptions.filter(each => each.value !== option.value)
    const newFormValue = newSelectedOptions.map(each => each.value)
    formikProps.setFieldValue(name, newFormValue)
  }

  return (
    <Container className={classNames(css.conatiner, className)}>
      <SelectList
        title={selectListProps?.title}
        withSearch={selectListProps?.withSearch}
        options={items.filter(each => !formValue.includes(each.value))}
        onSelect={handleAddOption}
        disabled={disabled}
      />
      <SelectedList
        title={selectedListProps?.title}
        options={selectedOptions}
        onReorder={handleReorderList}
        note={selectedListProps?.note}
        onRemove={handleRemoveItem}
        disabled={disabled}
      />
    </Container>
  )
}
