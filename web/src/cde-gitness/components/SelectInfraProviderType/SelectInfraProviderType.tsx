import { Container, Layout, Text } from '@harnessio/uicore'
import React from 'react'
import { Cloud } from 'iconoir-react'
import { Menu, MenuItem } from '@blueprintjs/core'
import { useFormikContext } from 'formik'
import { useStrings } from 'framework/strings'
import type { OpenapiCreateGitspaceRequest, TypesInfraProviderConfig } from 'services/cde'
import { HARNESS_GCP, HYBRID_VM_GCP, HYBRID_VM_AWS } from 'cde-gitness/constants'
import googleCloudIcon from 'icons/google-cloud.svg?url'
import awsIcon from 'cde-gitness/assests/aws.svg?url'
import HarnessIcon from 'icons/Harness.svg?url'
import type { dropdownProps } from 'cde-gitness/constants'
import { CDECustomDropdown } from '../CDECustomDropdown/CDECustomDropdown'
import css from './SelectInfraProviderType.module.scss'

// Function to get provider icon based on provider type
const getProviderIcon = (providerType: string) => {
  if (providerType === HYBRID_VM_GCP) {
    return googleCloudIcon
  } else if (providerType === HARNESS_GCP) {
    return HarnessIcon
  } else if (providerType === HYBRID_VM_AWS) {
    return awsIcon
  }
  return null
}

interface SelectInfraProviderTypeProps {
  infraProviders: dropdownProps[]
  allProviders?: TypesInfraProviderConfig[]
}

const SelectInfraProviderType = ({ infraProviders, allProviders = [] }: SelectInfraProviderTypeProps) => {
  const { getString } = useStrings()

  const { values, setFieldValue: onChange } = useFormikContext<OpenapiCreateGitspaceRequest>()

  const selectedInfraProvider = infraProviders?.find(
    (item: dropdownProps) => item?.value === values?.metadata?.infraProvider
  )

  const getProviderType = (identifier: string): string => {
    const provider = allProviders?.find(config => config.identifier === identifier)
    return provider?.type || ''
  }

  const selectedProviderType = getProviderType(values?.metadata?.infraProvider || '')

  const selectedProviderIcon = getProviderIcon(selectedProviderType)

  return (
    <Container>
      <CDECustomDropdown
        label={
          <Layout.Horizontal
            spacing={'small'}
            className={css.dropdownLabel}
            flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
            {selectedProviderIcon && (
              <img
                src={selectedProviderIcon}
                height={16}
                width={16}
                style={{ marginRight: '9px', display: 'flex', alignSelf: 'center' }}
              />
            )}
            {!selectedProviderIcon && (
              <Cloud height={16} width={16} style={{ marginRight: '9px', display: 'flex', alignSelf: 'center' }} />
            )}
            <Layout.Vertical>
              <Text font={'normal'}>{selectedInfraProvider?.label || getString('cde.infraProvider')}</Text>
            </Layout.Vertical>
          </Layout.Horizontal>
        }
        leftElement={
          <Layout.Horizontal>
            <Cloud height={20} width={20} style={{ marginRight: '8px', alignItems: 'center' }} />
            <Layout.Vertical spacing="small">
              <Text>Infra Provider Type</Text>
              <Text font="small">Your Gitspace will run on the selected infra provider type</Text>
            </Layout.Vertical>
          </Layout.Horizontal>
        }
        menu={
          <Menu>
            {infraProviders?.map(({ label, value }: dropdownProps) => {
              const providerType = getProviderType(value)
              const providerIcon = getProviderIcon(providerType)
              return (
                <MenuItem
                  key={label}
                  active={label === selectedInfraProvider?.label}
                  text={<Text font={{ size: 'normal', weight: 'bold' }}>{label}</Text>}
                  icon={
                    <div
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        height: '100%',
                        minHeight: '24px'
                      }}>
                      {providerIcon ? (
                        providerType === HARNESS_GCP ? (
                          <img src={providerIcon} height={20} width={20} />
                        ) : (
                          <img src={providerIcon} height={17} width={17} />
                        )
                      ) : (
                        <Cloud height={17} width={17} />
                      )}
                    </div>
                  }
                  onClick={() => {
                    onChange('metadata.infraProvider', value)
                    onChange('metadata.region', undefined)
                    onChange('resource_identifier', undefined)
                    onChange('resource_space_ref', undefined)
                  }}
                />
              )
            })}
          </Menu>
        }
      />
    </Container>
  )
}

export default SelectInfraProviderType
