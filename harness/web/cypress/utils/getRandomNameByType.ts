/*
 * Copyright 2024 Harness Inc. All rights reserved.
 * Use of this source code is governed by the PolyForm Shield 1.0.0 license
 * that can be found in the licenses directory at the root of this repository, also available at
 * https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
 */

import { getIdentifierFromName } from './getIdentifierFromName'
import { AUTOMATION_PROJECT_PREFIX, EntityType } from './types'

const hashID = (): string => crypto.randomUUID()

export const getRandomNameByType = (
  type: EntityType,
  seperator = '_',
  projectPrefix: AUTOMATION_PROJECT_PREFIX = 'cypress'
): string => {
  const randomName = `${projectPrefix}${seperator}${type}${seperator}${hashID()}`
  return getIdentifierFromName(randomName)
}
