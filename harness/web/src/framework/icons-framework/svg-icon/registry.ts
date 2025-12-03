const registry = new Map<string, string>()

export const registerIcon = (name: string, svg: string) => registry.set(name, svg)
export const unregisterIcon = (name: string) => registry.delete(name)
export const getIcon = (name: string) => registry.get(name)
export const hasIcon = (name: string) => registry.has(name)
export const getAllIcons = () => registry.keys()
