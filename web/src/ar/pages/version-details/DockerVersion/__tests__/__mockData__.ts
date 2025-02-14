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

import type {
  ArtifactVersionSummaryResponseResponse,
  DockerArtifactDetailIntegrationResponseResponse,
  DockerArtifactDetailResponseResponse,
  DockerArtifactManifestResponseResponse,
  DockerLayersResponseResponse,
  DockerManifestsResponseResponse,
  ListArtifactVersion,
  ListArtifactVersionResponseResponse
} from '@harnessio/react-har-service-client'

export const mockDockerLatestVersionListTableData: ListArtifactVersion = {
  artifactVersions: [
    {
      deploymentMetadata: {
        nonProdEnvCount: 0,
        prodEnvCount: 0
      },
      digestCount: 1,
      islatestVersion: true,
      lastModified: '1730978736333',
      name: '1.0.0',
      packageType: 'DOCKER',
      pullCommand: 'docker pull pkg.qa.harness.io/iwnhltqp7q/docker-repo/podinfo-artifact:1.0.0',
      registryIdentifier: '',
      registryPath: '',
      size: '69.56MB'
    }
  ],
  itemCount: 55,
  pageCount: 2,
  pageIndex: 0,
  pageSize: 50
}

export const mockDockerManifestListTableData: DockerManifestsResponseResponse = {
  data: {
    imageName: 'maven-app',
    manifests: [
      {
        createdAt: '1738923119376',
        digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d',
        downloadsCount: 11,
        osArch: 'linux/arm64',
        size: '246.43MB',
        stoExecutionId: 'Tbi7s6nETjmOMKU3Qrnm7A',
        stoPipelineId: 'HARNESS_ARTIFACT_SCAN_PIPELINE'
      },
      {
        createdAt: '1738923119376',
        digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d',
        downloadsCount: 11,
        osArch: 'linux/arm64',
        size: '246.43MB'
      }
    ],
    version: '1.0.0'
  },
  status: 'SUCCESS'
}

export const mockDockerVersionSummary: ArtifactVersionSummaryResponseResponse = {
  data: {
    imageName: 'maven-app',
    packageType: 'DOCKER',
    sscaArtifactId: '67a5dccf6d75916b0c3ea1b6',
    sscaArtifactSourceId: '67a5dccf6d75916b0c3ea1b5',
    stoExecutionId: 'Tbi7s6nETjmOMKU3Qrnm7A',
    stoPipelineId: 'HARNESS_ARTIFACT_SCAN_PIPELINE',
    version: '1.0.0'
  },
  status: 'SUCCESS'
}

export const mockDockerVersionSummaryWithoutSscaAndStoData: ArtifactVersionSummaryResponseResponse = {
  data: {
    imageName: 'maven-app',
    packageType: 'DOCKER',
    version: '1.0.0'
  },
  status: 'SUCCESS'
}

export const mockDockerVersionList: ListArtifactVersionResponseResponse = {
  data: {
    artifactVersions: [
      {
        deploymentMetadata: {
          nonProdEnvCount: 0,
          prodEnvCount: 0
        },
        digestCount: 1,
        downloadsCount: 11,
        islatestVersion: false,
        lastModified: '1738923119434',
        name: '1.0.0',
        packageType: 'DOCKER',
        pullCommand: 'docker pull pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-repo/maven-app:1.0.0',
        registryIdentifier: '',
        registryPath: '',
        size: '246.43MB'
      },
      {
        deploymentMetadata: {
          nonProdEnvCount: 0,
          prodEnvCount: 0
        },
        digestCount: 1,
        downloadsCount: 11,
        islatestVersion: false,
        lastModified: '1738923402541',
        name: '1.0.1',
        packageType: 'DOCKER',
        pullCommand: 'docker pull pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-repo/maven-app:1.0.1',
        registryIdentifier: '',
        registryPath: '',
        size: '246.89MB'
      },
      {
        deploymentMetadata: {
          nonProdEnvCount: 0,
          prodEnvCount: 0
        },
        digestCount: 1,
        downloadsCount: 11,
        islatestVersion: true,
        lastModified: '1738924148637',
        name: '1.0.2',
        packageType: 'DOCKER',
        pullCommand: 'docker pull pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-repo/maven-app:1.0.2',
        registryIdentifier: '',
        registryPath: '',
        size: '246.89MB'
      }
    ],
    itemCount: 3,
    pageCount: 1,
    pageIndex: 0,
    pageSize: 100
  },
  status: 'SUCCESS'
}

