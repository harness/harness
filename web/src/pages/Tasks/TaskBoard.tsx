/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useState } from 'react'
import { Link } from 'react-router-dom'
import { Container, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useAppContext } from 'AppContext'
import { StringKeys, useStrings } from 'framework/strings'
import { TaskStateBadge } from 'components/TaskStateBadge/TaskStateBadge'
import { TASK_MAIN_FLOW_STATES, TASK_SIDE_STATES, TaskState, TypesTask, getAssigneeLabel } from 'utils/TaskUtils'
import css from './Tasks.module.scss'

interface TaskBoardProps {
  repoPath: string
  tasks: TypesTask[]
  // Resolves when the state change has been applied (the caller shows API errors).
  onMove: (task: TypesTask, state: TaskState) => void
}

const DRAG_TYPE = 'application/x-task-number'

export function TaskBoard({ repoPath, tasks, onMove }: TaskBoardProps) {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const [overState, setOverState] = useState<TaskState | undefined>()

  const renderColumn = (state: TaskState) => {
    const columnTasks = tasks.filter(task => task.state === state)
    return (
      <Container
        key={state}
        className={`${css.column} ${overState === state ? css.columnOver : ''}`}
        onDragOver={(e: React.DragEvent) => {
          e.preventDefault()
          setOverState(state)
        }}
        onDragLeave={() => setOverState(undefined)}
        onDrop={(e: React.DragEvent) => {
          e.preventDefault()
          setOverState(undefined)
          const number = Number(e.dataTransfer.getData(DRAG_TYPE))
          const task = tasks.find(t => t.number === number)
          if (task && task.state !== state) {
            onMove(task, state)
          }
        }}>
        <div className={css.columnHeader}>
          <TaskStateBadge state={state} />
          <Text color={Color.GREY_500}>{columnTasks.length}</Text>
        </div>
        {columnTasks.map(task => (
          <div
            key={task.number}
            className={css.card}
            draggable
            onDragStart={(e: React.DragEvent) => e.dataTransfer.setData(DRAG_TYPE, String(task.number))}>
            <Link to={routes.toCODETask({ repoPath, taskNumber: String(task.number) })}>
              <Text color={Color.GREY_800} lineClamp={2} className={css.cardTitle}>
                #{task.number} {task.title}
              </Text>
            </Link>
            <div className={css.cardMeta}>
              <span>{task.priority?.toUpperCase()}</span>
              <span>{getString(`tasks.originValue.${task.origin}` as StringKeys)}</span>
              <span>{getAssigneeLabel(task, getString)}</span>
            </div>
          </div>
        ))}
      </Container>
    )
  }

  return (
    <div>
      <Text className={css.groupTitle}>{getString('tasks.mainFlow')}</Text>
      <div className={css.board}>{TASK_MAIN_FLOW_STATES.map(renderColumn)}</div>
      <Text className={css.groupTitle}>{getString('tasks.sideStates')}</Text>
      <div className={css.board}>{TASK_SIDE_STATES.map(renderColumn)}</div>
    </div>
  )
}
