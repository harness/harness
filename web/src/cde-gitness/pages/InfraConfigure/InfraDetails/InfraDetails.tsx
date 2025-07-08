import React, { useEffect, useState } from 'react'
import { Button, ButtonVariation, Container, Formik, FormikForm, Layout, Page, useToaster } from '@harnessio/uicore'
import { useHistory, useParams } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { routes } from 'cde-gitness/RouteDefinitions'
import { useAppContext } from 'AppContext'
import { HYBRID_VM_GCP, regionProp } from 'cde-gitness/constants'
import { OpenapiCreateInfraProviderConfigRequest, useCreateInfraProvider, useUpdateInfraProvider } from 'services/cde'
import { getErrorMessage } from 'utils/Utils'
import { useGetInfraDetails } from 'cde-gitness/hooks/useInfraDetailAPI'
import { validateInfraForm } from '../../../utils/InfraValidations.utils'
import BasicDetails from './BasicDetails'
import ConfigureLocations from './ConfigureLocations'
import css from './InfraDetails.module.scss'

interface RouteParamsProps {
  infraprovider_identifier?: string
}

interface InfraDetailsFormikProps {
  identifier?: string
  name?: string
  domain?: string
  machine_type?: string
  instances?: string
  project?: string
  delegateSelector?: string[]
  runner?: { region: string; zone: string }
}

const InfraDetails = () => {
  const initialData = {
    location: '',
    defaultSubnet: '',
    proxySubnet: '',
    domain: '',
    dns: '',
    identifier: 1
  }
  const history = useHistory()
  const { getString } = useStrings()
  const { accountInfo } = useAppContext()
  const { showSuccess, showError } = useToaster()
  const [regionData, setRegionData] = useState<regionProp[]>([])
  const { infraprovider_identifier } = useParams<RouteParamsProps>()
  const [infraDetails, setInfraDetails] = useState<InfraDetailsFormikProps>({})
  const { data } = useGetInfraDetails({
    infraprovider_identifier: infraprovider_identifier ?? 'undefined',
    accountIdentifier: accountInfo?.identifier
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
      const { identifier, metadata }: OpenapiCreateInfraProviderConfigRequest = data
      const delegate = metadata?.delegate_selectors?.map((del: { selector: string }) => del.selector)
      const payload: InfraDetailsFormikProps = {
        identifier: identifier,
        name: metadata?.name,
        domain: metadata?.domain,
        machine_type: metadata?.gateway?.machine_type,
        instances: metadata?.gateway?.instances,
        project: metadata?.project?.id,
        delegateSelector: delegate,
        runner: metadata?.runner
      }
      const regions: regionProp[] = []
      Object?.keys(data?.metadata?.region_configs ?? {}).forEach((key: string, index: number) => {
        const { certificates, region_name, proxy_subnet_ip_range, default_subnet_ip_range }: any =
          data?.metadata?.region_configs[key]
        const region: regionProp = {
          location: region_name,
          defaultSubnet: default_subnet_ip_range,
          proxySubnet: proxy_subnet_ip_range,
          domain: certificates?.contents?.[0]?.domain,
          identifier: index + 1
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
        infraprovider_identifier: infraprovider_identifier ?? identifier
      })
    )
  }

  return (
    <Page.Body className={css.main}>
      <Container className={css.basicDetailsContainer}>
        <Formik
          formName="edit-layout-name"
          onSubmit={async (values: InfraDetailsFormikProps) => {
            try {
              if (regionData?.length > 0) {
                const { identifier, name, domain, machine_type, instances, project, delegateSelector, runner } = values
                const region_configs: Unknown = {}
                regionData?.forEach((region: regionProp) => {
                  const { location, defaultSubnet, proxySubnet, domain: regionDomain } = region
                  // const regionKey = location?.replace(/-/g, '')
                  region_configs[location] = {
                    region_name: location,
                    default_subnet_ip_range: defaultSubnet,
                    proxy_subnet_ip_range: proxySubnet,
                    certificates: {
                      contents: [
                        {
                          domain: regionDomain
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
                    runner,
                    delegate_selectors: delegates,
                    name,
                    region_configs,
                    project: {
                      id: project
                    },
                    gateway: {
                      machine_type,
                      instances: parseInt(instances || '1')
                    }
                  },
                  name,
                  space_ref: '',
                  type: HYBRID_VM_GCP
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
          }}
          initialValues={infraDetails ?? {}}
          validationSchema={validateInfraForm(getString)}
          enableReinitialize>
          {formikProps => {
            return (
              <FormikForm>
                <Layout.Vertical spacing="medium">
                  <BasicDetails formikProps={formikProps} />
                  {/* <GatewayDetails formikProps={formikProps} /> */}
                  <ConfigureLocations
                    regionData={regionData}
                    setRegionData={setRegionData}
                    initialData={initialData}
                    runner={formikProps?.values?.runner || { region: '', zone: '' }}
                    setRunner={result => formikProps?.setFieldValue('runner', result)}
                  />
                  <Layout.Horizontal className={css.formFooter}>
                    <Button
                      text={getString('cde.configureInfra.cancel')}
                      variation={ButtonVariation.TERTIARY}
                      onClick={() => history.push(routes.toCDEGitspaceInfra({ accountId: accountInfo?.identifier }))}
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

export default InfraDetails
