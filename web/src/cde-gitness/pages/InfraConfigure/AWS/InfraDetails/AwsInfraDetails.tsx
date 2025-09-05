import React, { useEffect, useState } from 'react'
import { Button, ButtonVariation, Container, Formik, FormikForm, Layout, Page, useToaster } from '@harnessio/uicore'
import { useHistory, useParams } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { routes } from 'cde-gitness/RouteDefinitions'
import { useAppContext } from 'AppContext'
import { AwsRegionConfig, HYBRID_VM_AWS, ZoneConfig } from 'cde-gitness/constants'
import { OpenapiCreateInfraProviderConfigRequest, useCreateInfraProvider, useUpdateInfraProvider } from 'services/cde'
import { getErrorMessage } from 'utils/Utils'
import { useGetInfraDetails } from 'cde-gitness/hooks/useInfraDetailAPI'
import { validateAwsInfraForm } from 'cde-gitness/utils/InfraValidations.utils'
import BasicDetails from './BasicDetails'
import ConfigureLocations from './ConfigureLocations'
import css from './InfraDetails.module.scss'

interface RouteParamsProps {
  infraprovider_identifier?: string
}

interface InfraDetailsFormikProps {
  identifier?: string
  name?: string
  type?: string
  domain?: string
  instance_type?: string
  instances?: string
  delegateSelector?: string[]
  vpc_cidr_block?: string
  runner?: { region: string; availability_zones: string; ami_id: string }
}

interface ExtendedAwsRegionConfig extends AwsRegionConfig {
  zones?: ZoneConfig[]
  identifier: number
}

