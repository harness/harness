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

import type { GetAllArtifactsByRegistryOkResponse } from '@harnessio/react-har-service-client'

export const MockGetMavenRegistryResponseWithAllData = {
  content: {
    data: {
      config: {
        type: 'VIRTUAL',
        upstreamProxies: ['maven-central-proxy']
      },
      createdAt: '1738040653409',
      description: 'custom description',
      identifier: 'maven-repo',
      labels: ['label1', 'label2', 'label3', 'label4'],
      modifiedAt: '1738040653409',
      packageType: 'MAVEN',
      url: 'http://host.docker.internal:3000/artifact-registry/maven-repo',
      allowedPattern: ['test1', 'test2'],
      blockedPattern: ['test3', 'test4']
    },
    status: 'SUCCESS'
  }
}

export const MockGetMavenArtifactsByRegistryResponse: GetAllArtifactsByRegistryOkResponse = {
  content: {
    data: {
      artifacts: [
        {
          downloadsCount: 0,
          lastModified: '1738048875014',
          latestVersion: 'v1',
          name: 'artifact',
          packageType: 'MAVEN',
          registryIdentifier: 'maven-repo',
          registryPath: ''
        }
      ],
      itemCount: 0,
      pageCount: 0,
      pageIndex: 0,
      pageSize: 50
    },
    status: 'SUCCESS'
  }
}

