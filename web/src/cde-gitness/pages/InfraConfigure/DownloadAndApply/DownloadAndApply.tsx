import React from 'react'
import { Button, ButtonVariation, CodeBlock, Layout, Page, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useHistory } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { routes } from 'cde-gitness/RouteDefinitions'
import Index1 from '../../../../icons/num1.svg?url'
import Index2 from '../../../../icons/num2.svg?url'
import css from './DownloadAndApply.module.scss'

interface InfraDetailProps {
  onTabChange: (key: string) => void
  tabOptions: { [key: string]: string }
}

const DownloadAndApplySection = ({ onTabChange, tabOptions }: InfraDetailProps) => {
  const { getString } = useStrings()
  const { accountInfo } = useAppContext()
  const history = useHistory()
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
            onClick={() => onTabChange(tabOptions.tab1)}
          />
          <Button
            text={getString('cde.downloadAndApplySection.done')}
            variation={ButtonVariation.PRIMARY}
            onClick={() => history.push(routes.toCDEGitspaceInfra({ accountId: accountInfo?.identifier }))}
          />
        </Layout.Horizontal>
      </Layout.Vertical>
    </Page.Body>
  )
}

export default DownloadAndApplySection
