export enum CacheStrategyDuration {
  SHORT = 5 * 60 * 1000, // 5 mins
  MEDIUM = 30 * 60 * 1000 // 30 mins
}

export function newCacheStrategy(duration: CacheStrategyDuration = CacheStrategyDuration.MEDIUM) {
  let time = 0

  return {
    isExpired: () => Date.now() - time > duration,
    update: () => {
      time = Date.now()
    }
  }
}
