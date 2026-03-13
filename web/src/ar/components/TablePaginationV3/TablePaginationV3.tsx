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
import cx from 'classnames'
import { Position } from '@blueprintjs/core'
import { Button, ButtonSize, DropDown, Layout, Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'

import css from './TablePaginationV3.module.scss'

const PAGE_SIZE_OPTIONS = [5, 10, 25, 50]

export interface TablePaginationV3Props {
  pageSize: number
  page: number
  hasMore: boolean
  gotoPage: (pageNumber: number) => void
  onPageSizeChange: (size: number) => void
}

const pageSizeToSelectOptions = (options: number[]) => options.map(n => ({ label: n.toString(), value: n.toString() }))

export default function TablePaginationV3(props: TablePaginationV3Props): React.ReactElement {
  const { pageSize, page, hasMore, gotoPage, onPageSizeChange } = props
  const { getString } = useStrings()

  const isPrevDisabled = page === 0
  const isNextDisabled = !hasMore
  const selectOptions = pageSizeToSelectOptions(PAGE_SIZE_OPTIONS)

  return (
    <Layout.Horizontal
      className={css.wrapper}
      padding={{ top: 'medium', bottom: 'medium' }}
      flex={{ justifyContent: 'space-between', alignItems: 'center' }}>
      <Layout.Horizontal flex={{ alignItems: 'baseline' }} spacing="small">
        <DropDown
          items={selectOptions}
          onChange={item => onPageSizeChange(Number(item.value))}
          value={pageSize.toString()}
          filterable={false}
          width={60}
          className={css.pageSizeDropdown}
          popoverProps={{ position: Position.RIGHT_BOTTOM }}
        />
        <Text className={css.itemsPerPageText}>{getString('versionDetails.artifactFiles.itemsPerPage')}</Text>
        <div className={css.separator} aria-hidden />
        <Text className={css.pageText}>{getString('versionDetails.artifactFiles.pageLabel', { page: page + 1 })}</Text>
      </Layout.Horizontal>
      <Layout.Horizontal className={css.prevNextButtonGroup}>
        <Button
          icon="chevron-left"
          size={ButtonSize.SMALL}
          iconProps={{ size: 14 }}
          onClick={() => gotoPage(page - 1)}
          disabled={isPrevDisabled}
          className={cx(css.roundedButton, css.prevButton)}
          minimal
        />
        <Button
          icon="chevron-right"
          size={ButtonSize.SMALL}
          iconProps={{ size: 14 }}
          onClick={() => gotoPage(page + 1)}
          disabled={isNextDisabled}
          className={cx(css.roundedButton, css.nextButton)}
          minimal
        />
      </Layout.Horizontal>
    </Layout.Horizontal>
  )
}
