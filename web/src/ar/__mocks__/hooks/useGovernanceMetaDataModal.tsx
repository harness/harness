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

export type GovernanceMetadata = Record<string, unknown>

export interface UseConnectorGovernanceModalPayload {
  conditionallyOpenGovernanceErrorModal(
    governanceMetadata?: GovernanceMetadata,
    onModalCloseWhenNoErrorInGovernanceData?: () => void
  ): void
}
export interface UseGovernanceModalProps {
  // Will consider warnings as error  and does not process onSuccess method
  considerWarningAsError: boolean
  warningHeaderMsg: string
  errorHeaderMsg: string
  skipGovernanceCheck?: boolean // will skip warning modals
}

export function useGovernanceMetaDataModal(_props: UseGovernanceModalProps): UseConnectorGovernanceModalPayload {
  return {
    conditionallyOpenGovernanceErrorModal: (_governanceMetadata?: GovernanceMetadata, callback?: () => void) => {
      callback?.()
    }
  }
}
