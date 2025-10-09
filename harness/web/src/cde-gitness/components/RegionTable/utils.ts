export function handleToggleExpandableRow<T>(value: T): (prev: Set<T>) => Set<T> {
  return (prevValue: Set<T>): Set<T> => {
    if (value) {
      const isRowExpanded = prevValue.has(value)
      if (!isRowExpanded) {
        prevValue.add(value)
      } else {
        prevValue.delete(value)
      }
    }
    return new Set(prevValue)
  }
}
