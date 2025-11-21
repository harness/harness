export enum PipelineEntity {
  STEP = 'STEP'
}

export enum Action {
  ADD = 'ADD',
  EDIT = 'EDIT'
}

export interface CodeLensClickMetaData {
  entity: PipelineEntity
  action: Action
  highlightSelection?: boolean
}
