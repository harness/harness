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
import moment from 'moment'

export const formatLastUpdated = (timestamp?: number): string => {
  if (!timestamp || typeof timestamp !== 'number' || isNaN(timestamp)) return '-'

  const momentDate = moment(timestamp)
  if (!momentDate.isValid()) return '-'

  return momentDate.format('D MMM YYYY, h:mm A')
}

export const getTimeAgo = (timestamp: number): string => {
  if (!timestamp || typeof timestamp !== 'number' || isNaN(timestamp)) return 'N/A'

  const momentDate = moment(timestamp)
  if (!momentDate.isValid()) return 'N/A'

  const now = moment()
  const diff = now.diff(momentDate, 'seconds')

  if (diff < 60) return 'just now'

  return momentDate.fromNow()
}

export const formatToISOString = (date?: Date | number): string => {
  if (!date) return ''
  return moment(date).toISOString()
}

export const isToday = (timestamp: number): boolean => {
  return moment(timestamp).isSame(moment(), 'day')
}
