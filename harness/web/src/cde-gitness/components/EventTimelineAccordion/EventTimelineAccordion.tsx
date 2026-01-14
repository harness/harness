/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react'
import { defaultTo } from 'lodash-es'
import { Accordion } from '@harnessio/uicore'
import type { TypesGitspaceEventResponse } from 'cde-gitness/services'
import EventTimelineSummary from '../EventTimelineSummary/EventTimelineSummary'
import EventTimeline from '../EventTimeline/EventTimeline'
import parentCss from 'cde-gitness/pages/GitspaceDetails/GitspaceDetails.module.scss'

const EventTimelineAccordion = ({ data }: { data: TypesGitspaceEventResponse[] | null; polling?: boolean }) => {
  const sortedData = data?.sort((a, b) => defaultTo(a?.timestamp, 0) - defaultTo(b?.timestamp, 0))
  const latestEvent = sortedData?.[sortedData?.length - 1] || { message: '', timestamp: 0 }
  return (
    <Accordion activeId="eventsCard">
      <Accordion.Panel
        shouldRender
        id="eventsCard"
        details={<EventTimeline data={sortedData} />}
        summary={<EventTimelineSummary message={latestEvent.message} timestamp={latestEvent.timestamp} />}
        className={parentCss.accordionnCustomSummary}
      />
    </Accordion>
  )
}

export default EventTimelineAccordion