export const mockDockerManifestList: DockerManifestsResponseResponse = {
  data: {
    imageName: 'maven-app',
    manifests: [
      {
        createdAt: '1738923119376',
        digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d',
        downloadsCount: 11,
        osArch: 'linux/arm64',
        size: '246.43MB',
        stoExecutionId: 'Tbi7s6nETjmOMKU3Qrnm7A',
        stoPipelineId: 'HARNESS_ARTIFACT_SCAN_PIPELINE'
      },
      {
        createdAt: '1738923119376',
        digest: 'sha256:112cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d',
        downloadsCount: 11,
        osArch: 'linux/amd64',
        size: '246.43MB',
        stoExecutionId: 'Tbi7s6nETjmOMKU3Qrnm7A',
        stoPipelineId: 'HARNESS_ARTIFACT_SCAN_PIPELINE'
      }
    ],
    version: '1.0.0'
  },
  status: 'SUCCESS'
}

export const mockDockerArtifactDetails: DockerArtifactDetailResponseResponse = {
  data: {
    createdAt: '1738923119434',
    downloadsCount: 11,
    imageName: 'maven-app',
    isLatestVersion: false,
    modifiedAt: '1738923119434',
    packageType: 'DOCKER',
    pullCommand: 'docker pull pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-repo/maven-app:1.0.0',
    registryPath: 'docker-repo/maven-app/sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d',
    size: '246.43MB',
    url: 'https://pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-repo/maven-app/1.0.0',
    version: '1.0.0'
  },
  status: 'SUCCESS'
}

export const mockDockerArtifactIntegrationDetails: DockerArtifactDetailIntegrationResponseResponse = {
  data: {
    buildDetails: {
      orgIdentifier: 'default',
      pipelineDisplayName: 'deploy1',
      pipelineExecutionId: 'gs0w_JRPQMSPSd4PG1--hQ',
      pipelineIdentifier: 'deploy1',
      projectIdentifier: 'donotdeleteshivanand',
      stageExecutionId: 'Eo4H_SmyTZifDw3ygOha9Q',
      stepExecutionId: '4jYgJqIbR3q9jvh4oJEmjQ'
    },
    deploymentsDetails: {
      nonProdDeployment: 0,
      prodDeployment: 0,
      totalDeployment: 0
    },
    sbomDetails: {
      allowListViolations: 0,
      artifactId: '67a5dccf6d75916b0c3ea1b6',
      artifactSourceId: '67a5dccf6d75916b0c3ea1b5',
      avgScore: '7.305059523809524',
      componentsCount: 143,
      denyListViolations: 0,
      maxScore: '10',
      orchestrationId: 'yw0D70fiTqetxx0HIyvEUQ',
      orgId: 'default',
      projectId: 'donotdeleteshivanand'
    },
    slsaDetails: {
      provenanceId: '',
      status: ''
    },
    stoDetails: {
      critical: 17,
      executionId: 'Tbi7s6nETjmOMKU3Qrnm7A',
      high: 19,
      ignored: 0,
      info: 0,
      lastScanned: '1738923283',
      low: 0,
      medium: 13,
      pipelineId: 'HARNESS_ARTIFACT_SCAN_PIPELINE',
      total: 49
    }
  },
  status: 'SUCCESS'
}

