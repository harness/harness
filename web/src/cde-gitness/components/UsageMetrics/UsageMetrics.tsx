import React from 'react'
import cx from 'classnames'
import { Button, ButtonVariation, Container, Layout, Text } from '@harnessio/uicore'
import { InfoEmpty } from 'iconoir-react'
import { Color } from '@harnessio/design-system'
import { Classes, PopoverPosition } from '@blueprintjs/core'
import { String, useStrings } from 'framework/strings'
import { TypesUsage, useGetUsageForAccount } from 'services/cde'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'
import { getErrorMessage } from 'utils/Utils'
import BeatPlanIcon from './assets/BetaPlan.svg?url'
import MinimalDonutChart from '../MinimalDonut/MinimalDonut'
import css from './UsageMetrics.module.scss'

export enum UsageTypeEnum {
  LOW = 'Low',
  MEDIUM = 'Medium',
  HIGH = 'High'
}

export type UsageType = 'Low' | 'Medium' | 'High'

const PlanAndRenewSection = () => {
  const { getString } = useStrings()
  return (
    <Layout.Horizontal spacing="medium" flex={{ alignItems: 'center' }}>
      <img src={BeatPlanIcon} height="20px" />
      <Text font={{ size: 'normal' }}>{getString('cde.renewsEveryMonth', { days: 30 })}</Text>
      <Button
        variation={ButtonVariation.ICON}
        tooltip={
          <Container width={300} padding="medium">
            <Layout.Vertical spacing="small">
              <Text color={Color.WHITE} font={{ size: 'small', weight: 'semi-bold' }}>
                {getString('cde.minuteUsage')}
              </Text>
              <Text font="small">{getString('cde.renewTooltip.line1')}</Text>
              <Text font="small">{getString('cde.renewTooltip.line2')}</Text>
              <Text font="small">{getString('cde.renewTooltip.haveQuestion')}</Text>
              <Text
                color={Color.PRIMARY_7}
                font="small"
                style={{ cursor: 'pointer' }}
                onClick={() => {
                  // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                  // @ts-ignore
                  window?.document?.querySelector('#sidenav-footer [__type="SIDENAV_LINK"]')?.click()
                }}>
                {getString('cde.renewTooltip.contactUs')}
              </Text>
            </Layout.Vertical>
          </Container>
        }
        tooltipProps={{ isDark: true, position: PopoverPosition.AUTO }}>
        <InfoEmpty height={12} color="#0278D5" fill="white" />
      </Button>
    </Layout.Horizontal>
  )
}

export interface UsageInfo {
  colors: [string, string]
  degree: number
  background: string
}

function getUsageChartMetadata({ total_mins = 0, used_mins = 0 }: TypesUsage): UsageInfo {
  const usagePercent = (used_mins / total_mins) * 100
  const degree = (used_mins / total_mins) * 360

  let colors: [string, string]
  let background: string

  if (usagePercent <= 33) {
    colors = ['#42AB45', '#FFFFFF'] // Green and White
    background = '#299B2C' // Darker green (hardcoded)
  } else if (usagePercent > 33 && usagePercent <= 66) {
    colors = ['#FDCC35', '#FFFFFF'] // Orange and White
    background = '#E19C02' // Darker orange (hardcoded)
  } else if (usagePercent > 66) {
    colors = ['#EE5F54', '#FFFFFF'] // Red and White
    background = '#C31F17' // Darker red (hardcoded)
  } else {
    colors = ['#808080', '#FFFFFF'] // Grey and White
    background = '#505050' // Darker grey (hardcoded)
  }

  return {
    colors: colors,
    degree: degree,
    background
  }
}

const UsageChart = ({ usage }: { usage: TypesUsage }) => {
  const { total_mins = 0, used_mins = 0 } = usage || {}
  const remainingOfTotal = total_mins - used_mins
  const { colors, degree, background } = getUsageChartMetadata({ total_mins: total_mins, used_mins })
  return (
    <Layout.Horizontal height="32px" flex={{ alignItems: 'center', justifyContent: 'center' }} spacing="small">
      <Layout.Horizontal
        padding="small"
        flex={{ alignItems: 'center', justifyContent: 'center' }}
        spacing="small"
        style={{ backgroundColor: background }}
        className={css.chartContainer}>
        <MinimalDonutChart size={20} colors={colors} innerRadius={70} degree={degree} background={background} />
        <Text font={{ weight: 'semi-bold' }} color={Color.WHITE}>
          {remainingOfTotal} mins
        </Text>
      </Layout.Horizontal>
      <Text font={{ size: 'normal' }}>
        <String stringID="cde.remainingOfTotal" vars={{ total: total_mins }} useRichText />
      </Text>
    </Layout.Horizontal>
  )
}

const UsageMetrics = () => {
  const { accountIdentifier = '' } = useGetCDEAPIParams()
  const { data, loading, error } = useGetUsageForAccount({
    accountIdentifier
  })
  return (
    <Container className={cx(css.main, { [Classes.SKELETON]: loading })}>
      {data ? (
        <Layout.Horizontal spacing="medium" flex className={css.innerContainer}>
          <UsageChart usage={data} />
          <Container height="28px" background={Color.GREY_200} width="1px" />
          <PlanAndRenewSection />
        </Layout.Horizontal>
      ) : error ? (
        <Layout.Horizontal spacing="medium" flex className={cx(css.errorAndNoDataContainer)}>
          <Layout.Horizontal spacing="medium" flex={{ alignItems: 'center' }}>
            <Text font={'small'} color={Color.ERROR}>
              {getErrorMessage(error)}
            </Text>
          </Layout.Horizontal>
        </Layout.Horizontal>
      ) : (
        <Layout.Horizontal spacing="medium" flex className={cx(css.errorAndNoDataContainer)}>
          <Layout.Horizontal spacing="medium" flex={{ alignItems: 'center' }}>
            <Text font={'small'} color={Color.GREY_400}>
              No Usage Data Found
            </Text>
          </Layout.Horizontal>
        </Layout.Horizontal>
      )}
    </Container>
  )
}

export default UsageMetrics
