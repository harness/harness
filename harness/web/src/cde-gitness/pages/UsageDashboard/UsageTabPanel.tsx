import React from 'react'
import { Container, Text, Page } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import css from './UsageDashboardPage.module.scss'

const UsageTabPanel: React.FC = () => {
  const { getString } = useStrings()

  return (
    <Page.Body>
      <Container className={css.usageTab}>
        <Container className={css.dashboardCard}>
          <Text font={{ variation: FontVariation.H3 }}>{getString('cde.usageDashboard.comingSoon')}</Text>
        </Container>
      </Container>
    </Page.Body>
  )
}

export default UsageTabPanel
