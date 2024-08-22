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

import * as yup from 'yup'
import type { UseStringsReturn } from 'framework/strings'

export const validateGitnessForm = (getString: UseStringsReturn['getString']) =>
  yup.object().shape({
    branch: yup.string().trim().required(getString('cde.branchValidationMessage')),
    code_repo_url: yup.string().trim().required(getString('cde.repoValidationMessage')),
    identifier: yup.string().trim().required(),
    ide: yup.string().trim().required(),
    resource_identifier: yup.string().trim().required(getString('cde.machineValidationMessage')),
    name: yup.string().trim().required()
  })
