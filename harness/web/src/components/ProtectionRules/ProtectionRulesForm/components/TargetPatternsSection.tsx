import React, { useState } from 'react'
import cx from 'classnames'
import { Color, FontVariation } from '@harnessio/design-system'
import { ButtonVariation, Container, Layout, SplitButton, Text, TextInput } from '@harnessio/uicore'
import type { FormikProps } from 'formik'
import { Menu, PopoverPosition } from '@blueprintjs/core'
import { Icon } from '@harnessio/icons'
import { CodeIcon, ProtectionRulesType, RulesTargetType } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import Include from '../../../../icons/Include.svg?url'
import Exclude from '../../../../icons/Exclude.svg?url'
import type { RulesFormPayload } from '../../ProtectionRulesUtils'
import css from '../ProtectionRulesForm.module.scss'

export function TargetPatterns({
  formik,
  fieldName,
  targets
}: {
  formik: FormikProps<RulesFormPayload>
  fieldName: string
  targets?: string[][]
}) {
  if (!targets?.length) return null

  return (
    <Layout.Horizontal spacing={'small'} className={css.targetBox}>
      {targets.map((tgt, idx) => {
        return (
          <Container key={`${idx}-${tgt[1]}`} className={cx(css.greyButton, css.target)}>
            <img width={16} height={16} src={tgt[0] === RulesTargetType.INCLUDE ? Include : Exclude} />
            <Text lineClamp={1}>{tgt[1]}</Text>
            <Icon
              name="code-close"
              onClick={() => {
                const filteredData = targets.filter(item => !(item[0] === tgt[0] && item[1] === tgt[1]))
                formik.setFieldValue(fieldName, filteredData)
              }}
              className={css.codeClose}
            />
          </Container>
        )
      })}
    </Layout.Horizontal>
  )
}

const TargetPatternsSection = ({
  formik,
  repoTarget,
  ruleType,
  tooltipId
}: {
  formik: FormikProps<RulesFormPayload>
  repoTarget?: boolean
  ruleType?: ProtectionRulesType
  tooltipId?: string
}) => {
  const { getString } = useStrings()
  const [target, setTarget] = useState('')
  const [targetType, setTargetType] = useState(RulesTargetType.INCLUDE)
  const { targetList = [], repoTargetList = [] } = formik.values
  const targets = repoTarget ? repoTargetList : targetList

  return (
    <>
      <Layout.Horizontal>
        <TextInput
          value={target}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => setTarget(e.currentTarget.value)}
          placeholder={getString('protectionRules.targetPlaceholder', { target: repoTarget ? 'repository' : ruleType })}
          {...(tooltipId && { tooltipProps: { dataTooltipId: tooltipId } })}
          className={cx(css.widthContainer, css.label)}
          wrapperClassName={css.targetSpacingContainer}
        />
        <Container flex={{ alignItems: 'flex-end' }} padding={{ left: 'medium' }}>
          <SplitButton
            className={css.buttonContainer}
            variation={ButtonVariation.TERTIARY}
            text={
              <Container flex={{ alignItems: 'center' }}>
                <img width={16} height={16} src={targetType === RulesTargetType.INCLUDE ? Include : Exclude} />
                <Text
                  padding={{ left: 'xsmall' }}
                  color={Color.BLACK}
                  font={{ variation: FontVariation.BODY2_SEMI, weight: 'bold' }}>
                  {getString(targetType)}
                </Text>
              </Container>
            }
            popoverProps={{
              interactionKind: 'click',
              usePortal: true,
              popoverClassName: css.popover,
              position: PopoverPosition.BOTTOM_RIGHT
            }}
            onClick={() => {
              if (target !== '') {
                targets.push([targetType, target])
                formik.setFieldValue(repoTarget ? 'repoTargetList' : 'targetList', targets)
                setTarget('')
              }
            }}>
            {Object.values(RulesTargetType).map(type => (
              <Menu.Item
                key={type}
                className={css.menuItem}
                text={
                  <Container flex={{ justifyContent: 'flex-start' }}>
                    <Icon name={type === targetType ? CodeIcon.Tick : CodeIcon.Blank} />
                    <Text
                      padding={{ left: 'xsmall' }}
                      color={Color.BLACK}
                      font={{ variation: FontVariation.BODY2_SEMI, weight: 'bold' }}>
                      {getString(type)}
                    </Text>
                  </Container>
                }
                onClick={() => setTargetType(type)}
              />
            ))}
          </SplitButton>
        </Container>
      </Layout.Horizontal>
      <Text className={css.hintText} margin={{ top: 'xsmall', bottom: 'small' }}>
        {getString('protectionRules.targetPatternHint', { target: repoTarget ? 'repository' : ruleType })}
      </Text>
      <TargetPatterns formik={formik} fieldName={repoTarget ? 'repoTargetList' : 'targetList'} targets={targets} />
    </>
  )
}

export default TargetPatternsSection
