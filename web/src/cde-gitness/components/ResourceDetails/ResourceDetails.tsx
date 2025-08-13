import React from 'react'
import { Layout, Text, Container } from '@harnessio/uicore'
import { Cpu } from 'iconoir-react'
import RegionIcon from 'cde-gitness/assests/globe.svg?url'
import { useStrings } from 'framework/strings'
import type { TypesInfraProviderResource } from 'services/cde'
import css from './ResourceDetails.module.scss'

const ResourceDetails = ({ resource }: { resource?: TypesInfraProviderResource }) => {
  const { getString } = useStrings()
  const { region, name } = resource || {}
  return (
    <Layout.Horizontal spacing="small" className={css.container}>
      <Layout.Horizontal spacing={'xsmall'} flex={{ alignItems: 'center' }} className={css.textContainer}>
        <img height={12} width={12} src={RegionIcon} className={css.iconContainer} />
        <Text font={{ size: 'small' }} lineClamp={1} title={region?.toUpperCase() || getString('cde.na')}>
          {region?.toUpperCase() || getString('cde.na')}
        </Text>
      </Layout.Horizontal>
      <Container width={'3px'} />
      <Layout.Horizontal spacing={'xsmall'} flex={{ alignItems: 'center' }} className={css.textContainer}>
        <Cpu height={12} width={12} className={css.iconContainer} />
        <Text font={{ size: 'small' }} title={name || getString('cde.na')} lineClamp={1} className={css.truncatedText}>
          {name || getString('cde.na')}
        </Text>
      </Layout.Horizontal>
    </Layout.Horizontal>
  )
}

export default ResourceDetails
