import React from 'react'
import { Container, Layout, Text } from '@harnessio/uicore'
import { Cpu } from 'iconoir-react'
import RegionIcon from 'cde-gitness/assests/globe.svg?url'
import { useStrings } from 'framework/strings'
import type { TypesInfraProviderResource } from 'services/cde'

const ResourceDetails = ({ resource }: { resource?: TypesInfraProviderResource }) => {
  const { getString } = useStrings()
  const { region, name } = resource || {}
  return (
    <Layout.Horizontal spacing="small">
      <Layout.Horizontal spacing={'xsmall'} flex={{ alignItems: 'center' }}>
        <img height={12} width={12} src={RegionIcon} />{' '}
        <Text font={{ size: 'small' }}>{region?.toUpperCase() || getString('cde.na')}</Text>
      </Layout.Horizontal>
      <Container width={'5px'} />
      <Layout.Horizontal spacing={'xsmall'} flex={{ alignItems: 'center' }}>
        <Cpu height={12} width={12} /> <Text font={{ size: 'small' }}>{name || getString('cde.na')}</Text>
      </Layout.Horizontal>
    </Layout.Horizontal>
  )
}

export default ResourceDetails
