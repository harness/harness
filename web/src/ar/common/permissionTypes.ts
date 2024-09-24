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

export enum PermissionIdentifier {
  VIEW_ARTIFACT_REGISTRY = 'artifact_artregistry_view',
  EDIT_ARTIFACT_REGISTRY = 'artifact_artregistry_edit',
  DELETE_ARTIFACT_REGISTRY = 'artifact_artregistry_delete',
  DOWNLOAD_ARTIFACT = 'artifact_artregistry_downloadartifact',
  UPLOAD_ARTIFACT = 'artifact_artregistry_uploadartifact',
  DELETE_ARTIFACT = 'artifact_artregistry_deleteartifact'
}

export enum ResourceType {
  ARTIFACT_REGISTRY = 'ARTIFACT_REGISTRY'
}
