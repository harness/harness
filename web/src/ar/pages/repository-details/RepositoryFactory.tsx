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

import repositoryFactory from '@ar/frameworks/RepositoryStep/RepositoryFactory'
import { DockerRepositoryType } from './DockerRepository/DockerRepositoryType'
import { MavenRepositoryType } from './MavenRepository/MavenRepository'
import { HelmRepositoryType } from './HelmRepository/HelmRepositoryType'
import { GenericRepositoryType } from './GenericRepository/GenericRepositoryType'
import { NpmRepositoryType } from './NpmRepository/NpmRepositoryType'
import { PythonRepositoryType } from './PythonRepository/PythonRepositoryType'
import { NuGetRepositoryType } from './NuGetRepository/NuGetRepositoryType'
import { RPMRepositoryType } from './RPMRepository/RPMRepositoryType'
import { CargoRepositoryType } from './CargoRepository/CargoRepositoryType'

repositoryFactory.registerStep(new DockerRepositoryType())
repositoryFactory.registerStep(new HelmRepositoryType())
repositoryFactory.registerStep(new GenericRepositoryType())
repositoryFactory.registerStep(new MavenRepositoryType())
repositoryFactory.registerStep(new NpmRepositoryType())
repositoryFactory.registerStep(new PythonRepositoryType())
repositoryFactory.registerStep(new NuGetRepositoryType())
repositoryFactory.registerStep(new RPMRepositoryType())
repositoryFactory.registerStep(new CargoRepositoryType())
