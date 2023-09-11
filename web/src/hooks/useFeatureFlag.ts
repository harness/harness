// temp file to hide open source pipelines and secrets - can be extended if needs be

const featureFlags = {
  // eslint-disable-next-line @typescript-eslint/ban-ts-comment
  // @ts-ignore
  OPEN_SOURCE_PIPELINES: __STANDALONE__ ?? false,
  // eslint-disable-next-line @typescript-eslint/ban-ts-comment
  // @ts-ignore
  OPEN_SOURCE_SECRETS: __STANDALONE__ ?? false
}

export const useFeatureFlag = (): Record<keyof typeof featureFlags, boolean> => featureFlags
