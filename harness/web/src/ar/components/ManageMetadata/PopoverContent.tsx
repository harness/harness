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

import React from 'react'
import { useState } from 'react'
import { Container, Layout, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import type { TypesLabel, TypesLabelValue } from '@harnessio/react-har-service-client'

import { ColorName, LabelType } from 'utils/Utils'
import { Label, LabelTitle } from 'components/Label/Label'

import { useStrings } from '@ar/frameworks/strings'

import DropdownList from './DropdownList'

import css from './ManageMetadata.module.scss'

interface PopoverContentProps {
  getLabels: (searchTerm: string) => Promise<Array<TypesLabel>>
  getValues: (label: TypesLabel, searchTerm: string) => Promise<Array<TypesLabelValue>>
  onSelect: (label: TypesLabel, value?: TypesLabelValue) => void
}

function PopoverContent({ getLabels, getValues, onSelect }: PopoverContentProps) {
  const [selectedLabel, setSelectedLabel] = useState<TypesLabel | null>(null)
  const { getString } = useStrings()

  const renderLabel = (option: TypesLabel) => {
    return (
      <LabelTitle
        name={option.key as string}
        value_count={option.value_count}
        label_color={option.color as ColorName}
        scope={option.scope}
      />
    )
  }

  const renderLabelValue = (label: string, option: TypesLabelValue) => {
    return (
      <Label
        name={label}
        label_value={{
          name: option.value,
          color: option.color as ColorName
        }}
      />
    )
  }

  const renderNewItem = (item: string) => {
    if (!selectedLabel) return null
    return (
      <Layout.Horizontal spacing="small">
        <Text font={{ variation: FontVariation.BODY, weight: 'bold' }} color={Color.PRIMARY_7}>
          {getString('labels.addNewValue')}
        </Text>
        {renderLabelValue(selectedLabel.key as string, { value: item, color: selectedLabel.color })}
      </Layout.Horizontal>
    )
  }

  return (
    <Container className={css.popoverContainer}>
      {selectedLabel ? (
        <DropdownList<TypesLabelValue>
          key="values"
          getItems={q => getValues(selectedLabel, q)}
          getRowId={each => `${selectedLabel.key}-${each.value}`}
          getRowValue={each => each.value as string}
          renderItem={item => renderLabelValue(selectedLabel.key as string, item)}
          onSelect={item => {
            onSelect(selectedLabel, item)
          }}
          onCreateNewOption={item => {
            onSelect(selectedLabel, {
              value: item,
              label_id: selectedLabel.id,
              color: selectedLabel.color
            })
          }}
          renderNewItem={renderNewItem}
          shouldAllowCreateNewOption={selectedLabel?.type === LabelType.DYNAMIC}
          placeholder={
            selectedLabel?.type === LabelType.DYNAMIC
              ? getString('labels.findOrAddValue')
              : getString('labels.findAValue')
          }
          leftElement={
            <Label
              name={selectedLabel.key as string}
              label_color={selectedLabel.color as ColorName}
              scope={selectedLabel.scope}
            />
          }
        />
      ) : (
        <DropdownList<TypesLabel>
          key="labels"
          getItems={q => getLabels(q)}
          getRowId={each => each.key as string}
          getRowValue={each => each.key as string}
          renderItem={renderLabel}
          shouldAllowCreateNewOption={false}
          placeholder={getString('labels.findALabel')}
          onSelect={item => {
            if (item.value_count && item.value_count > 0) {
              setSelectedLabel(item)
            } else {
              onSelect(item)
            }
          }}
        />
      )}
    </Container>
  )
}

export default PopoverContent
