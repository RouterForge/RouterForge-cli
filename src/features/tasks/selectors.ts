import { createSelector } from '@reduxjs/toolkit';
import { RootState } from '../../app/store';

const selectTasksState = (state: RootState) => state.tasks;

export const selectAllTasks = createSelector(
  [selectTasksState],
  (tasks) => tasks.ids.map(id => tasks.entities[id])
);

export const selectTaskById = (taskId: string) =>
  createSelector(
    [selectTasksState],
    (tasks) => tasks.entities[taskId]
  );

export const selectTasksLoading = createSelector(
  [selectTasksState],
  (tasks) => tasks.loading
);

export const selectTasksError = createSelector(
  [selectTasksState],
  (tasks) => tasks.error
);

export const selectCompletedTasks = createSelector(
  [selectAllTasks],
  (tasks) => tasks.filter(task => task.completed)
);

export const selectPendingTasks = createSelector(
  [selectAllTasks],
  (tasks) => tasks.filter(task => !task.completed)
);

export const selectOverdueTasks = createSelector(
  [selectAllTasks],
  (tasks) => tasks.filter(task => 
    !task.completed && task.dueDate && task.dueDate < new Date().toISOString()
  )
);

export const selectTasksByPriority = (priority: string) =>
  createSelector(
    [selectAllTasks],
    (tasks) => tasks.filter(task => task.priority === priority)
  );

export const selectTasksByDeal = (dealId: string) =>
  createSelector(
    [selectAllTasks],
    (tasks) => tasks.filter(task => task.dealId === dealId)
  );