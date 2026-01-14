import DockerIcon from 'cde-gitness/assests/dockerhub.svg?url'
import AWSIcon from 'cde-gitness/assests/aws.svg?url'
import ArtifactoryIcon from 'cde-gitness/assests/artifactory.svg?url'
import NexusIcon from 'cde-gitness/assests/nexus.svg?url'

export const Connectors = {
  NEXUS: 'Nexus',
  ARTIFACTORY: 'Artifactory',
  AWS: 'Aws',
  DOCKER: 'DockerRegistry'
}

export const getConnectorDisplayName = (type: string | undefined): string => {
  switch (type) {
    case Connectors.DOCKER:
      return 'Docker Registry'
    case Connectors.AWS:
      return 'AWS - Cloud Provider'
    case Connectors.NEXUS:
      return 'Nexus'
    case Connectors.ARTIFACTORY:
      return 'Artifactory'
    default:
      return ''
  }
}

export const getConnectorIcon = (type: string | undefined): string => {
  switch (type) {
    case Connectors.DOCKER:
      return DockerIcon
    case Connectors.AWS:
      return AWSIcon
    case Connectors.NEXUS:
      return NexusIcon
    case Connectors.ARTIFACTORY:
      return ArtifactoryIcon
    default:
      return ''
  }
}
