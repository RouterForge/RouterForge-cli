import { createSlice, createAsyncThunk, PayloadAction } from '@reduxjs/toolkit';
import { Task, TaskFormData } from '../types';
import { TasksAPI } from '../api/tasks';

interface TasksState {
  items: Task[];
  selectedTaskId: string | null;
  status: 'idle' | 'loading' | 'succeeded' | 'failed';
  error: string | null;
  optimisticUpdates: Record<string, Task>;
}

const initialState: TasksState = {
  items: [],
  selectedTaskId: null,
  status: 'idle',
  error: null,
  optimisticUpdates: {},
};

export const fetchTasks = createAsyncThunk('tasks/fetchTasks', async () => {
  return await TasksAPI.getAll();
});

export const createTask = createAsyncThunk(
  'tasks/createTask',
  async (taskData: TaskFormData, { dispatch }) => {
    const tempId = `temp-task-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
    const optimisticTask: Task = {
      ...taskData,
      id: tempId,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      completed: false,
      priority: taskData.priority || 'medium',
      dueDate: taskData.dueDate || new Date().toISOString(),
      assignedTo: taskData.assignedTo || '',
      relatedTo: taskData.relatedTo || { type: 'contact', id: '' },
      tags: taskData.tags || [],
    };

    dispatch(addTaskOptimistic({ tempId, task: optimisticTask }));

    try {
      const realTask = await TasksAPI.create(taskData);
      dispatch(replaceOptimisticTask({ tempId, realTask }));
      return realTask;
    } catch (error) {
      dispatch(removeOptimisticTask(tempId));
      throw error;
    }
  }
);

export const updateTask = createAsyncThunk(
  'tasks/updateTask',
  async ({ id, data }: { id: string; data: Partial<TaskFormData> }, { dispatch, getState }) => {
    const state = getState() as { tasks: TasksState };
    const originalTask = state.tasks.items.find(t => t.id === id);

    const optimisticTask: Task = {
      ...originalTask!,
      ...data,
      updatedAt: new Date().toISOString(),
    };

    dispatch(updateTaskOptimistic({ id, task: optimisticTask }));

    try {
      const realTask = await TasksAPI.update(id, data);
      dispatch(replaceOptimisticTask({ tempId: id, realTask }));
      return realTask;
    } catch (error) {
      if (originalTask) {
        dispatch(replaceOptimisticTask({ tempId: id, realTask: originalTask }));
      }
      throw error;
    }
  }
);

export const toggleTaskCompletion = createAsyncThunk(
  'tasks/toggleTaskCompletion',
  async (id: string, { dispatch, getState }) => {
    const state = getState() as { tasks: TasksState };
    const originalTask = state.tasks.items.find(t => t.id === id);

    // Optimistic toggle
    const optimisticTask: Task = {
      ...originalTask!,
      completed: !originalTask!.completed,
      updatedAt: new Date().toISOString(),
    };

    dispatch(updateTaskOptimistic({ id, task: optimisticTask }));

    try {
      const realTask = await TasksAPI.toggleCompletion(id);
      dispatch(replaceOptimisticTask({ tempId: id, realTask }));
      return realTask;
    } catch (error) {
      if (originalTask) {
        dispatch(replaceOptimisticTask({ tempId: id, realTask: originalTask }));
      }
      throw error;
    }
  }
);

export const deleteTask = createAsyncThunk(
  'tasks/deleteTask',
  async (id: string, { dispatch, getState }) => {
    const state = getState() as { tasks: TasksState };
    const taskToDelete = state.tasks.items.find(t => t.id === id);

    dispatch(removeTaskOptimistic(id));

    try {
      await TasksAPI.delete(id);
      return id;
    } catch (error) {
      if (taskToDelete) {
        dispatch(addTaskOptimistic({ tempId: id, task: taskToDelete }));
      }
      throw error;
    }
  }
);

const tasksSlice = createSlice({
  name: 'tasks',
  initialState,
  reducers: {
    addTaskOptimistic: (state, action: PayloadAction<{ tempId: string; task: Task }>) => {
      const { tempId, task } = action.payload;
      state.items.push(task);
      state.optimisticUpdates[tempId] = task;
    },
    
    updateTaskOptimistic: (state, action: PayloadAction<{ id: string; task: Task }>) => {
      const { id, task } = action.payload;
      const index = state.items.findIndex(t => t.id === id);
      if (index !== -1) {
        state.items[index] = task;
        state.optimisticUpdates[id] = task;
      }
    },
    
    removeTaskOptimistic: (state, action: PayloadAction<string>) => {
      const id = action.payload;
      const taskToRemove = state.tasks.items.find(t => t.id === id);
      state.items = state.items.filter(t => t.id !== id);
      if (taskToRemove) {
        state.optimisticUpdates[id] = taskToRemove;
      }
    },
    
    replaceOptimisticTask: (state, action: PayloadAction<{ tempId: string; realTask: Task }>) => {
      const { tempId, realTask } = action.payload;
      const index = state.items.findIndex(t => t.id === tempId);
      if (index !== -1) {
        state.items[index] = realTask;
      }
      delete state.optimisticUpdates[tempId];
    },
    
    removeOptimisticTask: (state, action: PayloadAction<string>) => {
      const tempId = action.payload;
      state.items = state.items.filter(t => t.id !== tempId);
      delete state.optimisticUpdates[tempId];
    },
    
    setSelectedTask: (state, action: PayloadAction<string | null>) => {
      state.selectedTaskId = action.payload;
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchTasks.pending, (state) => {
        state.status = 'loading';
      })
      .addCase(fetchTasks.fulfilled, (state, action) => {
        state.status = 'succeeded';
        state.items = action.payload;
        state.error = null;
      })
      .addCase(fetchTasks.rejected, (state, action) => {
        state.status = 'failed';
        state.error = action.error.message || 'Failed to fetch tasks';
      })
      .addCase(createTask.rejected, (state, action) => {
        state.error = action.error.message || 'Failed to create task';
      })
      .addCase(updateTask.rejected, (state, action) => {
        state.error = action.error.message || 'Failed to update task';
      })
      .addCase(toggleTaskCompletion.rejected, (state, action) => {
        state.error = action.error.message || 'Failed to toggle task completion';
      })
      .addCase(deleteTask.rejected, (state, action) => {
        state.error = action.error.message || 'Failed to delete task';
      });
  },
});

export const {
  addTaskOptimistic,
  updateTaskOptimistic,
  removeTaskOptimistic,
  replaceOptimisticTask,
  removeOptimisticTask,
  setSelectedTask,
} = tasksSlice.actions;

export default tasksSlice.reducer;