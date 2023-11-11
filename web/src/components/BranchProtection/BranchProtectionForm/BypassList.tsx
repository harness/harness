import React, { useMemo } from 'react'
import cx from 'classnames'
import { Icon } from '@harnessio/icons'
import { Avatar, Container, FlexExpander, Layout, Text } from '@harnessio/uicore'
import css from './BranchProtectionForm.module.scss'

const BypassList = (props: {
  bypassList?: string[] // eslint-disable-next-line @typescript-eslint/no-explicit-any
  setFieldValue: (field: string, value: any, shouldValidate?: boolean | undefined) => void
}) => {
  const { bypassList, setFieldValue } = props
  const bypassContent = useMemo(() => {
    return (
      <Container className={cx(css.widthContainer, css.bypassContainer)}>
        {bypassList?.map((owner: string, idx: number) => {
          const name = owner.slice(owner.indexOf(' ') + 1)
          return (
            <Layout.Horizontal key={`${name}-${idx}`} flex={{ align: 'center-center' }} padding={'small'}>
              <Avatar hoverCard={false} size="small" name={name.toString()} />
              <Text padding={{ top: 'tiny' }} lineClamp={1}>
                {name}
              </Text>
              <FlexExpander />
              <Icon
                name="code-close"
                onClick={() => {
                  const filteredData = bypassList.filter(item => !(item[0] === owner[0] && item[1] === owner[1]))
                  setFieldValue('bypassList', filteredData)
                }}
                className={css.codeClose}
              />
            </Layout.Horizontal>
          )
        })}
      </Container>
    )
  }, [bypassList, setFieldValue])
  return <>{bypassContent}</>
}

export default BypassList
