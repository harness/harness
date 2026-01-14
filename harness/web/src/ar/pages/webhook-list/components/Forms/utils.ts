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

import produce from 'immer'
import { get, set } from 'lodash-es'
import type { WebhookRequest, Webhook } from '@harnessio/react-har-service-client'

import type { Scope } from '@ar/MFEAppTypes'
import { Parent } from '@ar/common/types'
import {
  getSecretScopeDetailsByIdentifier,
  getSecretSpacePath
} from '@ar/pages/upstream-proxy-details/components/Forms/utils'

import type { WebhookRequestUI } from './types'

function convertSecretInputToFormFields(
  formData: WebhookRequestUI,
  secretField: keyof WebhookRequestUI,
  secretSpacePathField: keyof WebhookRequestUI,
  scope?: Scope
) {
  const secret = get(formData, secretField)
  set(formData, secretSpacePathField, getSecretSpacePath(get(secret, 'referenceString', ''), scope))
  set(formData, secretField, get(secret, 'identifier'))
}

export function transformFormValuesToSubmitValues(
  formValues: WebhookRequestUI,
  parent: Parent,
  scope: Scope
): WebhookRequest {
  return produce(formValues, draft => {
    if (draft.triggerType === 'all') {
      draft.triggers = []
    }
    if (draft.extraHeaders?.length) {
      draft.extraHeaders = draft.extraHeaders.filter(each => !!each.key && !!each.value)
    }
    if (parent === Parent.Enterprise) {
      convertSecretInputToFormFields(draft, 'secretIdentifier', 'secretSpacePath', scope)
    }
    set(draft, 'triggerType', undefined)
    return draft
  })
}

function convertFormFieldsToSecreteInput(formData: Webhook, secretField: string, secretSpacePathField: string) {
  const secretIdentifier = get(formData, secretField, '')
  const secretSpacePath = get(formData, secretSpacePathField, '')
  if (secretIdentifier) {
    set(formData, secretField, getSecretScopeDetailsByIdentifier(secretIdentifier, secretSpacePath))
  } else {
    set(formData, secretField, undefined)
  }
}

export function transformWebhookDataToFormValues(data: Webhook, parent: Parent): WebhookRequestUI {
  return produce(data, draft => {
    if (draft.triggers?.length) {
      set(draft, 'triggerType', 'custom')
    } else {
      set(draft, 'triggerType', 'all')
    }
    if (!draft.extraHeaders?.length) {
      set(draft, 'extraHeaders', [{ key: '', value: '' }])
    }
    if (parent === Parent.Enterprise) {
      convertFormFieldsToSecreteInput(draft, 'secretIdentifier', 'secretSpacePath')
    }
    return draft
  }) as WebhookRequestUI
}
