import React from 'react'
import { Container, Table, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import type { regionProp } from 'cde-gitness/constants'
import css from './RegionTable.module.scss'

interface RegionTableProps {
  columns: any
  regionData: regionProp[]
  addNewRegion: () => void
}

function RegionTable({ columns, regionData, addNewRegion }: RegionTableProps) {
  const { getString } = useStrings()
  const bpTableProps = { bordered: false, condensed: true, striped: true }
  return (
    <Container>
      <Container className={css.locationTable}>
        <Table columns={columns} bpTableProps={bpTableProps} className={css.tableContainer} data={regionData} />
      </Container>
      <Text
        icon="plus"
        iconProps={{ size: 10, color: Color.PRIMARY_7 }}
        color={Color.PRIMARY_7}
        onClick={addNewRegion}
        className={css.newRegion}>
        {getString('cde.gitspaceInfraHome.newRegion')}
      </Text>
    </Container>
  )
}

export default RegionTable
