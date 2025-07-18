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
import * as Yup from 'yup'
import { compact, get, isEmpty, set } from 'lodash-es'
import type { AccessKeySecretKey, Anonymous, UserPassword } from '@harnessio/react-har-service-client'

import { Parent } from '@ar/common/types'
import { URL_REGEX } from '@ar/constants'
import type { Scope } from '@ar/MFEAppTypes'
import type { StringKeys } from '@ar/frameworks/strings'
import { SecretValueType } from '@ar/components/MultiTypeSecretInput/MultiTypeSecretInput'

import {
  UpstreamRepositoryURLInputSource,
  UpstreamProxyAuthenticationMode,
  type UpstreamRegistryRequest
} from '../../types'

export function getSecretSpacePath(referenceString: string, scope?: Scope) {
  if (!scope) return referenceString
  if (referenceString.startsWith('account.')) {
    return compact([scope.accountId]).join('/')
  }
  if (referenceString.startsWith('org.')) {
    return compact([scope.accountId, scope.orgIdentifier]).join('/')
  }
  return compact([scope.accountId, scope.orgIdentifier, scope.projectIdentifier]).join('/')
}

export function getReferenceStringFromSecretSpacePath(identifier: string, secretSpacePath: string) {
  const [accountId, orgIdentifier, projectIdentifier] = secretSpacePath.split('/')
  if (projectIdentifier) return identifier
  if (orgIdentifier) return `org.${identifier}`
  if (accountId) return `account.${identifier}`
  return identifier
}

function convertSecretInputToFormFields(
  formData: UpstreamRegistryRequest,
  secretField: string,
  secretSpacePathField: string,
  scope?: Scope
) {
  const password = get(formData, secretField)
  set(formData, secretSpacePathField, getSecretSpacePath(get(password, 'referenceString', ''), scope))
  set(formData, secretField, get(password, 'identifier'))
}

function convertMultiTypeSecretInputToFormFields(
  formData: UpstreamRegistryRequest,
  typeField: string,
  textField: string,
  secretField: string,
  secretSpacePathField: string,
  scope?: Scope
) {
  const accessKeyType = get(formData, typeField)
  if (accessKeyType === SecretValueType.TEXT) {
    const value = get(formData, textField, '')
    set(formData, textField, value)
    set(formData, typeField, undefined)
    set(formData, secretField, undefined)
    set(formData, secretSpacePathField, undefined)
  } else {
    const accessKeySecret = get(formData, secretField)
    set(formData, secretSpacePathField, getSecretSpacePath(get(accessKeySecret, 'referenceString', ''), scope))
    set(formData, secretField, get(accessKeySecret, 'identifier'))
    set(formData, typeField, undefined)
    set(formData, textField, undefined)
  }
}

export function getFormattedFormDataForAuthType(
  values: UpstreamRegistryRequest,
  parent?: Parent,
  scope?: Scope
): UpstreamRegistryRequest {
  return produce(values, (draft: UpstreamRegistryRequest) => {
    if (draft.config.authType === UpstreamProxyAuthenticationMode.USER_NAME_AND_PASSWORD) {
      set(draft, 'config.auth.authType', draft.config.authType)
      if (parent === Parent.Enterprise) {
        convertSecretInputToFormFields(draft, 'config.auth.secretIdentifier', 'config.auth.secretSpacePath', scope)
      }
    } else if (draft.config.authType === UpstreamProxyAuthenticationMode.ACCESS_KEY_AND_SECRET_KEY) {
      set(draft, 'config.auth.authType', draft.config.authType)
      if (parent === Parent.Enterprise) {
        convertSecretInputToFormFields(
          draft,
          'config.auth.secretKeyIdentifier',
          'config.auth.secretKeySpacePath',
          scope
        )
        convertMultiTypeSecretInputToFormFields(
          draft,
          'config.auth.accessKeyType',
          'config.auth.accessKey',
          'config.auth.accessKeySecretIdentifier',
          'config.auth.accessKeySecretSpacePath',
          scope
        )
      }
    } else if (draft.config.authType === UpstreamProxyAuthenticationMode.ANONYMOUS) {
      set(draft, 'config.auth', null)
    }
    if (
      [
        UpstreamRepositoryURLInputSource.Dockerhub,
        UpstreamRepositoryURLInputSource.MavenCentral,
        UpstreamRepositoryURLInputSource.Crates,
        UpstreamRepositoryURLInputSource.NpmJS,
        UpstreamRepositoryURLInputSource.NugetOrg,
        UpstreamRepositoryURLInputSource.PyPi,
        UpstreamRepositoryURLInputSource.GoProxy
      ].includes(draft.config.source as UpstreamRepositoryURLInputSource)
    ) {
      set(draft, 'config.url', '')
    }
  })
}

export function getSecretScopeDetailsByIdentifier(identifier: string, secretSpacePath: string) {
  const referenceString = getReferenceStringFromSecretSpacePath(identifier, secretSpacePath)
  const [, orgIdentifier, projectIdentifier] = secretSpacePath.split('/')
  return {
    identifier: identifier,
    name: identifier,
    referenceString,
    orgIdentifier,
    projectIdentifier
  }
}

function convertFormFieldsToSecreteInput(
  formData: UpstreamRegistryRequest,
  secretField: string,
  secretSpacePathField: string
) {
  const secretIdentifier = get(formData, secretField, '')
  const secretSpacePath = get(formData, secretSpacePathField, '')
  set(formData, secretField, getSecretScopeDetailsByIdentifier(secretIdentifier, secretSpacePath))
}

