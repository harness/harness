import React, { useCallback } from 'react'
import { Container, TableV2 } from '@harnessio/uicore'
import type { Column, Row } from 'react-table'
import { useStrings } from 'framework/strings'
import type { AwsRegionConfig, ZoneConfig } from 'cde-gitness/constants'

import RegionTableV2 from './RegionTableV2'
import css from './RegionTable.module.scss'

interface ExtendedAwsRegionConfig extends AwsRegionConfig {
  zones?: ZoneConfig[]
  identifier: number
}

interface RegionTableProps {
  columns: any
  regionData: ExtendedAwsRegionConfig[]
  addNewRegion: () => void
  disableAddButton?: boolean
}

function RegionTable({ columns, regionData, addNewRegion, disableAddButton }: RegionTableProps) {
  const { getString } = useStrings()

  const zoneColumns: Column<ZoneConfig>[] = [
    {
      Header: getString('cde.Aws.availabilityZone'),
      accessor: 'zone',
      width: '30%'
    },
    {
      Header: getString('cde.Aws.privateSubnet'),
      accessor: 'privateSubnet',
      width: '30%'
    },
    {
      Header: getString('cde.Aws.publicSubnet'),
      accessor: 'publicSubnet',
      width: '30%'
    }
  ] as Column<ZoneConfig>[]

  const renderRowSubComponent = useCallback(
    ({ row }: { row: Row<ExtendedAwsRegionConfig> }) => {
      const region = row.original
      const hasZones = region.zones && region.zones.length > 0

      if (!hasZones) return null

      return (
        <Container
          className={css.tableRowSubComponent}
          onClick={e => e.stopPropagation()}
          margin={{ top: '10px', bottom: '10px' }}>
          <TableV2<ZoneConfig> columns={zoneColumns} data={region.zones || []} className={css.zonesTable} minimal />
        </Container>
      )
    },
    [zoneColumns]
  )

  return (
    <RegionTableV2
      columns={columns}
      regionData={regionData}
      addNewRegion={addNewRegion}
      getString={getString}
      showNoDataState={true}
      autoExpandNewRegions={true}
      tableClassName={css.table}
      containerClassName={css.locationTable}
      tableRowClassName={css.tableRow}
      newRegionClassName={css.newRegion}
      newRegionLabel={getString('cde.gitspaceInfraHome.newRegion')}
      renderRowSubComponent={renderRowSubComponent}
      disableAddButton={disableAddButton}
    />
  )
}

export default RegionTable
