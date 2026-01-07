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
import cx from 'classnames'
import { debounce, get, has, omit, set } from 'lodash-es'
import { FormikContextType, connect } from 'formik'
import { Layout, Text, FormInput, Button, ButtonVariation, ButtonSize, Container } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { Icon } from '@harnessio/icons'
import { useStrings } from 'framework/strings'

import css from './MultiMap.module.scss'

interface MultiMapConnectedProps extends MultiMapProps {
  formik?: FormikContextType<any>
}

interface MultiMapProps {
  /** unique field identifier */
  identifier: string
  /** fully qualified field name */
  name: string
  label: string
  readOnly?: boolean
}

/* Allows user to create following structure:
<field-name>:
  <field-name-1> : <field-value-1>,
  <field-name-2> : <field-value-2>,
  ...
*/

interface KVPair {
  key: string
  value: string
}

const DefaultKVPair: KVPair = {
  key: '',
  value: ''
}

enum KVPairProperty {
  KEY = 'key',
  VALUE = 'value'
}

export const MultiMap = ({ identifier, name, label, readOnly, formik }: MultiMapConnectedProps): JSX.Element => {
  const { getString } = useStrings()
  const [rowValues, setRowValues] = useState<Map<string, KVPair>>(new Map<string, KVPair>([]))
  const [formErrors, setFormErrors] = useState<Map<string, string>>(new Map<string, string>([]))
  /*
  <field-name-1>: {key:  <field-name-1-key>, value:  <field-name-1-value>},
  <field-name-2>: {key:  <field-name-2-key>, value:  <field-name-2-value>},
  ...
  */
  const counter = useRef<number>(0)

  /* When map already has key-value pairs in it */
  useEffect((): void => {
    const existingKVPairs: Record<string, string> = get(formik?.initialValues, name, {})
    const existingKVPairCount = Object.keys(existingKVPairs).length
    if (existingKVPairCount > 0) {
      const formValues = {}
      const initialValueMap = new Map<string, KVPair>([])
      Object.keys(existingKVPairs).forEach((key, index) => {
        const _value = existingKVPairs[key]
        const kvPair = { key, value: _value }
        const rowKeyToAdd = getFieldName(identifier, index)
        initialValueMap.set(rowKeyToAdd, kvPair)
        set(formValues, `${rowKeyToAdd}-key`, key)
        set(formValues, `${rowKeyToAdd}-value`, _value)
      })
      formik?.setValues(formValues)
      setRowValues(initialValueMap)
      counter.current += existingKVPairCount
    }
  }, [get(formik?.initialValues, name)])

  useEffect(() => {
    const values = Array.from(rowValues.values()).filter((value: KVPair) => !!value.key && !!value.value)
    if (values.length > 0) {
      formik?.setFieldValue(name, createKVMap(values))
    }
  }, [rowValues])

  useEffect(() => {
    rowValues.forEach((value: KVPair, rowIdentifier: string) => {
      validateEntry({ rowIdentifier, kvPair: value })
    })
  }, [rowValues])

  /*
  Convert
  [
    {key:  <field-name-1-key>, value:  <field-name-1-value>},
    {key:  <field-name-2-key>, value:  <field-name-2-value>}
  ]
  to
  {
    <field-name-1-key>:  <field-name-1-value>,
    <field-name-2-key>:  <field-name-2-value>
  }
  */
  const createKVMap = useCallback((values: KVPair[]): { [key: string]: string } => {
    const map: { [key: string]: string } = values.reduce(function (_map, obj: KVPair) {
      set(_map, obj.key, obj.value)
      return _map
    }, {})
    return map
  }, [])

  const getFieldName = useCallback((prefix: string, index: number): string => {
    return `${prefix}-${index}`
  }, [])

  const getFormikNameForRowKey = useCallback((rowIdentifier: string): string => {
    return `${rowIdentifier}-key`
  }, [])

  const handleAddRowToList = useCallback((): void => {
    setRowValues((existingValueMap: Map<string, KVPair>) => {
      const rowKeyToAdd = getFieldName(identifier, counter.current)
      if (!existingValueMap.has(rowKeyToAdd)) {
        const existingValueMapClone = new Map(existingValueMap)
        /* Add key with default kv pair
          <field-name-1> : {key: '', value: ''},
          <field-name-2> : {key: '', value: ''},
          ...
        */
        existingValueMapClone.set(rowKeyToAdd, DefaultKVPair)
        counter.current++ /* this counter always increases, even if a row is removed. This ensures no key collision in the existing value map. */
        return existingValueMapClone
      }
      return existingValueMap
    })
  }, [])

  const handleRemoveRowFromList = useCallback((removedRowKey: string): void => {
    setRowValues((existingValueMap: Map<string, KVPair>) => {
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

  const validateEntry = useCallback(({ rowIdentifier, kvPair }: { rowIdentifier: string; kvPair: KVPair }) => {
    setFormErrors((existingFormErrors: Map<string, string>) => {
      const fieldNameKey = getFormikNameForRowKey(rowIdentifier)
      const existingFormErrorsClone = new Map(existingFormErrors)
      if (kvPair.value && !kvPair.key) {
        existingFormErrorsClone.set(fieldNameKey, kvPair.key ? '' : getString('validation.key'))
      } else {
        existingFormErrorsClone.set(fieldNameKey, '')
      }
      return existingFormErrorsClone
    })
  }, [])

  const handleAddItemToRow = useCallback(
    ({
      rowIdentifier,
      insertedValue,
      property
    }: {
      rowIdentifier: string
      insertedValue: string
      property: KVPairProperty
    }): void => {
      setRowValues((existingValueMap: Map<string, KVPair>) => {
        if (existingValueMap.has(rowIdentifier)) {
          const existingValueMapClone = new Map(existingValueMap)
          const existingPair = existingValueMapClone.get(rowIdentifier)
          if (existingPair) {
            if (property === KVPairProperty.KEY) {
              existingValueMapClone.set(rowIdentifier, { key: insertedValue, value: existingPair.value })
            } else if (property === KVPairProperty.VALUE) {
              existingValueMapClone.set(rowIdentifier, { key: existingPair.key, value: insertedValue })
            }
          }
          return existingValueMapClone
        }
        return existingValueMap
      })
    },
    []
  )

  const debouncedAddItemToList = useCallback(debounce(handleAddItemToRow, 500), [handleAddItemToRow])

  const renderRow = useCallback(
    (rowIdentifier: string): React.ReactElement => {
      const rowValidationError = formErrors.get(getFormikNameForRowKey(rowIdentifier))
      return (
        <Layout.Vertical spacing="xsmall" key={rowIdentifier}>
          <Layout.Horizontal
            margin={{ bottom: 'none' }}
            flex={{ justifyContent: 'space-between', alignItems: 'center' }}>
            <Layout.Horizontal width="90%" flex={{ justifyContent: 'flex-start' }} spacing="medium">
              <Container width="50%" className={cx({ [css.rowError]: rowValidationError })}>
                <FormInput.Text
                  name={getFormikNameForRowKey(rowIdentifier)}
                  disabled={readOnly}
                  onChange={event => {
                    const value = (event.target as HTMLInputElement).value
                    debouncedAddItemToList({ rowIdentifier, insertedValue: value, property: KVPairProperty.KEY })
                  }}
                />
              </Container>
              <Container width="50%" className={cx({ [css.rowError]: rowValidationError })}>
                <FormInput.Text
                  name={`${rowIdentifier}-value`}
                  disabled={readOnly}
                  onChange={event => {
                    const value = (event.target as HTMLInputElement).value
                    debouncedAddItemToList({ rowIdentifier, insertedValue: value, property: KVPairProperty.VALUE })
                  }}
                />
              </Container>
            </Layout.Horizontal>
            <Icon
              name="code-delete"
              size={25}
              padding={rowValidationError ? {} : { bottom: 'medium' }}
              className={css.deleteRowBtn}
              onClick={event => {
                event.preventDefault()
                handleRemoveRowFromList(rowIdentifier)
              }}
            />
          </Layout.Horizontal>
          {rowValidationError && (
            <Text font={{ variation: FontVariation.SMALL }} color={Color.RED_500}>
              {rowValidationError}
            </Text>
          )}
        </Layout.Vertical>
      )
    },
    [formErrors]
  )

  const renderMap = useCallback((): React.ReactElement => {
    return (
      <Layout.Vertical width="100%">
        <Layout.Horizontal width="90%" flex={{ justifyContent: 'flex-start' }} spacing="medium">
          <Text width="50%" font={{ variation: FontVariation.SMALL }}>
            {getString('key')}
          </Text>
          <Text width="50%" font={{ variation: FontVariation.SMALL }}>
            {getString('value')}
          </Text>
        </Layout.Horizontal>
        <Container padding={{ top: 'small' }}>{renderRows()}</Container>
      </Layout.Vertical>
    )
  }, [rowValues, formErrors])

  const renderRows = useCallback((): React.ReactElement => {
    const rows: React.ReactElement[] = []
    rowValues.forEach((_value: KVPair, key: string) => {
      rows.push(renderRow(key))
    })
    return <Layout.Vertical>{rows}</Layout.Vertical>
  }, [rowValues, formErrors])

  return (
    <Layout.Vertical spacing="small">
      <Layout.Vertical>
        <Text font={{ variation: FontVariation.FORM_LABEL }}>{label}</Text>
        {rowValues.size > 0 && <Container padding={{ top: 'small' }}>{renderMap()}</Container>}
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

export default connect(MultiMap)
