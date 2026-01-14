import React from 'react'
import { Button, ButtonVariation, Container, Dialog, Layout, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import type { Violation } from 'utils/Utils'
import css from './RuleViolationAlertModal.module.scss'
interface ViolationAlertModalProps {
  setOpen: (val: boolean) => void
  open: boolean
  title: string
  text: string
  rules: Violation[] | undefined
}

const RuleViolationAlertModal = (props: ViolationAlertModalProps) => {
  const { title, text, rules, setOpen, open } = props
  const { getString } = useStrings()

  return (
    <Dialog
      className={css.dialog}
      onClose={() => {
        setOpen(false)
      }}
      isOpen={open}>
      <Container>
        <Text
          icon={'warning-sign'}
          iconProps={{ color: Color.RED_600, size: 20, className: css.warningIcon }}
          font={{ variation: FontVariation.H4 }}
          padding={{ right: 'small' }}>
          {title}
        </Text>
        <Text padding={{ top: 'medium' }} font={{ variation: FontVariation.BODY2 }}>
          {text}
        </Text>
        <Layout.Vertical spacing="small" padding={{ top: 'medium', bottom: 'medium' }}>
          {rules?.map((rule, idx) => {
            return (
              <Container key={`violation-${idx}`} flex={{ alignItems: 'center' }} className={css.ruleContainer}>
                <Text padding="small" lineClamp={1} className={css.ruleText}>
                  {rule.violation}
                </Text>
              </Container>
            )
          })}
        </Layout.Vertical>
        <Layout.Horizontal spacing="small">
          <Button
            variation={ButtonVariation.TERTIARY}
            text={getString('cancel')}
            onClick={() => {
              setOpen(false)
            }}></Button>
        </Layout.Horizontal>
      </Container>
    </Dialog>
  )
}

export default RuleViolationAlertModal