function convertFormFieldsToMultiTypeSecretInput(
  formData: UpstreamRegistryRequest,
  typeField: string,
  formField: string,
  secretIdentifierField: string,
  secretSpacePathField: string
) {
  const value = get(formData, formField)
  if (value) {
    set(formData, formField, value)
    set(formData, typeField, SecretValueType.TEXT)
    set(formData, secretIdentifierField, undefined)
    set(formData, secretSpacePathField, undefined)
  } else {
    const secretIdentifierValue = get(formData, secretIdentifierField, '')
    const secretSpacePathValue = get(formData, secretSpacePathField, '')
    set(formData, typeField, SecretValueType.ENCRYPTED)
    set(formData, secretIdentifierField, getSecretScopeDetailsByIdentifier(secretIdentifierValue, secretSpacePathValue))
    set(formData, formField, undefined)
  }
}

export function getFormattedInitialValuesForAuthType(values: UpstreamRegistryRequest, parent?: Parent) {
  return produce(values, (draft: UpstreamRegistryRequest) => {
    if (draft.config.authType === UpstreamProxyAuthenticationMode.USER_NAME_AND_PASSWORD) {
      if (parent === Parent.Enterprise) {
        convertFormFieldsToSecreteInput(draft, 'config.auth.secretIdentifier', 'config.auth.secretSpacePath')
      }
    }
    if (draft.config.authType === UpstreamProxyAuthenticationMode.ACCESS_KEY_AND_SECRET_KEY) {
      if (parent === Parent.Enterprise) {
        convertFormFieldsToSecreteInput(draft, 'config.auth.secretKeyIdentifier', 'config.auth.secretKeySpacePath')
        convertFormFieldsToMultiTypeSecretInput(
          draft,
          'config.auth.accessKeyType',
          'config.auth.accessKey',
          'config.auth.accessKeySecretIdentifier',
          'config.auth.accessKeySecretSpacePath'
        )
      } else {
        const accessKeyType = isEmpty(get(draft, 'config.auth.accessKey'))
          ? SecretValueType.ENCRYPTED
          : SecretValueType.TEXT
        set(draft, 'config.auth.accessKeyType', accessKeyType)
      }
    }
  })
}

export function getValidationSchemaForUpstreamForm(getString: (key: StringKeys, vars?: Record<string, any>) => string) {
  return {
    config: Yup.object().shape({
      authType: Yup.string()
        .required()
        .oneOf([
          UpstreamProxyAuthenticationMode.ANONYMOUS,
          UpstreamProxyAuthenticationMode.USER_NAME_AND_PASSWORD,
          UpstreamProxyAuthenticationMode.ACCESS_KEY_AND_SECRET_KEY
        ]),
      auth: Yup.object()
        .when(['authType'], {
          is: (authType: UpstreamProxyAuthenticationMode) =>
            authType === UpstreamProxyAuthenticationMode.USER_NAME_AND_PASSWORD,
          then: (schema: Yup.ObjectSchema<UserPassword | Anonymous>) =>
            schema.shape({
              userName: Yup.string().trim().required(getString('validationMessages.userNameRequired')),
              secretIdentifier: Yup.string().trim().required(getString('validationMessages.passwordRequired'))
            }),
          otherwise: Yup.object().optional().nullable()
        })
        .when(['authType'], {
          is: (authType: UpstreamProxyAuthenticationMode) =>
            authType === UpstreamProxyAuthenticationMode.ACCESS_KEY_AND_SECRET_KEY,
          then: (schema: Yup.ObjectSchema<AccessKeySecretKey | Anonymous>) =>
            schema.shape({
              accessKey: Yup.string()
                .trim()
                .test('access-key-validation', getString('validationMessages.accessKeyRequired'), function (value) {
                  if (this.parent.accessKeyType === SecretValueType.TEXT) {
                    return !isEmpty(value)
                  }
                  return true
                }),
              accessKeySecretIdentifier: Yup.string()
                .trim()
                .test(
                  'access-key-secret-validation',
                  getString('validationMessages.accessKeyRequired'),
                  function (value) {
                    if (this.parent.accessKeyType === SecretValueType.ENCRYPTED) {
                      return !isEmpty(value)
                    }
                    return true
                  }
                ),
              secretKeyIdentifier: Yup.string().trim().required(getString('validationMessages.secretKeyRequired'))
            }),
          otherwise: Yup.object().optional().nullable()
        })
        .nullable(),
      url: Yup.string().when(['source'], {
        is: (source: UpstreamRepositoryURLInputSource) =>
          [UpstreamRepositoryURLInputSource.Custom, UpstreamRepositoryURLInputSource.AwsEcr].includes(source),
        then: (schema: Yup.StringSchema) =>
          schema
            .trim()
            .required(getString('validationMessages.urlRequired'))
            .matches(URL_REGEX, getString('validationMessages.urlPattern')),
        otherwise: (schema: Yup.StringSchema) => schema.trim().notRequired()
      })
    }),
    cleanupPolicy: Yup.array()
      .of(
        Yup.object().shape({
          name: Yup.string().trim().required(getString('validationMessages.cleanupPolicy.nameRequired')),
          expireDays: Yup.number()
            .required(getString('validationMessages.cleanupPolicy.expireDaysRequired'))
            .positive(getString('validationMessages.cleanupPolicy.positiveExpireDays'))
        })
      )
      .optional()
      .nullable()
  }
}