export const MockGetMavenSetupClientOnRegistryConfigPageResponse = {
  content: {
    data: {
      mainHeader: 'Maven Client Setup',
      secHeader: 'Follow these instructions to install/use Maven artifacts or compatible packages.',
      sections: [
        {
          header: '1. Generate Identity Token',
          secHeader: 'An identity token will serve as the password for uploading and downloading artifacts.',
          steps: [
            {
              header: 'Generate an identity token',
              type: 'GenerateToken'
            }
          ],
          type: 'INLINE'
        },
        {
          tabs: [
            {
              header: 'Maven',
              sections: [
                {
                  header: '2. Pull a Maven Package',
                  secHeader: 'Set default repository in your pom.xml file.',
                  steps: [
                    {
                      commands: [
                        {
                          value:
                            '\u003crepositories\u003e\n  \u003crepository\u003e\n    \u003cid\u003emaven-dev\u003c/id\u003e\n    \u003curl\u003ehttp://host.docker.internal:3000/maven/artifact-registry/maven-repo\u003c/url\u003e\n    \u003creleases\u003e\n      \u003cenabled\u003etrue\u003c/enabled\u003e\n      \u003cupdatePolicy\u003ealways\u003c/updatePolicy\u003e\n    \u003c/releases\u003e\n    \u003csnapshots\u003e\n      \u003cenabled\u003etrue\u003c/enabled\u003e\n      \u003cupdatePolicy\u003ealways\u003c/updatePolicy\u003e\n    \u003c/snapshots\u003e\n  \u003c/repository\u003e\n\u003c/repositories\u003e'
                        }
                      ],
                      header: 'To set default registry in your pom.xml file by adding the following:',
                      type: 'Static'
                    },
                    {
                      commands: [
                        {
                          value:
                            '\u003csettings\u003e\n  \u003cservers\u003e\n    \u003cserver\u003e\n      \u003cid\u003emaven-dev\u003c/id\u003e\n      \u003cusername\u003eadmin@gitness.io\u003c/username\u003e\n      \u003cpassword\u003eidentity-token\u003c/password\u003e\n    \u003c/server\u003e\n  \u003c/servers\u003e\n\u003c/settings\u003e'
                        }
                      ],
                      header:
                        'Copy the following your ~/ .m2/settings.xml file for MacOs, or $USERPROFILE$\\ .m2\\settings.xml for Windows to authenticate with token to pull from your Maven registry.',
                      type: 'Static'
                    },
                    {
                      commands: [
                        {
                          value:
                            '\u003cdependency\u003e\n  \u003cgroupId\u003e\u003cGROUP_ID\u003e\u003c/groupId\u003e\n  \u003cartifactId\u003e\u003cARTIFACT_ID\u003e\u003c/artifactId\u003e\n  \u003cversion\u003e\u003cVERSION\u003e\u003c/version\u003e\n\u003c/dependency\u003e'
                        }
                      ],
                      header:
                        "Add a dependency to the project's pom.xml (replace \u003cGROUP_ID\u003e, \u003cARTIFACT_ID\u003e \u0026 \u003cVERSION\u003e with your own):",
                      type: 'Static'
                    },
                    {
                      commands: [
                        {
                          value: 'mvn install'
                        }
                      ],
                      header: 'Install dependencies in pom.xml file',
                      type: 'Static'
                    }
                  ],
                  type: 'INLINE'
                },
                {
                  header: '3. Push a Maven Package',
                  secHeader: 'Set default repository in your pom.xml file.',
                  steps: [
                    {
                      commands: [
                        {
                          value:
                            '\u003cdistributionManagement\u003e\n  \u003csnapshotRepository\u003e\n    \u003cid\u003emaven-dev\u003c/id\u003e\n    \u003curl\u003ehttp://host.docker.internal:3000/maven/artifact-registry/maven-repo\u003c/url\u003e\n  \u003c/snapshotRepository\u003e\n  \u003crepository\u003e\n    \u003cid\u003emaven-dev\u003c/id\u003e\n    \u003curl\u003ehttp://host.docker.internal:3000/maven/artifact-registry/maven-repo\u003c/url\u003e\n  \u003c/repository\u003e\n\u003c/distributionManagement\u003e'
                        }
                      ],
                      header: 'To set default registry in your pom.xml file by adding the following:',
                      type: 'Static'
                    },
                    {
                      commands: [
                        {
                          value:
                            '\u003csettings\u003e\n  \u003cservers\u003e\n    \u003cserver\u003e\n      \u003cid\u003emaven-dev\u003c/id\u003e\n      \u003cusername\u003eadmin@gitness.io\u003c/username\u003e\n      \u003cpassword\u003eidentity-token\u003c/password\u003e\n    \u003c/server\u003e\n  \u003c/servers\u003e\n\u003c/settings\u003e'
                        }
                      ],
                      header:
                        'Copy the following your ~/ .m2/setting.xml file for MacOs, or $USERPROFILE$\\ .m2\\settings.xml for Windows to authenticate with token to push to your Maven registry.',
                      type: 'Static'
                    },
                    {
                      commands: [
                        {
                          value: 'mvn deploy'
                        }
                      ],
                      header: 'Publish package to your Maven registry.',
                      type: 'Static'
                    }
                  ],
                  type: 'INLINE'
                }
              ]
            },
            {
              header: 'Gradle',
              sections: [
                {
                  header: '2. Pull a Gradle Package',
                  secHeader: 'Set default repository in your build.gradle file.',
                  steps: [
                    {
                      commands: [
                        {
                          value:
                            'repositories{\n    maven{\n      url “http://host.docker.internal:3000/maven/artifact-registry/maven-repo”\n\n      credentials {\n         username “admin@gitness.io”\n         password “identity-token”\n      }\n   }\n}'
                        }
                      ],
                      header: 'Set the default registry in your project’s build.gradle by adding the following:',
                      type: 'Static'
                    },
                    {
                      commands: [
                        {
                          value: 'repositoryUser=admin@gitness.io\nrepositoryPassword={{identity-token}}'
                        }
                      ],
                      header:
                        'As this is a private registry, you’ll need to authenticate. Create or add to the ~/.gradle/gradle.properties file with the following:',
                      type: 'Static'
                    },
                    {
                      commands: [
                        {
                          value:
                            'dependencies {\n  implementation ‘\u003cGROUP_ID\u003e:\u003cARTIFACT_ID\u003e:\u003cVERSION\u003e’\n}'
                        }
                      ],
                      header: 'Add a dependency to the project’s build.gradle',
                      type: 'Static'
                    },
                    {
                      commands: [
                        {
                          value: 'gradlew build     // Linux or OSX\n gradlew.bat build  // Windows'
                        }
                      ],
                      header: 'Install dependencies in build.gradle file',
                      type: 'Static'
                    }
                  ],
                  type: 'INLINE'
                },
                {
                  header: '3. Push a Gradle Package',
                  secHeader: 'Set default repository in your build.gradle file.',
                  steps: [
                    {
                      commands: [
                        {
                          value:
                            "publishing {\n    publications {\n        maven(MavenPublication) {\n            groupId = '\u003cGROUP_ID\u003e'\n            artifactId = '\u003cARTIFACT_ID\u003e'\n            version = '\u003cVERSION\u003e'\n\n            from components.java\n        }\n    }\n}"
                        }
                      ],
                      header: 'Add a maven publish plugin configuration to the project’s build.gradle.',
                      type: 'Static'
                    },
                    {
                      commands: [
                        {
                          value: 'gradlew publish'
                        }
                      ],
                      header: 'Publish package to your Maven registry.',
                      type: 'Static'
                    }
                  ],
                  type: 'INLINE'
                }
              ]
            },
            {
              header: 'Sbt/Scala',
              sections: [
                {
                  header: '2. Pull a Sbt/Scala Package',
                  secHeader: 'Set default repository in your build.sbt file.',
                  steps: [
                    {
                      commands: [
                        {
                          value:
                            'resolver += “Harness Registry” at “http://host.docker.internal:3000/maven/artifact-registry/maven-repo”\ncredentials += Credentials(Path.userHome / “.sbt” / “.Credentials”)'
                        }
                      ],
                      header: 'Set the default registry in your project’s build.sbt by adding the following:',
                      type: 'Static'
                    },
                    {
                      commands: [
                        {
                          value:
                            'realm=Harness Registry\nhost=host.docker.internal:3000\nuser=admin@gitness.io\npassword={{identity-token}}'
                        }
                      ],
                      header:
                        'As this is a private registry, you’ll need to authenticate. Create or add to the ~/.sbt/.credentials file with the following:',
                      type: 'Static'
                    },
                    {
                      commands: [
                        {
                          value:
                            'libraryDependencies += “\u003cGROUP_ID\u003e” % “\u003cARTIFACT_ID\u003e” % “\u003cVERSION\u003e”'
                        }
                      ],
                      header: 'Add a dependency to the project’s build.sbt',
                      type: 'Static'
                    },
                    {
                      commands: [
                        {
                          value: 'sbt update'
                        }
                      ],
                      header: 'Install dependencies in build.sbt file',
                      type: 'Static'
                    }
                  ],
                  type: 'INLINE'
                },
                {
                  header: '3. Push a Sbt/Scala Package',
                  secHeader: 'Set default repository in your build.sbt file.',
                  steps: [
                    {
                      commands: [
                        {
                          value:
                            'publishTo := Some("Harness Registry" at "http://host.docker.internal:3000/maven/artifact-registry/maven-repo")'
                        }
                      ],
                      header: 'Add publish configuration to the project’s build.sbt.',
                      type: 'Static'
                    },
                    {
                      commands: [
                        {
                          value: 'sbt publish'
                        }
                      ],
                      header: 'Publish package to your Maven registry.',
                      type: 'Static'
                    }
                  ],
                  type: 'INLINE'
                }
              ]
            }
          ],
          type: 'TABS'
        }
      ]
    },
    status: 'SUCCESS'
  }
}

export const MockGetMavenUpstreamRegistryResponseWithMavenCentralSourceAllData = {
  content: {
    data: {
      allowedPattern: ['test1', 'test2'],
      blockedPattern: ['test3', 'test4'],
      config: {
        auth: null,
        authType: 'Anonymous',
        source: 'MavenCentral',
        type: 'UPSTREAM',
        url: ''
      },
      createdAt: '1738516362995',
      identifier: 'maven-up-repo',
      description: 'test description',
      packageType: 'MAVEN',
      labels: ['label1', 'label2', 'label3', 'label4'],
      url: ''
    },
    status: 'SUCCESS'
  }
}
