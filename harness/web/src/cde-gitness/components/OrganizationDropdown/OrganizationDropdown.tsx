import React, { useCallback, useState, useEffect } from 'react'
import {
  Container,
  ExpandingSearchInput,
  Layout,
  Text,
  SortDropdown,
  sortByCreated,
  sortByLastModified,
  sortByName,
  SortMethod,
  Popover,
  Button,
  ButtonVariation
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { Position } from '@blueprintjs/core'
import cx from 'classnames'
import { useStrings } from 'framework/strings'
import { useOrganizations } from 'cde-gitness/hooks/useOrganizations'
import { useLocalStorage } from 'hooks/useLocalStorage'
import css from './OrganizationDropdown.module.scss'

export interface OrganizationDropdownProps {
  value: string[]
  onChange: (value: string[]) => void
  disabled?: boolean
  pageSize?: number
}

interface OrganizationItemProps {
  org: { identifier: string; name: string }
  isSelected: boolean
  onToggle: () => void
}

const OrganizationItem = ({ org, isSelected, onToggle }: OrganizationItemProps): React.ReactElement => {
  return (
    <Container onClick={onToggle} className={cx(css.organizationContainer, { [css.selected]: isSelected })}>
      <Layout.Horizontal spacing="small" className={css.orgItemHorizontal}>
        <Container
          className={css.checkbox}
          border={{ color: isSelected ? Color.PRIMARY_7 : Color.GREY_300 }}
          background={isSelected ? Color.PRIMARY_7 : Color.WHITE}>
          {isSelected && <Icon name="tick" color={Color.WHITE} size={12} />}
        </Container>
        <Container
          className={cx(css.orgIcon, { [css.orgIconSelected]: isSelected, [css.orgIconUnselected]: !isSelected })}>
          <Icon name="nav-organization" color={isSelected ? Color.WHITE : Color.PURPLE_700} size={16} />
        </Container>
        <Text font={{ variation: FontVariation.BODY2 }} color={Color.GREY_900}>
          {org.name || org.identifier}
        </Text>
      </Layout.Horizontal>
    </Container>
  )
}

export default function OrganizationDropdown(props: OrganizationDropdownProps): JSX.Element {
  const { value = [], onChange, disabled } = props
  const { getString } = useStrings()
  const [searchQuery, setSearchQuery] = useState<string>('')
  const [isOpen, setIsOpen] = useState(false)
  const [allLoadedOrganizations, setAllLoadedOrganizations] = useState<any[]>([])
  const MAX_CACHED_ORGANIZATIONS = 1200

  const [sortPreference, setSortPreference] = useLocalStorage<SortMethod>(
    'sort-orgselector',
    SortMethod.LastModifiedDesc
  )

  const { organizations, loading, hasMore, loadMore, search, refetch } = useOrganizations({
    searchTerm: searchQuery,
    sortOrder: sortPreference
  })

  useEffect(() => {
    if (organizations.length > 0) {
      setAllLoadedOrganizations(prevAll => {
        const existingIds = new Set(prevAll.map(org => org.identifier))
        const newOrganizations = organizations.filter(org => !existingIds.has(org.identifier))
        const combinedOrgs = [...prevAll, ...newOrganizations]

        if (combinedOrgs.length > MAX_CACHED_ORGANIZATIONS) {
          return combinedOrgs.slice(combinedOrgs.length - MAX_CACHED_ORGANIZATIONS)
        }

        return combinedOrgs
      })
    }
  }, [organizations])

  const handleSearch = useCallback(
    (query: string) => {
      setSearchQuery(query)
      search(query)
    },
    [search]
  )

  useEffect(() => {
    if (value.length && allLoadedOrganizations.length > 0) {
      const validOrgIds = allLoadedOrganizations.map(org => org.identifier)
      const validValues = value.filter(id => validOrgIds.includes(id))

      if (validValues.length !== value.length) {
        const hasSearched = searchQuery.length > 0
        if (hasSearched || allLoadedOrganizations.length >= 200) {
          onChange(validValues)
        }
      }
    }
  }, [allLoadedOrganizations, value, onChange, searchQuery])

  const safeOrganizations = Array.isArray(organizations) ? organizations : []

  const buttonText = getString('cde.usageDashboard.organizations')
  const selectedCount = value.length

  const dropdownContent = (
    <Container className={css.dropdownContent}>
      <Layout.Horizontal spacing={'small'} className={css.searchContainer}>
        <ExpandingSearchInput
          onChange={handleSearch}
          className={css.searchWidth}
          alwaysExpanded
          autoFocus={false}
          placeholder={getString('cde.usageDashboard.searchOrgPlaceholder')}
        />
        <Container className={css.sortWidth}>
          <SortDropdown
            selectedSortMethod={sortPreference}
            onSortMethodChange={option => {
              if (option) {
                setSortPreference(option.value as SortMethod)
              }
            }}
            sortOptions={[...sortByLastModified, ...sortByCreated, ...sortByName]}
          />
        </Container>
      </Layout.Horizontal>

      {loading ? (
        <Container className={css.loadingContainer}>
          <Icon name="loading" size={24} color={Color.PRIMARY_7} />
        </Container>
      ) : null}

      {!loading && safeOrganizations.length === 0 ? (
        <Container height={120} className={css.emptyContainer}>
          <Text font={{ variation: FontVariation.H5 }} className={css.noResultsTitle}>
            {getString('cde.usageDashboard.noResultsFound')}
          </Text>
          <Text color={Color.GREY_500}>{getString('cde.usageDashboard.tryDifferentKeywords')}</Text>
        </Container>
      ) : !loading ? (
        <Container className={css.organizationsContainer}>
          {value.length > 0 && (
            <Container
              onClick={() => {
                onChange([])
                refetch()
              }}
              className={css.actionContainer}>
              <Layout.Horizontal spacing="xsmall" flex={{ alignItems: 'center' }}>
                <Icon name="main-trash" size={16} color={Color.RED_500} />
                <Text font={{ variation: FontVariation.BODY2 }} color={Color.RED_500}>
                  {getString('cde.usageDashboard.clearAll')}
                </Text>
              </Layout.Horizontal>
            </Container>
          )}

          {safeOrganizations.map(org => {
            const isSelected = value.includes(org.identifier)
            return (
              <OrganizationItem
                key={org.identifier}
                org={org}
                isSelected={isSelected}
                onToggle={() => {
                  if (isSelected) {
                    onChange(value.filter(id => id !== org.identifier))
                  } else {
                    onChange([...value, org.identifier])
                  }
                }}
              />
            )
          })}

          {hasMore && (
            <Container onClick={() => loadMore()} className={cx(css.actionContainer)}>
              <Layout.Horizontal spacing="xsmall" flex={{ alignItems: 'center' }}>
                <Icon name={loading ? 'loading' : 'chevron-down'} size={16} color={Color.PRIMARY_7} />
                <Text font={{ variation: FontVariation.BODY2 }} color={Color.PRIMARY_7}>
                  {loading ? getString('cde.usageDashboard.loading') : getString('cde.usageDashboard.loadMore')}
                </Text>
              </Layout.Horizontal>
            </Container>
          )}
        </Container>
      ) : null}
    </Container>
  )

  return (
    <Container className={css.customPopover}>
      <Popover
        isOpen={isOpen}
        onInteraction={(nextOpenState, _event) => {
          setIsOpen(nextOpenState)
        }}
        position={Position.BOTTOM_LEFT}
        className={css.popover}
        popoverClassName={css.popoverContent}
        content={dropdownContent}>
        <Button
          variation={ButtonVariation.TERTIARY}
          rightIcon="main-chevron-down"
          disabled={disabled}
          className={css.dropdownButton}
          data-testid="organization-select"
          onClick={() => setIsOpen(!isOpen)}
          iconProps={{
            size: 8,
            color: 'var(--grey-400)'
          }}>
          <Layout.Horizontal spacing="small" className={css.buttonContent}>
            <Text className={css.buttonText}>{buttonText}</Text>
            {selectedCount > 0 && (
              <Container className={css.countBadge}>
                <Text className={css.countText}>{selectedCount < 10 ? `0${selectedCount}` : selectedCount}</Text>
              </Container>
            )}
          </Layout.Horizontal>
        </Button>
      </Popover>
    </Container>
  )
}
