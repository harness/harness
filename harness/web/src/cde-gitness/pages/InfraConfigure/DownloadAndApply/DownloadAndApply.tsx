import React, { useEffect } from 'react'
import { Button, ButtonVariation, CodeBlock, Layout, Page, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useHistory, useParams } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { useGetInfraDetails } from 'cde-gitness/hooks/useInfraDetailAPI'
import { routes } from 'cde-gitness/RouteDefinitions'
import Index1 from '../../../../icons/num1.svg?url'
import Index2 from '../../../../icons/num2.svg?url'
import css from './DownloadAndApply.module.scss'

interface RouteParamsProps {
  infraprovider_identifier?: string
  provider: string
}

const DownloadAndApplySection = () => {
  const { getString } = useStrings()
  const { accountInfo } = useAppContext()
  const history = useHistory()
  const { infraprovider_identifier, provider } = useParams<RouteParamsProps>()

  const { data } = useGetInfraDetails({
    accountIdentifier: accountInfo?.identifier,
    infraprovider_identifier: infraprovider_identifier ?? 'undefined',
    queryParams: {
      acl_filter: 'false'
    }
  })

  useEffect(() => {
    if (!infraprovider_identifier) {
      return history.push(routes.toCDEGitspaceInfra({ accountId: accountInfo?.identifier }))
    }
  }, [infraprovider_identifier])

  const downloadYaml = () => {
    const blob = new Blob([data?.setup_yaml ?? ''], { type: 'text/yaml' })

    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = 'data.yaml'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
  }

  return (
    <Page.Body className={css.main}>
      <Layout.Vertical spacing={'medium'}>
        <Text className={css.pageTitle} color={Color.GREY_1000}>
          {getString('cde.downloadAndApplySection.title')}
        </Text>
        <Layout.Vertical spacing="medium" className={css.bodyContainer}>
          <Layout.Vertical spacing={'medium'}>
            <Text icon={<img src={Index1} width={16} />} className={css.listTitle}>
              {getString('cde.downloadAndApplySection.downloadGeneratedYaml')}
            </Text>
            <Button
              className={css.buttonStyle}
              icon={'download-manifests-inverse'}
              iconProps={{ size: 14 }}
              text={getString('cde.downloadAndApplySection.downloadYaml')}
              variation={ButtonVariation.PRIMARY}
              onClick={downloadYaml}
            />
          </Layout.Vertical>
          <Layout.Vertical spacing="none">
            <Text icon={<img src={Index2} width={16} />} className={css.listTitle}>
              {getString('cde.downloadAndApplySection.applyYamlText')}
            </Text>
            <Layout.Vertical spacing={'medium'} className={css.codeBlockContainer}>
              <CodeBlock allowCopy format="pre" snippet={'tofu apply [options] [plan file]'} />
              <CodeBlock allowCopy format="pre" snippet={'tofu apply [options] [plan file]'} />
            </Layout.Vertical>
          </Layout.Vertical>
        </Layout.Vertical>
        <Layout.Horizontal className={css.formFooter}>
          <Button
            text={getString('cde.downloadAndApplySection.back')}
            icon="chevron-left"
            variation={ButtonVariation.SECONDARY}
            onClick={() => {
              history.push(
                routes.toCDEInfraConfigureDetail({
                  accountId: accountInfo?.identifier,
                  infraprovider_identifier: infraprovider_identifier ?? '',
                  provider: provider
                })
              )
            }}
          />
          <Button
            text={getString('cde.downloadAndApplySection.done')}
            variation={ButtonVariation.PRIMARY}
            onClick={() => {
              const baseUrl = routes.toCDEGitspaceInfra({ accountId: accountInfo?.identifier })
              const urlWithQuery = provider === 'hybrid_vm_aws' ? `${baseUrl}?type=hybrid_vm_aws` : baseUrl
              history.push(urlWithQuery)
            }}
          />
        </Layout.Horizontal>
      </Layout.Vertical>
    </Page.Body>
  )
}

export default DownloadAndApplySection
