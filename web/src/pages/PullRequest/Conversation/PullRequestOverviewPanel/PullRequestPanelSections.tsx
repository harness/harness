import React from 'react'
import { Layout } from '@harnessio/uicore'
import { PanelSectionOutletPosition } from 'pages/PullRequest/PullRequestUtils'

interface PullRequestPanelSectionsProps {
  outlets?: Partial<Record<PanelSectionOutletPosition, React.ReactNode>>
}

const PullRequestPanelSections = (props: PullRequestPanelSectionsProps) => {
  const { outlets = {} } = props
  return (
    <Layout.Vertical>
      {outlets[PanelSectionOutletPosition.CHANGES]}
      {outlets[PanelSectionOutletPosition.COMMENTS]}
      {outlets[PanelSectionOutletPosition.CHECKS]}
      {outlets[PanelSectionOutletPosition.MERGEABILITY]}
    </Layout.Vertical>
  )
}

export default PullRequestPanelSections
