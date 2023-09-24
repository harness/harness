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

import React, { useCallback, useMemo } from 'react'
import cx from 'classnames'
import { Button, ButtonSize, Container, Layout, Pagination } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import css from './ResourceListingPagination.module.scss'

interface ResourceListingPaginationProps {
  response: Response | null
  page: number
  setPage: React.Dispatch<React.SetStateAction<number>>
  scrollTop?: boolean
}

// There are two type of pagination results returned from Code API.
// One returns information that works with UICore Pagination component in which we know total pages, total items, etc... The other
// has only information to render Prev, Next.
//
// This component consolidates both cases to remove same pagination logic in pages and components.
export const ResourceListingPagination: React.FC<ResourceListingPaginationProps> = ({
  response,
  page,
  setPage,
  scrollTop = true
}) => {
  const { updateQueryParams } = useUpdateQueryParams()
  const { X_NEXT_PAGE, X_PREV_PAGE, totalItems, totalPages, pageSize } = useParsePaginationInfo(response)
  const _setPage = useCallback(
    (_page: number) => {
      if (scrollTop) {
        setTimeout(() => {
          window.scrollTo({
            top: 0,
            left: 0,
            behavior: 'smooth'
          })
        }, 0)
      }
      setPage(_page)
      updateQueryParams({ page: _page.toString() })
    },
    [setPage, scrollTop, response] // eslint-disable-line react-hooks/exhaustive-deps
  )

  return totalItems ? (
    page === 1 && totalItems < pageSize ? null : (
      <Container margin={{ left: 'medium', right: 'medium' }}>
        <Pagination
          className={css.pagination}
          hidePageNumbers
          gotoPage={index => _setPage(index + 1)}
          itemCount={totalItems}
          pageCount={totalPages}
          pageIndex={page - 1}
          pageSize={pageSize}
        />
      </Container>
    )
  ) : page === 1 && !X_PREV_PAGE && !X_NEXT_PAGE ? null : (
    <PrevNextPagination
      onPrev={
        !!X_PREV_PAGE &&
        (() => {
          _setPage(page - 1)
        })
      }
      onNext={
        !!X_NEXT_PAGE &&
        (() => {
          _setPage(page + 1)
        })
      }
    />
  )
}

function useParsePaginationInfo(response: Nullable<Response>) {
  const totalItems = useMemo(() => parseInt(response?.headers?.get('x-total') || '0'), [response])
  const totalPages = useMemo(() => parseInt(response?.headers?.get('x-total-pages') || '0'), [response])
  const pageSize = useMemo(() => parseInt(response?.headers?.get('x-per-page') || '0'), [response])
  const X_NEXT_PAGE = useMemo(() => parseInt(response?.headers?.get('x-next-page') || '0'), [response])
  const X_PREV_PAGE = useMemo(() => parseInt(response?.headers?.get('x-prev-page') || '0'), [response])

  return { totalItems, totalPages, pageSize, X_NEXT_PAGE, X_PREV_PAGE }
}

interface PrevNextPaginationProps {
  onPrev?: false | (() => void)
  onNext?: false | (() => void)
  skipLayout?: boolean
}

function PrevNextPagination({ onPrev, onNext, skipLayout }: PrevNextPaginationProps) {
  const { getString } = useStrings()

  return (
    <Container className={skipLayout ? undefined : css.main}>
      <Layout.Horizontal>
        <Button
          text={getString('prev')}
          icon="arrow-left"
          size={ButtonSize.SMALL}
          className={cx(css.roundedButton, css.buttonLeft)}
          iconProps={{ size: 12 }}
          onClick={onPrev ? onPrev : undefined}
          disabled={!onPrev}
        />
        <Button
          text={getString('next')}
          rightIcon="arrow-right"
          size={ButtonSize.SMALL}
          className={cx(css.roundedButton, css.buttonRight)}
          iconProps={{ size: 12 }}
          onClick={onNext ? onNext : undefined}
          disabled={!onNext}
        />
      </Layout.Horizontal>
    </Container>
  )
}
