import React, { useCallback, useMemo } from 'react'
import cx from 'classnames'
import { Button, ButtonSize, Container, Layout, Pagination } from '@harness/uicore'
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
    [setPage, scrollTop]
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
          // updateQueryParams({ page: page.toString() })
        })
      }
      onNext={
        !!X_NEXT_PAGE &&
        (() => {
          _setPage(page + 1)
          // updateQueryParams({ page: page.toString() })
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
