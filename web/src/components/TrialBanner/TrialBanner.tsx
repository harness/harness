import React, { ReactElement } from 'react'
import cx from 'classnames'
import moment from 'moment'
import { Container, Icon, Text } from '@harness/uicore'
import { Color } from '@harness/design-system'
import { useGetTrialInfo } from 'utils/GovernanceUtils'
import { useStrings } from 'framework/strings'
import css from './TrialBanner.module.scss'

const TrialBanner = (): ReactElement => {
  const trialInfo = useGetTrialInfo()
  const { getString } = useStrings()

  if (!trialInfo) return <></>

  const { expiryTime } = trialInfo

  const time = moment(trialInfo.expiryTime)
  const days = Math.round(time.diff(moment.now(), 'days', true))
  const expiryDate = time.format('DD MMM YYYY')
  const isExpired = expiryTime !== -1 && days < 0
  const expiredDays = Math.abs(days)

  const expiryMessage = isExpired
    ? getString('banner.expired', {
        days: expiredDays
      })
    : getString('banner.expiryCountdown', {
        days
      })

  const bannerMessage = `Harness Policy Engine trial ${expiryMessage} on ${expiryDate}`
  const bannerClassnames = cx(css.banner, isExpired ? css.expired : css.expiryCountdown)
  const color = isExpired ? Color.RED_700 : Color.ORANGE_700

  return (
    <Container
      padding="small"
      intent="warning"
      flex={{
        justifyContent: 'start'
      }}
      className={bannerClassnames}
      font={{
        align: 'center'
      }}>
      <Icon name={'warning-sign'} size={15} className={css.bannerIcon} color={color} />
      <Text color={color}>{bannerMessage}</Text>
    </Container>
  )
}

export default TrialBanner
