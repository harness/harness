import React, { useCallback, useState, useEffect } from 'react'
import { Container, Text, TableV2 } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { Icon } from '@harnessio/icons'
import type { CellProps, Column, Row } from 'react-table'
import classNames from 'classnames'
import type { AwsRegionConfig } from 'cde-gitness/constants'

import NoDataState from 'cde-gitness/components/NoDataState'
import { handleToggleExpandableRow } from './utils'

// Extended AwsRegionConfig to include zones array needed for renderRowSubComponent
interface ExtendedAwsRegionConfig extends AwsRegionConfig {
  zones?: any[]
  identifier: number
}

interface GenericRegionTableProps {
  // Required props
  columns: Column<ExtendedAwsRegionConfig>[]
  regionData: ExtendedAwsRegionConfig[]
  addNewRegion: () => void
  getString: (key: any, vars?: Record<string, any>) => string

  // Configuration props
  showNoDataState?: boolean
  autoExpandNewRegions?: boolean
  disableAddButton?: boolean

  // Style props
  tableClassName?: string
  containerClassName?: string
  tableRowClassName?: string
  newRegionClassName?: string

  // Labels
  newRegionLabel?: string

  // Custom renderers
  customToggleRenderer?: (props: {
    region: ExtendedAwsRegionConfig
    isExpanded: boolean
    onToggle: (region: ExtendedAwsRegionConfig) => void
  }) => React.ReactNode

  customNoDataRenderer?: () => React.ReactNode
  renderRowSubComponent?: ({ row }: { row: Row<ExtendedAwsRegionConfig> }) => React.ReactNode
}

function RegionTableV2({
  columns,
  regionData,
  addNewRegion,
  getString,
  showNoDataState = true,
  autoExpandNewRegions = false,
  tableClassName,
  containerClassName,
  tableRowClassName,
  newRegionClassName,
  newRegionLabel,
  customToggleRenderer,
  customNoDataRenderer,
  renderRowSubComponent,
  disableAddButton
}: GenericRegionTableProps) {
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set())
  const [previousRegionCount, setPreviousRegionCount] = useState(regionData.length)

  // Auto-expand newly added regions (if autoExpandNewRegions is enabled)
  useEffect(() => {
    if (autoExpandNewRegions && regionData.length > previousRegionCount) {
      const newRegion = regionData[regionData.length - 1]
      const regionId = `region-${newRegion.region_name}`
      setExpandedRows(new Set([regionId]))
    }
    setPreviousRegionCount(regionData.length)
  }, [regionData, previousRegionCount, autoExpandNewRegions])

  const getRowId = useCallback((rowData: ExtendedAwsRegionConfig) => {
    return `region-${rowData.region_name}`
  }, [])

  const onToggleRow = useCallback(
    (rowData: ExtendedAwsRegionConfig): void => {
      if (!renderRowSubComponent) return

      const value = getRowId(rowData)
      setExpandedRows(handleToggleExpandableRow(value))
    },
    [getRowId, renderRowSubComponent]
  )

  const RenderToggle = ({ row }: CellProps<ExtendedAwsRegionConfig>) => {
    const region = row.original
    const isExpanded = expandedRows.has(getRowId(region))

    if (customToggleRenderer) {
      return customToggleRenderer({
        region,
        isExpanded,
        onToggle: onToggleRow
      }) as React.ReactElement
    }

    // Default toggle renderer - only show if renderRowSubComponent is provided
    if (!renderRowSubComponent) {
      return <Container style={{ width: '20px' }} />
    }

    return (
      <Container
        style={{ cursor: 'pointer', width: '20px' }}
        onClick={e => {
          e.stopPropagation()
          onToggleRow(region)
        }}>
        <Icon name={isExpanded ? 'chevron-down' : 'chevron-right'} size={16} />
      </Container>
    )
  }

  const processedColumns = columns.map((col: any) => {
    if (col.accessor === 'toggle') {
      return {
        ...col,
        Cell: RenderToggle,
        expandedRows,
        setExpandedRows,
        getRowId
      }
    }
    return {
      ...col,
      Cell: col.Cell || undefined
    }
  })

  const renderNoData = () => {
    if (customNoDataRenderer) {
      return customNoDataRenderer()
    }
    if (showNoDataState) {
      return <NoDataState type="region" onButtonClick={addNewRegion} />
    }
    return null
  }

  const renderTable = () => {
    if (regionData.length === 0) {
      return renderNoData()
    }

    return (
      <TableV2<ExtendedAwsRegionConfig>
        className={classNames(tableClassName)}
        columns={processedColumns}
        data={regionData.map(region => ({
          ...region,
          subRows: [],
          expanded: expandedRows.has(getRowId(region))
        }))}
        renderRowSubComponent={renderRowSubComponent}
        getRowClassName={_row => classNames(tableRowClassName)}
        onRowClick={renderRowSubComponent ? onToggleRow : undefined}
        autoResetExpanded={false}
        minimal
      />
    )
  }

  const renderAddNewRegionButton = () => {
    if (disableAddButton) {
      return (
        <Text color={Color.GREY_500} className={newRegionClassName}>
          {getString('cde.Aws.singleRegionRestriction')}
        </Text>
      )
    }

    return (
      <Text
        icon="plus"
        iconProps={{ size: 10, color: Color.PRIMARY_7 }}
        color={Color.PRIMARY_7}
        onClick={addNewRegion}
        className={newRegionClassName}>
        {newRegionLabel || getString('cde.gitspaceInfraHome.newRegion')}
      </Text>
    )
  }

  return (
    <Container>
      <Container className={containerClassName}>{renderTable()}</Container>
      {regionData.length > 0 && renderAddNewRegionButton()}
    </Container>
  )
}

export default RegionTableV2
