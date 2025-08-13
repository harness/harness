import { Container, Layout, Text } from '@harnessio/uicore'
import React from 'react'
import { Cloud } from 'iconoir-react'
import { Menu, MenuItem } from '@blueprintjs/core'
import { useFormikContext } from 'formik'
import { Color } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import type { OpenapiCreateGitspaceRequest, TypesInfraProviderConfig } from 'services/cde'
import { HARNESS_GCP } from 'cde-gitness/constants'
import infrasvg from 'icons/Infrastructure.svg?url'
import type { dropdownProps } from 'cde-gitness/constants'
import getProviderIcon from '../../utils/InfraProvider.utils'
import { CDECustomDropdown } from '../CDECustomDropdown/CDECustomDropdown'
import css from './SelectInfraProviderType.module.scss'

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
                style={{ marginRight: '12px', display: 'flex', alignSelf: 'center' }}
              />
            )}
            {!selectedProviderIcon && (
              <Cloud height={16} width={16} style={{ marginRight: '12px', display: 'flex', alignSelf: 'center' }} />
            )}
            <Layout.Vertical>
              <Text font={'normal'}>{selectedInfraProvider?.label || getString('cde.infraProvider')}</Text>
            </Layout.Vertical>
          </Layout.Horizontal>
        }
        leftElement={
          <Layout.Horizontal>
            <img src={infrasvg} className={css.icon} />
            <Layout.Vertical spacing="small">
              <Text color={Color.GREY_500} font={{ weight: 'bold' }}>
                {getString('cde.create.infraProviderType')}
              </Text>
              <Text font="small">{getString('cde.create.infraProviderTypeText')}</Text>
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
