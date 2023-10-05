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

import React, { useCallback, useEffect, useRef, useState } from 'react'
import { debounce, has, omit } from 'lodash-es'
import { FormikContextType, connect } from 'formik'
import { Layout, Text, FormInput, Button, ButtonVariation, ButtonSize, Container } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { Icon } from '@harnessio/icons'
import { useStrings } from 'framework/strings'

import css from './MultiList.module.scss'

interface MultiListConnectedProps extends MultiListProps {
  formik?: FormikContextType<any>
}

interface MultiListProps {
  name: string
  label: string
  readOnly?: boolean
}

/* Allows user to create following structure:
<field-name>:
  - <field-value-1>,
  - <field-value-2>,
  ...
*/
export const MultiList = ({ name, label, readOnly, formik }: MultiListConnectedProps): JSX.Element => {
  const { getString } = useStrings()
  const [valueMap, setValueMap] = useState<Map<string, string>>(new Map<string, string>([]))
  /*
  <field-name-1>: <field-value-1>,
  <field-name-2>: <field-value-2>,
  ...
  */
  const counter = useRef<number>(0)

  useEffect(() => {
    const values = Array.from(valueMap.values() || []).filter((value: string) => !!value)
    if (values.length > 0) {
      formik?.setFieldValue(name, values)
    } else {
      cleanupField()
    }
  }, [valueMap])

  const cleanupField = useCallback((): void => {
    formik?.setValues(omit({ ...formik?.values }, name))
  }, [formik?.values])

  const getFieldName = useCallback(
    (index: number): string => {
      return `${name}-${index}`
    },
    [name]
  )

  const handleAddRowToList = useCallback((): void => {
    setValueMap((existingValueMap: Map<string, string>) => {
      const rowKeyToAdd = getFieldName(counter.current)
      if (!existingValueMap.has(rowKeyToAdd)) {
        const existingValueMapClone = new Map(existingValueMap)
        existingValueMapClone.set(rowKeyToAdd, '') /* Add key <field-name-1>, <field-name-2>, ... */
        counter.current++ /* this counter always increases, even if a row is removed. This ensures no key collision in the existing value map. */
        return existingValueMapClone
      }
      return existingValueMap
    })
  }, [])

  const handleRemoveRowFromList = useCallback((removedRowKey: string): void => {
    setValueMap((existingValueMap: Map<string, string>) => {
      if (existingValueMap.has(removedRowKey)) {
        const existingValueMapClone = new Map(existingValueMap)
        existingValueMapClone.delete(removedRowKey)
        return existingValueMapClone
      }
      return existingValueMap
    })
    /* remove <field-name-1>, <field-name-2>, ... from formik values, if exist */
    if (removedRowKey && has(formik?.values, removedRowKey)) {
      formik?.setValues(omit({ ...formik?.values }, removedRowKey))
    }
  }, [])

  const handleAddItemToRow = useCallback((rowKey: string, insertedValue: string): void => {
    setValueMap((existingValueMap: Map<string, string>) => {
      if (existingValueMap.has(rowKey)) {
        const existingValueMapClone = new Map(existingValueMap)
        existingValueMapClone.set(rowKey, insertedValue)
        return existingValueMapClone
      }
      return existingValueMap
    })
  }, [])

  const debouncedAddItemToList = useCallback(debounce(handleAddItemToRow, 500), [handleAddItemToRow])

  const renderRow = useCallback((rowKey: string): React.ReactElement => {
    return (
      <Layout.Horizontal margin={{ bottom: 'none' }} flex={{ justifyContent: 'space-between', alignItems: 'center' }}>
        <Container width="90%">
          <FormInput.Text
            name={rowKey}
            disabled={readOnly}
            onChange={event => {
              const value = (event.target as HTMLInputElement).value
              debouncedAddItemToList(rowKey, value)
            }}
          />
        </Container>
        <Icon
          name="code-delete"
          size={25}
          padding={{ bottom: 'medium' }}
          className={css.deleteRowBtn}
          onClick={event => {
            event.preventDefault()
            handleRemoveRowFromList(rowKey)
          }}
        />
      </Layout.Horizontal>
    )
  }, [])

  const renderRows = useCallback((): React.ReactElement => {
    const rows: React.ReactElement[] = []
    valueMap.forEach((_value: string, key: string) => {
      rows.push(renderRow(key))
    })
    return <Layout.Vertical>{rows}</Layout.Vertical>
  }, [valueMap])

  return (
    <Layout.Vertical spacing="small">
      <Layout.Vertical>
        <Text font={{ variation: FontVariation.FORM_LABEL }}>{label}</Text>
        {valueMap.size > 0 && <Container padding={{ top: 'small' }}>{renderRows()}</Container>}
      </Layout.Vertical>
      <Button
        text={getString('addLabel')}
        rightIcon="plus"
        variation={ButtonVariation.PRIMARY}
        size={ButtonSize.SMALL}
        className={css.addBtn}
        onClick={handleAddRowToList}
      />
    </Layout.Vertical>
  )
}

export default connect(MultiList)
