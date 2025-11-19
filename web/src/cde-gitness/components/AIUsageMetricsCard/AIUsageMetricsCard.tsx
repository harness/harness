import React from 'react'
import { Container, Layout, Text, Tag, Popover } from '@harnessio/uicore'
import { Position } from '@blueprintjs/core'
import { useStrings } from 'framework/strings'
import type { TypesAIUsageMetric } from 'services/cde'
import css from './AIUsageMetricsCard.module.scss'

export interface AIUsageMetricsCardProps {
  metric?: TypesAIUsageMetric | null
}

export const AIUsageMetricsCard: React.FC<AIUsageMetricsCardProps> = ({ metric }) => {
  const { getString } = useStrings()
  const duration =
    metric?.duration_ms !== undefined && metric?.duration_ms !== null
      ? (metric.duration_ms / 1000).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })
      : '—'
  const llmModels: string[] = Array.isArray(metric?.llm_models) ? (metric?.llm_models as string[]) : []
  const totalCost =
    metric?.total_cost_usd !== undefined && metric?.total_cost_usd !== null
      ? `$${metric.total_cost_usd.toFixed(4)}`
      : '—'
  const inputTokens =
    metric?.total_input_tokens !== undefined && metric?.total_input_tokens !== null
      ? metric.total_input_tokens.toLocaleString()
      : '—'
  const outputTokens =
    metric?.total_output_tokens !== undefined && metric?.total_output_tokens !== null
      ? metric.total_output_tokens.toLocaleString()
      : '—'

  const renderModelTags = (): JSX.Element => {
    if (llmModels.length === 0) {
      return <Text className={css.noModelsMessage}>No models</Text>
    }

    const displayTags = llmModels.slice(0, 1)
    const excessTags = llmModels.length - 1
    return (
      <Container className={css.modelTagsWrapper}>
        {displayTags.map((tag: string, index: number) => (
          <Tag key={`model-tag-${index}`} intent="none" className={css.modelTag}>
            <span className={css.modelTagText}>{tag}</span>
          </Tag>
        ))}
        {excessTags > 0 && (
          <Tag className={css.modelTagExtra}>
            <span className={css.modelTagExtraText}>+{excessTags}</span>
          </Tag>
        )}
      </Container>
    )
  }

  const getModelTooltipContent = (): JSX.Element => (
    <Container className={css.modelTooltipContent}>
      {llmModels.map((selector: string, index: number) => (
        <Container key={`tooltip-model-${index}`}>{selector}</Container>
      ))}
    </Container>
  )

  return (
    <Layout.Horizontal width="90%" className={css.detailsContainer} padding={{ top: 'xlarge', bottom: 'xlarge' }}>
      <Layout.Vertical
        spacing="small"
        flex={{ justifyContent: 'center', alignItems: 'flex-start' }}
        className={css.marginLeftContainer}>
        <Text className={css.rowHeaders}>{getString('cde.aiTasks.details.duration')}</Text>
        <Text className={css.providerText}>{duration}</Text>
      </Layout.Vertical>

      <Layout.Vertical spacing="small" flex={{ justifyContent: 'center', alignItems: 'flex-start' }}>
        <Text className={css.rowHeaders}>{getString('cde.aiTasks.details.totalCost')}</Text>
        <Text className={css.providerText}>{totalCost}</Text>
      </Layout.Vertical>

      <Layout.Vertical spacing="small" flex={{ justifyContent: 'center', alignItems: 'flex-start' }}>
        <Text className={css.rowHeaders}>{getString('cde.aiTasks.details.inputTokens')}</Text>
        <Text className={css.providerText}>{inputTokens}</Text>
      </Layout.Vertical>

      <Layout.Vertical spacing="small" flex={{ justifyContent: 'center', alignItems: 'flex-start' }}>
        <Text className={css.rowHeaders}>{getString('cde.aiTasks.details.outputTokens')}</Text>
        <Text className={css.providerText}>{outputTokens}</Text>
      </Layout.Vertical>

      <Layout.Vertical spacing="small" flex={{ justifyContent: 'center', alignItems: 'flex-start' }}>
        <Text className={css.rowHeaders}>{getString('cde.aiTasks.details.llmModels')}</Text>
        {llmModels.length > 0 ? (
          <Popover content={getModelTooltipContent()} position={Position.BOTTOM_RIGHT}>
            <Container className={css.modelTagsContainer}>{renderModelTags()}</Container>
          </Popover>
        ) : (
          <Container className={css.modelTagsContainer}>
            <Text className={css.noModelsMessage}> — </Text>
          </Container>
        )}
      </Layout.Vertical>
    </Layout.Horizontal>
  )
}

export default AIUsageMetricsCard
