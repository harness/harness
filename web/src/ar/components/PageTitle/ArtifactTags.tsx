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

import React, { useState } from 'react'
import classNames from 'classnames'
import { Button, ButtonVariation, Layout, useToggleOpen } from '@harnessio/uicore'

import { useParentComponents } from '@ar/hooks'
import TagIcon from '@ar/components/MultiTagsInput/TagIcon'
import type { RbacButtonProps } from '@ar/__mocks__/components/RbacButton'
import MultiTagsInput from '@ar/components/MultiTagsInput/MultiTagsInput'

import css from './PageTitle.module.scss'

interface ArtifactTagsProps {
  labels: string[]
  placeholder?: string
  onChange: (items: string[]) => Promise<boolean>
  permission?: RbacButtonProps['permission']
}

const EMPTY_TAG_VALUE = '+ Labels'

export default function ArtifactTags(props: ArtifactTagsProps): JSX.Element | null {
  const { labels, onChange, placeholder, permission } = props
  const [selectedItems, setSelectedItems] = useState(labels)
  const [query, setQuery] = useState('')
  const { RbacButton } = useParentComponents()
  const { isOpen: isEdit, open, close } = useToggleOpen(false)

  const handleOnSubmit = async () => {
    try {
      const isSuccess = await onChange(selectedItems as string[])
      if (isSuccess) {
        close()
        setQuery('')
      }
    } catch {
      setQuery('')
    }
  }

  return (
    <Layout.Horizontal
      className={classNames(css.artifactTagContainer, { [css.editMode]: isEdit })}
      spacing="small"
      flex={{ alignItems: 'flex-start' }}>
      {!isEdit && <TagIcon margin={{ top: 'small' }} />}
      <MultiTagsInput
        fill
        noInputBorder
        allowNewTag
        hidePopover
        query={query}
        onQueryChange={setQuery}
        readonly={!isEdit}
        items={[]}
        placeholder={placeholder}
        selectedItems={!isEdit && !selectedItems.length ? [EMPTY_TAG_VALUE] : selectedItems}
        getTagProps={value => ({
          interactive: true,
          onClick: () => {
            if (value === EMPTY_TAG_VALUE && !isEdit && !selectedItems.length) {
              open()
            }
          }
        })}
        className={classNames(css.tagWrapper, {
          [css.editMode]: isEdit,
          [css.readonly]: !isEdit
        })}
        onChange={val => {
          setSelectedItems(val as string[])
        }}
      />
      {!isEdit && !!selectedItems.length && (
        <RbacButton
          className={css.iconBtn}
          minimal
          iconProps={{ size: 20 }}
          variation={ButtonVariation.ICON}
          icon="code-edit"
          onClick={() => open()}
          permission={permission}
        />
      )}
      {isEdit && (
        <>
          <Button
            className={css.iconBtn}
            minimal
            small
            variation={ButtonVariation.ICON}
            icon="small-cross"
            iconProps={{ size: 20 }}
            onClick={() => {
              setSelectedItems(labels)
              close()
              setQuery('')
            }}
          />
          <RbacButton
            className={css.iconBtn}
            minimal
            variation={ButtonVariation.ICON}
            iconProps={{ size: 20 }}
            icon="small-tick"
            onClick={handleOnSubmit}
            permission={permission}
          />
        </>
      )}
    </Layout.Horizontal>
  )
}
