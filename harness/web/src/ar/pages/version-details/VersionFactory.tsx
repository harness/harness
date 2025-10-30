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

import versionFactory from '@ar/frameworks/Version/VersionFactory'
import { DockerVersionType } from './DockerVersion/DockerVersionType'
import { HelmVersionType } from './HelmVersion/HelmVersionType'
import { GenericVersionType } from './GenericVersion/GenericVersionType'
import { MavenVersionType } from './MavenVersion/MavenVersion'
import { NpmVersionType } from './NpmVersion/NpmVersionType'
import { PythonVersionType } from './PythonVersion/PythonVersionType'
import { NuGetVersionType } from './NuGetVersion/NuGetVersionType'
import { RPMVersionType } from './RPMVersion/RPMVersionType'
import { CargoVersionType } from './CargoVersion/CargoVersionType'
import { GoVersionType } from './GoVersion/GoVersionType'
import { HuggingfaceVersionType } from './HuggingfaceVersion/HuggingfaceVersionType'

versionFactory.registerStep(new DockerVersionType())
versionFactory.registerStep(new HelmVersionType())
versionFactory.registerStep(new GenericVersionType())
versionFactory.registerStep(new MavenVersionType())
versionFactory.registerStep(new NpmVersionType())
versionFactory.registerStep(new PythonVersionType())
versionFactory.registerStep(new NuGetVersionType())
versionFactory.registerStep(new RPMVersionType())
versionFactory.registerStep(new CargoVersionType())
versionFactory.registerStep(new GoVersionType())
versionFactory.registerStep(new HuggingfaceVersionType())
