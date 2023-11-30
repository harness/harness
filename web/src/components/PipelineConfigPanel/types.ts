export enum PipelineEntity {
  STEP = 'STEP'
}

export enum CodeLensAction {
  ADD = 'ADD',
  EDIT = 'EDIT'
}

export interface CodeLensClickMetaData {
  entity: PipelineEntity
  action: CodeLensAction
  highlightSelection?: boolean
}