const AwsInfraDetails = () => {
  const initialData: ExtendedAwsRegionConfig = {
    region_name: '',
    gateway_ami_id: '',
    domain: '',
    private_cidr_block: '',
    public_cidr_block: '',
    zone: '',
    zones: [],
    identifier: 1
  }
  const history = useHistory()
  const { getString } = useStrings()
  const { accountInfo } = useAppContext()
  const { showSuccess, showError } = useToaster()
  const [regionData, setRegionData] = useState<ExtendedAwsRegionConfig[]>([])
  const { infraprovider_identifier } = useParams<RouteParamsProps>()
  const [infraDetails, setInfraDetails] = useState<InfraDetailsFormikProps>({})
  const { data } = useGetInfraDetails({
    infraprovider_identifier: infraprovider_identifier ?? '',
    accountIdentifier: accountInfo?.identifier,
    queryParams: {
      acl_filter: 'false'
    }
  })

  const { mutate } = useCreateInfraProvider({
    accountIdentifier: accountInfo?.identifier
  })
  const { mutate: updateInfraProvider } = useUpdateInfraProvider({
    accountIdentifier: accountInfo?.identifier,
    infraprovider_identifier: infraprovider_identifier ?? ''
  })

  useEffect(() => {
    if (data) {
      const { identifier, name, metadata }: OpenapiCreateInfraProviderConfigRequest = data
      const delegate = metadata?.delegate_selectors?.map((del: { selector: string }) => del.selector)
      const payload: InfraDetailsFormikProps = {
        identifier: identifier,
        name: name,
        domain: metadata?.domain,
        instance_type: metadata?.gateway?.instance_type,
        instances: metadata?.gateway?.instances,
        delegateSelector: delegate,
        vpc_cidr_block: metadata?.vpc_cidr_block, // Extract VPC CIDR block from API data
        runner: metadata?.runner
      }

      const regions: ExtendedAwsRegionConfig[] = []
      Object?.keys(data?.metadata?.region_configs ?? {}).forEach((key: string, index: number) => {
        const { certificates, region_name }: any = data?.metadata?.region_configs[key]

        // Extract availability zones from the API response and map them to the ZoneConfig format
        const availabilityZones = data?.metadata?.region_configs[key]?.availability_zones || []
        const zones: ZoneConfig[] = availabilityZones.map((az: any, azIndex: number) => ({
          zone: az.zone,
          privateSubnet: az.private_cidr_block,
          publicSubnet: az.public_cidr_block,
          id: azIndex
        }))

        // Get the gateway_ami_id from the region-specific configuration
        const regionGatewayAmiId = data?.metadata?.region_configs[key]?.gateway_ami_id

        const region: ExtendedAwsRegionConfig = {
          region_name, // This comes from the key in region_configs
          private_cidr_block: availabilityZones[0]?.private_cidr_block || '',
          public_cidr_block: availabilityZones[0]?.public_cidr_block || '',
          domain: certificates?.contents?.[0]?.domain || '',
          gateway_ami_id: regionGatewayAmiId || '', // Use the region-specific gateway_ami_id
          zone: availabilityZones[0]?.zone || '',
          zones: zones, // Add the zones array for the renderRowSubComponent
          identifier: index
        }
        regions.push(region)
      })
      setRegionData(regions)
      setInfraDetails(payload)
    }
  }, [data])

  const navigateToDownload = (identifier: string) => {
    history.push(
      routes.toCDEInfraConfigureDetailDownload({
        accountId: accountInfo?.identifier,
        infraprovider_identifier: infraprovider_identifier ?? identifier,
        provider: HYBRID_VM_AWS
      })
    )
  }

  const handleSubmit = async (values: InfraDetailsFormikProps) => {
    try {
      if (regionData?.length > 0) {
        const { identifier, name, domain, instance_type, instances, delegateSelector, vpc_cidr_block, runner } = values
        const region_configs: Record<string, any> = {}

        regionData?.forEach((region: ExtendedAwsRegionConfig) => {
          const regionName = region.region_name || (region as any).location
          const gatewayAmiId = region.gateway_ami_id || (region as any).gatewayAmiId
          const { domain: regionDomain, zones } = region
          const availability_zones =
            zones?.map(zone => ({
              zone: zone.zone,
              private_cidr_block: zone.privateSubnet,
              public_cidr_block: zone.publicSubnet
            })) || []
          region_configs[regionName] = {
            region_name: regionName,
            gateway_ami_id: gatewayAmiId,
            domain: regionDomain,
            availability_zones,
            certificates: {
              contents: [
                {
                  domain: regionDomain,
                  // Include DNS managed zone name if needed
                  dns_managed_zone_name: regionDomain?.replace(/\./g, '-').toLowerCase() || ''
                }
              ]
            }
          }
        })
        const delegates = delegateSelector?.map((del: string) => ({ selector: del }))
        const payload: OpenapiCreateInfraProviderConfigRequest = {
          identifier,
          metadata: {
            domain,
            delegate_selectors: delegates,
            name: identifier,
            region_configs,
            runner: {
              region: runner?.region,
              availability_zones: runner?.availability_zones, // Change from zone
              ami_id: runner?.ami_id
            },
            gateway: {
              instance_type: instance_type,
              instances: parseInt(instances || '3')
            },
            vpc_cidr_block
          },
          name,
          space_ref: '',
          type: HYBRID_VM_AWS
        }
        if (infraprovider_identifier) {
          await updateInfraProvider({ ...payload })
          showSuccess(getString('cde.update.infraProviderSuccess'))
          navigateToDownload(identifier ?? '')
        } else {
          await mutate({ ...payload })
          showSuccess(getString('cde.create.infraProviderSuccess'))
          navigateToDownload(identifier ?? '')
        }
      } else {
        showError(getString('cde.atleastOneRegion'))
      }
    } catch (err) {
      showError(
        infraprovider_identifier
          ? getString('cde.update.infraProviderFailed')
          : getString('cde.create.infraProviderFailed')
      )
      showError(getErrorMessage(err))
    }
  }

  return (
    <Page.Body className={css.main}>
      <Container className={css.basicDetailsContainer}>
        <Formik
          formName="edit-layout-name"
          onSubmit={handleSubmit}
          initialValues={infraDetails ?? {}}
          validationSchema={validateAwsInfraForm(getString)}
          enableReinitialize
          validateOnBlur={true}>
          {formikProps => {
            return (
              <FormikForm>
                <Layout.Vertical spacing="medium">
                  <BasicDetails formikProps={formikProps} />
                  <ConfigureLocations
                    regionData={regionData}
                    setRegionData={setRegionData}
                    initialData={initialData}
                    runner={formikProps?.values?.runner || { region: '', availability_zones: '', ami_id: '' }}
                    setRunner={result => formikProps?.setFieldValue('runner', result)}
                    formikProps={formikProps}
                  />
                  <Layout.Horizontal className={css.formFooter}>
                    <Button
                      text={getString('cde.configureInfra.cancel')}
                      variation={ButtonVariation.TERTIARY}
                      onClick={() =>
                        history.push(
                          `${routes.toCDEGitspaceInfra({ accountId: accountInfo?.identifier })}?type=${HYBRID_VM_AWS}`
                        )
                      }
                    />
                    <Button
                      text={getString('cde.configureInfra.downloadAndApply')}
                      rightIcon="chevron-right"
                      variation={ButtonVariation.PRIMARY}
                      type="submit"
                    />
                  </Layout.Horizontal>
                </Layout.Vertical>
              </FormikForm>
            )
          }}
        </Formik>
      </Container>
    </Page.Body>
  )
}

export default AwsInfraDetails
