import React from 'react'
import { Container, Layout, Text, Tag, Popover } from '@harnessio/uicore'
import { Position } from '@blueprintjs/core'
import { useStrings } from 'framework/strings'
import type { StringKeys } from 'framework/strings'
import type { TypesInfraProviderConfig } from 'services/cde'
import AWSIcon from 'cde-gitness/assests/aws.svg?url'
import { HYBRID_VM_GCP, HYBRID_VM_AWS } from 'cde-gitness/constants'
import { formatLastUpdated } from 'cde-gitness/utils/time.utils'
import GatewayDetails from './GatewayDetails'
import VMRunnerDetails from './VMRunnerDetails'
import GCPIcon from '../../../icons/google-cloud.svg?url'
import css from './InfraDetailCard.module.scss'

interface InfraDetailCardProps {
  infraDetails: TypesInfraProviderConfig
  regionCount: number
  provider: string
}

type CardField = {
  stringKey: StringKeys
  value: string | number | undefined
  customRender?: JSX.Element
  className?: string
  colSpan?: number
}

function InfraDetailCard({ infraDetails, regionCount, provider }: InfraDetailCardProps) {
  const { getString } = useStrings()
  const gatewayDetailsOpen = true
  const runnerDetailsOpen = true

  const renderDelegateTags = (): JSX.Element => {
    const delegateSelectors =
      infraDetails?.metadata?.delegate_selectors?.map((d: { selector: string }) => d.selector) || []

    if (delegateSelectors.length === 0) {
      return <Text className={css.noTagsMessage}>No delegate selectors selected</Text>
    }

    const displayTags = delegateSelectors.slice(0, 2)
    const excessTags = delegateSelectors.length - 2

    return (
      <Container className={css.delegateTagsWrapper}>
        {displayTags.map((tag: string, index: number) => (
          <Tag key={`delegate-tag-${index}`} intent="none" className={css.delegateTag}>
            {tag}
          </Tag>
        ))}
        {excessTags > 0 && (
          <Tag className={css.delegateTagExtra}>
            <span className={css.delegateTagExtraText}>+{excessTags}</span>
          </Tag>
        )}
      </Container>
    )
  }

  const getDelegateTooltipContent = (): JSX.Element => {
    const delegateSelectors =
      infraDetails?.metadata?.delegate_selectors?.map((d: { selector: string }) => d.selector) || []

    return (
      <Container className={css.delegateTooltipContent}>
        {delegateSelectors.map((selector: string, index: number) => (
          <Container key={`tooltip-selector-${index}`}>{selector}</Container>
        ))}
      </Container>
    )
  }

  const providerConfigs: Record<string, { icon: string; fields: CardField[] }> = {
    [HYBRID_VM_GCP]: {
      icon: GCPIcon,
      fields: [
        { stringKey: 'cde.configureInfra.domain', value: infraDetails?.metadata?.domain },
        { stringKey: 'cde.configureInfra.numberOfLocations', value: regionCount },
        {
          stringKey: 'cde.delegate.delegateSelectorTags',
          value: undefined,
          customRender:
            infraDetails?.metadata?.delegate_selectors?.length > 0 ? (
              <Popover content={getDelegateTooltipContent()} position={Position.BOTTOM_RIGHT}>
                <Container className={css.delegateTagsContainer}>{renderDelegateTags()}</Container>
              </Popover>
            ) : (
              <Container className={css.delegateTagsContainer}>
                <Text className={css.noTagsMessage}>{getString('cde.delegate.noDelegateSelectors')}</Text>
              </Container>
            )
        },
        {
          stringKey: 'cde.configureInfra.updateTime',
          value: formatLastUpdated(infraDetails?.updated)
        }
      ]
    },
    [HYBRID_VM_AWS]: {
      icon: AWSIcon,
      fields: [
        { stringKey: 'cde.Aws.VpcCidrBlock', value: infraDetails?.metadata?.vpc_cidr_block },
        { stringKey: 'cde.configureInfra.domain', value: infraDetails?.metadata?.domain },
        { stringKey: 'cde.Aws.numberOfRegions', value: regionCount },
        {
          stringKey: 'cde.delegate.delegateSelectorTags',
          value: undefined,
          customRender:
            infraDetails?.metadata?.delegate_selectors?.length > 0 ? (
              <Popover content={getDelegateTooltipContent()} position={Position.BOTTOM_RIGHT}>
                <Container className={css.delegateTagsContainer}>{renderDelegateTags()}</Container>
              </Popover>
            ) : (
              <Container className={css.delegateTagsContainer}>
                <Text className={css.noTagsMessage}>{getString('cde.delegate.noDelegateSelectors')}</Text>
              </Container>
            )
        },
        {
          stringKey: 'cde.configureInfra.updateTime',
          value: formatLastUpdated(infraDetails?.updated)
        }
      ]
    }
  }

  const currentConfig = providerConfigs[provider] || providerConfigs[HYBRID_VM_GCP]

  return (
    <Container className={css.infraDetailCard}>
      <Layout.Vertical spacing={'normal'}>
        <Layout.Horizontal spacing={'normal'}>
          <img src={currentConfig.icon} width={24} />
          <Text className={css.cardTitle}>{getString('cde.gitspaceInfraHome.infraDetails')}</Text>
        </Layout.Horizontal>
        <Container className={css.cardGridContainer}>
          {currentConfig.fields.map((field, index) => (
            <Container key={index} className={`${css.cardGridItem}`}>
              <Text className={css.rowHeader}>{getString(field.stringKey)}</Text>
              {field.customRender ? field.customRender : <Text className={css.rowContent}>{field.value}</Text>}
            </Container>
          ))}
        </Container>
        {/* Gateway Image Details */}
        <GatewayDetails infraDetails={infraDetails} provider={provider} initialOpen={gatewayDetailsOpen} />
        <VMRunnerDetails infraDetails={infraDetails} provider={provider} initialOpen={runnerDetailsOpen} />
      </Layout.Vertical>
    </Container>
  )
}

export default InfraDetailCard
