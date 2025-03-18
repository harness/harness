import React, { useMemo } from 'react'
import cx from 'classnames'
import { Icon } from '@harnessio/icons'
import { Container, FlexExpander, Layout, Text } from '@harnessio/uicore'
import { Classes, Popover, PopoverInteractionKind, PopoverPosition } from '@blueprintjs/core'
import { Color, FontVariation } from '@harnessio/design-system'
import css from '../BranchProtectionForm.module.scss'

const DefaultReviewersList = (props: {
  defaultReviewersList?: string[] // eslint-disable-next-line @typescript-eslint/no-explicit-any
  setFieldValue: (field: string, value: any, shouldValidate?: boolean | undefined) => void
}) => {
  const { defaultReviewersList, setFieldValue } = props
  const defaultReviewerContent = useMemo(() => {
    return (
      <Layout.Horizontal className={cx(css.widthContainer, css.defaultReviewerContainer)} padding={{ bottom: 'large' }}>
        {defaultReviewersList?.map((owner: string, idx: number) => {
          const str = owner.slice(owner.indexOf(' ') + 1)
          const name = str.split(' (')[0]
          const email = str.split(' (')[1].replace(')', '')
          return (
            <Popover
              key={`${name}-${idx}`}
              interactionKind={PopoverInteractionKind.HOVER}
              position={PopoverPosition.TOP_LEFT}
              popoverClassName={Classes.DARK}
              content={
                <Container padding="medium">
                  <Text font={{ variation: FontVariation.FORM_HELP }} color={Color.WHITE}>
                    {email}
                  </Text>
                </Container>
              }>
              <Layout.Horizontal key={`${name}-${idx}`} flex={{ align: 'center-center' }} className={css.reviewerBlock}>
                <Text padding={{ top: 'tiny' }} lineClamp={1}>
                  {name}
                </Text>
                <FlexExpander />
                <Icon
                  name="code-close"
                  onClick={() => {
                    const filteredData = defaultReviewersList.filter(item => !(item === owner))
                    setFieldValue('defaultReviewersList', filteredData)
                  }}
                  className={css.codeCloseBtn}
                />
              </Layout.Horizontal>
            </Popover>
          )
        })}
      </Layout.Horizontal>
    )
  }, [defaultReviewersList, setFieldValue])
  return <>{defaultReviewerContent}</>
}

export default DefaultReviewersList
