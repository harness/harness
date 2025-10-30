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

export interface ARProps {
  space: string
}

export const pathProps: Readonly<Required<ARProps>> = {
  space: ':space*'
}

export interface ARArtifactsProps extends ARProps {
  repositoryIdentifier: string
}

export interface ARRoutes {
  toAR: (args: ARProps) => string
  toARArtifacts: (args: ARArtifactsProps) => string
  toARRepositoryWebhooks: (args: ARArtifactsProps) => string
}

export const routes: ARRoutes = {
  toAR: ({ space }) => `/spaces/${space}/registries`,
  toARArtifacts: params => `/spaces/${params.space}/registries/${params?.repositoryIdentifier}/packages`,
  toARRepositoryWebhooks: params => `/spaces/${params.space}/registries/${params?.repositoryIdentifier}/webhooks`
}
