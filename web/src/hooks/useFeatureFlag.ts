// temp file to hide open source pipelines and secrets - can be extended if needs be

const featureFlags = {
  OPEN_SOURCE_PIPELINES: false,
  OPEN_SOURCE_SECRETS: false
}

export const useFeatureFlag = (): Record<keyof typeof featureFlags, boolean> => featureFlags
