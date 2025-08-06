import React from 'react'
import { Layout, Text } from '@harnessio/uicore'
import { Cpu } from 'iconoir-react'
import RegionIcon from 'cde-gitness/assests/globe.svg?url'
import { useStrings } from 'framework/strings'
import type { TypesInfraProviderResource } from 'services/cde'
import css from './ResourceDetails.module.scss'

const ResourceDetails = ({ resource }: { resource?: TypesInfraProviderResource }) => {
  const { getString } = useStrings()
  const { region, name } = resource || {}
  return (
    <Layout.Vertical spacing="small">
      <span className={css.iconTextStyle}>
        <img height={12} width={12} src={RegionIcon} />
        <Text font={{ size: 'small' }} lineClamp={1}>
          {region?.toUpperCase() || getString('cde.na')}
        </Text>
      </span>
      <span className={css.iconTextStyle}>
        <Cpu height={12} width={12} />
        <Text font={{ size: 'small' }} lineClamp={1}>
          {name || getString('cde.na')}
        </Text>
      </span>
    </Layout.Vertical>
  )
}

export default ResourceDetails
