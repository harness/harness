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

import { Parent } from '@ar/common/types'

import type { UpstreamRegistryRequest } from '@ar/pages/upstream-proxy-details/types'
import {
  getFormattedFormDataForCleanupPolicy,
  getFormattedIntialValuesForCleanupPolicy
} from '@ar/components/CleanupPolicyList/utils'

import {
  getFormattedFormDataForAuthType,
  getFormattedInitialValuesForAuthType,
  getReferenceStringFromSecretSpacePath,
  getSecretSpacePath
} from '../utils'

const mockUpstreamProxyFormData = {
  packageType: 'DOCKER',
  identifier: 'test1',
  isPublic: false,
  config: {
    type: 'UPSTREAM',
    source: 'Dockerhub',
    url: '',
    authType: 'UserPassword',
    auth: {
      userName: 'admin',
      secretIdentifier: {
        name: 'prod-shiv-pass',
        identifier: 'prod-shiv-pass',
        orgIdentifier: 'org',
        projectIdentifier: 'project',
        type: 'SecretText',
        referenceString: 'prod-shiv-pass'
      }
    }
  },
  cleanupPolicy: [],
  scanners: []
} as UpstreamRegistryRequest

const mockUpstreamProxyFormDataForAwsEcr = {
  blockedPattern: ['test3', 'test4'],
  isPublic: false,
  config: {
    auth: {
      accessKey: 'accessKey',
      accessKeySecretIdentifier: undefined,
      accessKeySecretSpacePath: undefined,
      accessKeyType: 'TEXT',
      authType: 'AccessKeySecretKey',
      secretKeyIdentifier: {
        identifier: 'secretKey',
        name: 'secretKey',
        orgIdentifier: undefined,
        projectIdentifier: undefined,
        referenceString: 'secretKey'
      }
    },
    authType: 'AccessKeySecretKey',
    source: 'AwsEcr',
    type: 'UPSTREAM',
    url: 'https://aws.ecr.com'
  },
  createdAt: '1738516362995',
  description: 'test description',
  identifier: 'docker-up-repo',
  labels: ['label1', 'label2', 'label3', 'label4'],
  modifiedAt: '1738516362995',
  packageType: 'DOCKER',
  url: ''
} as UpstreamRegistryRequest

const mockUpstreamProxyData = {
  cleanupPolicy: [],
  isPublic: false,
  config: {
    auth: {
      authType: 'UserPassword',
      secretIdentifier: 'prod-shiv-pass',
      secretSpacePath: 'acc/org/project',
      userName: 'admin'
    },
    authType: 'UserPassword',
    source: 'Dockerhub',
    type: 'UPSTREAM',
    url: ''
  },
  identifier: 'test1',
  packageType: 'DOCKER',
  scanners: []
} as UpstreamRegistryRequest

const mockUpstreamProxyDataForAwsEcr = {
  blockedPattern: ['test3', 'test4'],
  isPublic: false,
  config: {
    auth: {
      accessKey: 'accessKey',
      accessKeyType: 'TEXT',
      authType: 'AccessKeySecretKey',
      secretKeyIdentifier: 'secretKey'
    },
    authType: 'AccessKeySecretKey',
    source: 'AwsEcr',
    type: 'UPSTREAM',
    url: 'https://aws.ecr.com'
  },
  createdAt: '1738516362995',
  description: 'test description',
  identifier: 'docker-up-repo',
  labels: ['label1', 'label2', 'label3', 'label4'],
  modifiedAt: '1738516362995',
  packageType: 'DOCKER',
  url: ''
} as UpstreamRegistryRequest

const scope = {
  accountId: 'acc',
  orgIdentifier: 'org',
  projectIdentifier: 'project',
  space: 'acc/org/project'
}

