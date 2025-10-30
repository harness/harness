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

import type React from 'react'

import type { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'
import type RbacButton from '@ar/__mocks__/components/RbacButton'
import type RbacMenuItem from '@ar/__mocks__/components/RbacMenuItem'
import type NGBreadcrumbs from '@ar/__mocks__/components/NGBreadcrumbs'
import type PageNotPublic from '@ar/__mocks__/components/PageNotPublic'
import type DependencyView from '@ar/__mocks__/components/DependencyView'
import type SecretFormInput from '@ar/__mocks__/components/SecretFormInput'
import type VulnerabilityView from '@ar/__mocks__/components/VulnerabilityView'
import type PolicySetFixedTypeSelector from '@ar/__mocks__/components/PolicySetFixedTypeSelector'
import type {
  ModalProvider,
  useConfirmationDialog,
  useDefaultPaginationProps,
  useModalHook,
  useQueryParams,
  useQueryParamsOptions,
  useUpdateQueryParams
} from '@ar/__mocks__/hooks'
import type { usePreferenceStore } from '@ar/__mocks__/contexts/PreferenceStoreContext'
import type { ARRouteDefinitionsReturn } from '@ar/routes/RouteDefinitions'
import type { Parent } from '@ar/common/types'
import type { LicenseStoreContextProps } from '@ar/common/LicenseTypes'
import type { GovernanceMetadata, useGovernanceMetaDataModal } from './__mocks__/hooks/useGovernanceMetaDataModal'

export interface Scope {
  accountId?: string
  orgIdentifier?: string
  projectIdentifier?: string
  space?: string
}

export interface PipelineExecutionPathProps {
  executionIdentifier: string
  pipelineIdentifier: string
  module: 'ci' | 'cd'
}

export interface ServiceDetailsPathProps {
  serviceId: string
}

export interface PermissionsRequest {
  resource: { resourceType: ResourceType; resourceIdentifier?: string }
  permissions: PermissionIdentifier[]
  [key: string]: unknown
}

export type FeatureFlagMap = Partial<Record<FeatureFlags, boolean>>

export interface AppstoreContext {
  updateAppStore: (value: Record<string, unknown>) => void
  featureFlags: FeatureFlagMap
}

export interface ParentContextObj {
  appStoreContext: React.Context<AppstoreContext>
  permissionsContext: React.Context<Record<string, unknown>>
  licenseStoreProvider: React.Context<LicenseStoreContextProps>
  tooltipContext?: React.Context<Record<string, unknown>>
  tokenContext?: React.Context<Record<string, unknown>>
}

export interface Components {
  RbacButton: typeof RbacButton
  NGBreadcrumbs: typeof NGBreadcrumbs
  RbacMenuItem: typeof RbacMenuItem
}

export interface Hooks {
  useDocumentTitle: (title: string | string[]) => { updateTitle: (newTitle: string | string[]) => void }
  useLogout: () => { forceLogout: (errorCode?: string) => void }
  usePermission: (permissionsRequest?: PermissionsRequest, deps?: Array<any>) => Array<boolean>
}

export interface CustomHooks {
  useQueryParams: typeof useQueryParams
  useUpdateQueryParams: typeof useUpdateQueryParams
  useQueryParamsOptions: typeof useQueryParamsOptions
  useDefaultPaginationProps: typeof useDefaultPaginationProps
  usePreferenceStore: typeof usePreferenceStore
  useModalHook: typeof useModalHook
  useConfirmationDialog: typeof useConfirmationDialog
  useGovernanceMetaDataModal: typeof useGovernanceMetaDataModal
}

export interface CustomComponents {
  ModalProvider: typeof ModalProvider
  SecretFormInput: typeof SecretFormInput
  VulnerabilityView: typeof VulnerabilityView
  DependencyView: typeof DependencyView
  PolicySetFixedTypeSelector: typeof PolicySetFixedTypeSelector
  PageNotPublic: typeof PageNotPublic
}

export interface GenerateTokenResponse {
  data: string
  metaData?: {
    governanceMetadata?: GovernanceMetadata
  }
}

export interface CustomUtils {
  generateToken: () => Promise<string | GenerateTokenResponse>
  getCustomHeaders: () => Record<string, string>
  getApiBaseUrl: (url: string) => string
  getRouteDefinitions?: (routeParams: Record<string, string>) => ARRouteDefinitionsReturn
  getRouteToPipelineExecutionView?: (params: Scope & PipelineExecutionPathProps) => string
  getRouteToServiceDetailsView?: (params: Scope & ServiceDetailsPathProps) => string
  routeToMode?: (params: Scope & { module: string }) => string
  routeToRegistryDetails: (params: Scope & { module: string; repositoryIdentifier: string }) => string
}

export interface MFEAppProps {
  renderUrl: string
  matchPath: string
  scope: Scope
  customScope: Record<string, string>
  on401: () => void
  children?: React.ReactNode
  NavComponent?: React.FC
  parentContextObj: ParentContextObj
  customHooks: CustomHooks
  components: Components
  customComponents: CustomComponents
  customUtils: CustomUtils
  hooks: Hooks
  parent: Parent
  routingId?: string
  isPublicAccessEnabledOnResources: boolean
  isCurrentSessionPublic: boolean
}

export enum FeatureFlags {
  HAR_TRIGGERS = 'HAR_TRIGGERS',
  HAR_CUSTOM_METADATA_ENABLED = 'HAR_CUSTOM_METADATA_ENABLED',
  HAR_ARTIFACT_QUARANTINE_ENABLED = 'HAR_ARTIFACT_QUARANTINE_ENABLED',
  HAR_ENABLE_UNTAGGED_IMAGES_SUPPORT = 'HAR_ENABLE_UNTAGGED_IMAGES_SUPPORT'
}