export const mockDockerArtifactLayers: DockerLayersResponseResponse = {
  data: {
    digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d',
    layers: [
      {
        command: '/bin/sh -c #(nop) ADD file:6fef7a4ab2de57c438dad76949e7eb87cfb1ea6f45b0f2423f71188efdaa0d8e in /',
        size: '40.07 MB'
      },
      {
        command: '/bin/sh -c #(nop)  CMD ["/bin/bash"]',
        size: '0 B'
      },
      {
        command:
          '/bin/sh -c set -eux; \tmicrodnf install \t\tgzip \t\ttar \t\t\t\tbinutils \t\tfreetype fontconfig \t; \tmicrodnf clean all',
        size: '13.63 MB'
      },
      {
        command: '/bin/sh -c #(nop) ENV JAVA_HOME=/usr/java/openjdk-17',
        size: '0 B'
      },
      {
        command:
          '/bin/sh -c #(nop) ENV PATH=/usr/java/openjdk-17/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin',
        size: '0 B'
      },
      {
        command: '/bin/sh -c #(nop)  ENV LANG=C.UTF-8',
        size: '0 B'
      },
      {
        command: '/bin/sh -c #(nop)  ENV JAVA_VERSION=17.0.2',
        size: '0 B'
      },
      {
        command:
          '/bin/sh -c set -eux; \t\tarch="$(objdump="$(command -v objdump)" \u0026\u0026 objdump --file-headers "$objdump" | awk -F \'[:,]+[[:space:]]+\' \'$1 == "architecture" { print $2 }\')"; \tcase "$arch" in \t\t\'i386:x86-64\') \t\t\tdownloadUrl=\'https://download.java.net/java/GA/jdk17.0.2/dfd4a8d0985749f896bed50d7138ee7f/8/GPL/openjdk-17.0.2_linux-x64_bin.tar.gz\'; \t\t\tdownloadSha256=\'0022753d0cceecacdd3a795dd4cea2bd7ffdf9dc06e22ffd1be98411742fbb44\'; \t\t\t;; \t\t\'aarch64\') \t\t\tdownloadUrl=\'https://download.java.net/java/GA/jdk17.0.2/dfd4a8d0985749f896bed50d7138ee7f/8/GPL/openjdk-17.0.2_linux-aarch64_bin.tar.gz\'; \t\t\tdownloadSha256=\'13bfd976acf8803f862e82c7113fb0e9311ca5458b1decaef8a09ffd91119fa4\'; \t\t\t;; \t\t*) echo \u003e\u00262 "error: unsupported architecture: \'$arch\'"; exit 1 ;; \tesac; \t\tcurl -fL -o openjdk.tgz "$downloadUrl"; \techo "$downloadSha256 *openjdk.tgz" | sha256sum --strict --check -; \t\tmkdir -p "$JAVA_HOME"; \ttar --extract \t\t--file openjdk.tgz \t\t--directory "$JAVA_HOME" \t\t--strip-components 1 \t\t--no-same-owner \t; \trm openjdk.tgz*; \t\trm -rf "$JAVA_HOME/lib/security/cacerts"; \tln -sT /etc/pki/ca-trust/extracted/java/cacerts "$JAVA_HOME/lib/security/cacerts"; \t\tln -sfT "$JAVA_HOME" /usr/java/default; \tln -sfT "$JAVA_HOME" /usr/java/latest; \tfor bin in "$JAVA_HOME/bin/"*; do \t\tbase="$(basename "$bin")"; \t\t[ ! -e "/usr/bin/$base" ]; \t\talternatives --install "/usr/bin/$base" "$base" "$bin" 20000; \tdone; \t\tjava -Xshare:dump; \t\tfileEncoding="$(echo \'System.out.println(System.getProperty("file.encoding"))\' | jshell -s -)"; [ "$fileEncoding" = \'UTF-8\' ]; rm -rf ~/.java; \tjavac --version; \tjava --version',
        size: '177.73 MB'
      },
      {
        command: '/bin/sh -c #(nop) CMD ["jshell"]',
        size: '0 B'
      },
      {
        command: 'WORKDIR /app',
        size: '93 B'
      },
      {
        command: 'COPY target/mavensampleapp-1.0.0.jar app.jar # buildkit',
        size: '14.98 MB'
      },
      {
        command: 'EXPOSE map[8080/tcp:{}]',
        size: '0 B'
      },
      {
        command: 'ENTRYPOINT ["java" "-jar" "app.jar"]',
        size: '0 B'
      }
    ],
    osArch: 'linux/arm64'
  },
  status: 'SUCCESS'
}

export const mockDockerArtifactManifest: DockerArtifactManifestResponseResponse = {
  data: {
    manifest:
      '{\n  "schemaVersion": 2,\n  "mediaType": "application/vnd.docker.distribution.manifest.v2+json",\n  "config": {\n    "mediaType": "application/vnd.docker.container.image.v1+json",\n    "digest": "sha256:f152582e19dbda7e3fb677e83af07fb60b05a56757b15edd8915aef2191aad62",\n    "size": 4682\n  },\n  "layers": [\n    {\n      "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",\n      "digest": "sha256:416105dc84fc8cf66df5d2c9f81570a2cc36a6cae58aedd4d58792f041f7a2f5",\n      "size": 42018977\n    },\n    {\n      "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",\n      "digest": "sha256:fe66142579ff5bb0bb5cf989222e2bc77a97dcbd0283887dec04d5b9dfd48cfa",\n      "size": 14294224\n    },\n    {\n      "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",\n      "digest": "sha256:1250d2aa493e8744c8f6cb528c8a882c14b6d7ff0af6862bbbfe676f60ea979e",\n      "size": 186363988\n    },\n    {\n      "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",\n      "digest": "sha256:6a92a853917fafa754249a4f309e5c34caae0ee5df4369dc1c9383d7cd1b395e",\n      "size": 93\n    },\n    {\n      "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",\n      "digest": "sha256:b2e80a78034b601c2513a1b00947bda6c6cda46a95e217b954ed84f5b1b5c0fe",\n      "size": 15712414\n    }\n  ]\n}'
  },
  status: 'SUCCESS'
}

export const mockDockerSbomData = {
  sbom: 'Test Data'
}
