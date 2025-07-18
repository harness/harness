import React from 'react'
import { Container, Table, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import NoDataState from 'cde-gitness/components/NoDataState'
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
    <Container className={css.locationTable}>
      {regionData && regionData.length > 0 ? (
        <>
          <Table columns={columns} bpTableProps={bpTableProps} className={css.tableContainer} data={regionData} />
          <Text
            icon="plus"
            iconProps={{ size: 10, color: Color.PRIMARY_7 }}
            color={Color.PRIMARY_7}
            onClick={addNewRegion}
            className={css.newRegion}>
            {getString('cde.gitspaceInfraHome.newRegion')}
          </Text>
        </>
      ) : (
        <NoDataState type="region" onButtonClick={addNewRegion} />
      )}
    </Container>
  )
}

export default RegionTable
