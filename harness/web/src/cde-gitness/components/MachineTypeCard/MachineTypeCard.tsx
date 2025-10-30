import React from 'react'
import { Card, Text, Layout, Checkbox, Container } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import cx from 'classnames'
import type { MachineType } from 'cde-gitness/utils/cloudRegionsUtils'
import css from './MachineTypeCard.module.scss'

interface MachineTypeCardProps {
  machineType: MachineType
  checked: boolean
  onChange: (checked: boolean) => void
  className?: string
}

const MachineTypeCard: React.FC<MachineTypeCardProps> = ({ machineType, checked, onChange, className }) => {
  const persistentDiskType = machineType.metadata?.persistent_disk_type
    ? ` (${machineType.metadata?.persistent_disk_type})`
    : ''
  return (
    <Card className={cx(css.machineTypeCard, className)}>
      <Layout.Horizontal className={css.machineTypeItem}>
        <Checkbox
          checked={checked}
          onChange={e => onChange(e.currentTarget.checked)}
          className={css.machineTypeCheckbox}
        />
        <Container className={css.machineTypeInfo}>
          <Text font={{ variation: FontVariation.BODY2 }} color={Color.GREY_800}>
            {machineType.name}
          </Text>
          <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_500}>
            {machineType.cpu && `Up to ${machineType.cpu} cores, `}
            {machineType.memory && `${machineType.memory}GB RAM, `}
            {machineType.disk && `${machineType.disk}GB Storage` + persistentDiskType}
          </Text>
        </Container>
      </Layout.Horizontal>
    </Card>
  )
}

export default MachineTypeCard
