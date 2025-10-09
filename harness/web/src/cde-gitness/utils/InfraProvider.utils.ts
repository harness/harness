import googleCloudIcon from 'icons/google-cloud.svg?url'
import awsIcon from 'cde-gitness/assests/aws.svg?url'
import HarnessIcon from 'icons/Harness.svg?url'
import { HARNESS_GCP, HYBRID_VM_GCP, HYBRID_VM_AWS } from 'cde-gitness/constants'

const getProviderIcon = (providerType: string) => {
  if (providerType === HYBRID_VM_GCP) {
    return googleCloudIcon
  } else if (providerType === HARNESS_GCP) {
    return HarnessIcon
  } else if (providerType === HYBRID_VM_AWS) {
    return awsIcon
  }
  return null
}

export default getProviderIcon
