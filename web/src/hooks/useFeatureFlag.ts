// temp file to hide open source pipelines and secrets - can be extended if needs be

const featureFlags = {
  OPEN_SOURCE_PIPELINES: true,
  OPEN_SOURCE_SECRETS: true
}

export const useFeatureFlag = (): Record<keyof typeof featureFlags, boolean> => featureFlags
