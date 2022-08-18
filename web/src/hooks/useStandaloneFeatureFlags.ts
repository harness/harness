import type { FeatureFlagMap } from '../utils/GovernanceUtils'

export function useStandaloneFeatureFlags(): FeatureFlagMap {
  return {
    OPA_PIPELINE_GOVERNANCE: true,
    OPA_FF_GOVERNANCE: false,
    CUSTOM_POLICY_STEP: false,
    OPA_GIT_GOVERNANCE: false,
    OPA_SECRET_GOVERNANCE: false
  }
}