describe('Verify Upstream Proxy Form utils', () => {
  test('verify getFormattedFormDataForAuthType', () => {
    const updatedValue = getFormattedFormDataForAuthType(mockUpstreamProxyFormData, Parent.Enterprise, scope)
    expect(updatedValue).toEqual({
      cleanupPolicy: [],
      config: {
        auth: {
          authType: 'UserPassword',
          secretIdentifier: 'prod-shiv-pass',
          secretSpacePath: 'acc/org/project',
          userName: 'admin'
        },
        authType: 'UserPassword',
        source: 'Dockerhub',
        type: 'UPSTREAM',
        url: ''
      },
      identifier: 'test1',
      packageType: 'DOCKER',
      scanners: [],
      isPublic: false
    })
  })

  test('verify getFormattedFormDataForAuthType for aws ecr: Text', () => {
    const updatedValue = getFormattedFormDataForAuthType(mockUpstreamProxyFormDataForAwsEcr, Parent.Enterprise, scope)
    expect(updatedValue).toEqual({
      blockedPattern: ['test3', 'test4'],
      config: {
        auth: {
          accessKey: 'accessKey',
          accessKeySecretIdentifier: undefined,
          accessKeySecretSpacePath: undefined,
          accessKeyType: undefined,
          authType: 'AccessKeySecretKey',
          secretKeyIdentifier: 'secretKey',
          secretKeySpacePath: 'acc/org/project'
        },
        authType: 'AccessKeySecretKey',
        source: 'AwsEcr',
        type: 'UPSTREAM',
        url: 'https://aws.ecr.com'
      },
      createdAt: '1738516362995',
      description: 'test description',
      identifier: 'docker-up-repo',
      labels: ['label1', 'label2', 'label3', 'label4'],
      modifiedAt: '1738516362995',
      packageType: 'DOCKER',
      url: '',
      isPublic: false
    })
  })

  test('verify getFormattedFormDataForAuthType for aws ecr: Secret', () => {
    const updatedValue = getFormattedFormDataForAuthType(
      {
        ...mockUpstreamProxyFormDataForAwsEcr,
        config: {
          ...mockUpstreamProxyFormDataForAwsEcr.config,
          auth: {
            accessKey: undefined,
            accessKeySecretIdentifier: {
              identifier: 'accessKey',
              name: 'accessKey',
              orgIdentifier: undefined,
              projectIdentifier: undefined,
              referenceString: 'accessKey'
            },
            accessKeyType: 'ENCRYPTED',
            authType: 'AccessKeySecretKey',
            secretKeyIdentifier: {
              identifier: 'secretKey',
              name: 'secretKey',
              orgIdentifier: undefined,
              projectIdentifier: undefined,
              referenceString: 'secretKey'
            }
          }
        }
      },
      Parent.Enterprise,
      scope
    )
    expect(updatedValue).toEqual({
      blockedPattern: ['test3', 'test4'],
      config: {
        auth: {
          accessKey: undefined,
          accessKeySecretIdentifier: 'accessKey',
          accessKeySecretSpacePath: 'acc/org/project',
          accessKeyType: undefined,
          authType: 'AccessKeySecretKey',
          secretKeyIdentifier: 'secretKey',
          secretKeySpacePath: 'acc/org/project'
        },
        authType: 'AccessKeySecretKey',
        source: 'AwsEcr',
        type: 'UPSTREAM',
        url: 'https://aws.ecr.com'
      },
      createdAt: '1738516362995',
      description: 'test description',
      identifier: 'docker-up-repo',
      labels: ['label1', 'label2', 'label3', 'label4'],
      modifiedAt: '1738516362995',
      packageType: 'DOCKER',
      url: '',
      isPublic: false
    })
  })

  test('verify getFormattedFormDataForCleanupPolicy', () => {
    const updatedValue = getFormattedFormDataForCleanupPolicy(mockUpstreamProxyFormData)
    expect(updatedValue).toEqual({
      cleanupPolicy: [],
      config: {
        auth: {
          secretIdentifier: {
            identifier: 'prod-shiv-pass',
            name: 'prod-shiv-pass',
            orgIdentifier: 'org',
            projectIdentifier: 'project',
            referenceString: 'prod-shiv-pass',
            type: 'SecretText'
          },
          userName: 'admin'
        },
        authType: 'UserPassword',
        source: 'Dockerhub',
        type: 'UPSTREAM',
        url: ''
      },
      identifier: 'test1',
      packageType: 'DOCKER',
      scanners: [],
      isPublic: false
    })
  })

  test('verify getFormattedInitialValuesForAuthType', () => {
    const updatedValue = getFormattedInitialValuesForAuthType(mockUpstreamProxyData, Parent.Enterprise)
    expect(updatedValue).toEqual({
      cleanupPolicy: [],
      config: {
        auth: {
          authType: 'UserPassword',
          secretIdentifier: {
            identifier: 'prod-shiv-pass',
            name: 'prod-shiv-pass',
            orgIdentifier: 'org',
            projectIdentifier: 'project',
            referenceString: 'prod-shiv-pass'
          },
          secretSpacePath: 'acc/org/project',
          userName: 'admin'
        },
        authType: 'UserPassword',
        source: 'Dockerhub',
        type: 'UPSTREAM',
        url: ''
      },
      identifier: 'test1',
      packageType: 'DOCKER',
      scanners: [],
      isPublic: false
    })
  })

  test('verify getFormattedInitialValuesForAuthType for aws ecr: Text', () => {
    const updatedValue = getFormattedInitialValuesForAuthType(mockUpstreamProxyDataForAwsEcr, Parent.Enterprise)
    expect(updatedValue).toEqual({
      blockedPattern: ['test3', 'test4'],
      config: {
        auth: {
          accessKey: 'accessKey',
          accessKeySecretIdentifier: undefined,
          accessKeySecretSpacePath: undefined,
          accessKeyType: 'TEXT',
          authType: 'AccessKeySecretKey',
          secretKeyIdentifier: {
            identifier: 'secretKey',
            name: 'secretKey',
            orgIdentifier: undefined,
            projectIdentifier: undefined,
            referenceString: 'secretKey'
          }
        },
        authType: 'AccessKeySecretKey',
        source: 'AwsEcr',
        type: 'UPSTREAM',
        url: 'https://aws.ecr.com'
      },
      createdAt: '1738516362995',
      description: 'test description',
      identifier: 'docker-up-repo',
      labels: ['label1', 'label2', 'label3', 'label4'],
      modifiedAt: '1738516362995',
      packageType: 'DOCKER',
      url: '',
      isPublic: false
    })
  })

  test('verify getFormattedInitialValuesForAuthType for aws ecr: Secret', () => {
    const updatedValue = getFormattedInitialValuesForAuthType(
      {
        ...mockUpstreamProxyDataForAwsEcr,
        config: {
          ...mockUpstreamProxyDataForAwsEcr.config,
          auth: {
            accessKeySecretIdentifier: 'accessKey',
            accessKeyType: 'ENCRYPTED',
            authType: 'AccessKeySecretKey',
            secretKeyIdentifier: 'secretKey'
          }
        }
      },
      Parent.Enterprise
    )
    expect(updatedValue).toEqual({
      blockedPattern: ['test3', 'test4'],
      config: {
        auth: {
          accessKey: undefined,
          accessKeySecretIdentifier: {
            identifier: 'accessKey',
            name: 'accessKey',
            orgIdentifier: undefined,
            projectIdentifier: undefined,
            referenceString: 'accessKey'
          },
          accessKeyType: 'ENCRYPTED',
          authType: 'AccessKeySecretKey',
          secretKeyIdentifier: {
            identifier: 'secretKey',
            name: 'secretKey',
            orgIdentifier: undefined,
            projectIdentifier: undefined,
            referenceString: 'secretKey'
          }
        },
        authType: 'AccessKeySecretKey',
        source: 'AwsEcr',
        type: 'UPSTREAM',
        url: 'https://aws.ecr.com'
      },
      createdAt: '1738516362995',
      description: 'test description',
      identifier: 'docker-up-repo',
      labels: ['label1', 'label2', 'label3', 'label4'],
      modifiedAt: '1738516362995',
      packageType: 'DOCKER',
      url: '',
      isPublic: false
    })
  })

  test('verify getFormattedIntialValuesForCleanupPolicy', () => {
    const updatedValue = getFormattedIntialValuesForCleanupPolicy(mockUpstreamProxyData)
    expect(updatedValue).toEqual({
      cleanupPolicy: [],
      config: {
        auth: {
          authType: 'UserPassword',
          secretIdentifier: 'prod-shiv-pass',
          secretSpacePath: 'acc/org/project',
          userName: 'admin'
        },
        authType: 'UserPassword',
        source: 'Dockerhub',
        type: 'UPSTREAM',
        url: ''
      },
      identifier: 'test1',
      packageType: 'DOCKER',
      scanners: [],
      isPublic: false
    })
  })

  test('verify getSecretSpacePath', () => {
    let response = getSecretSpacePath('dummy')
    expect(response).toEqual(response)
    response = getSecretSpacePath('account.dummy', scope)
    expect(response).toEqual('acc')
    response = getSecretSpacePath('org.dummy', scope)
    expect(response).toEqual('acc/org')
    response = getSecretSpacePath('dummy', scope)
    expect(response).toEqual('acc/org/project')
  })

  test('verify getReferenceStringFromSecretSpacePath', () => {
    let response = getReferenceStringFromSecretSpacePath('dummy', 'acc/org/project')
    expect(response).toEqual('dummy')
    response = getReferenceStringFromSecretSpacePath('dummy', 'acc/org')
    expect(response).toEqual('org.dummy')
    response = getReferenceStringFromSecretSpacePath('dummy', 'acc')
    expect(response).toEqual('account.dummy')
    response = getReferenceStringFromSecretSpacePath('dummy', '')
    expect(response).toEqual('dummy')
  })
})
