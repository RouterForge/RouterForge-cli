import { TypedUseSelectorHook, useDispatch, useSelector } from 'react-redux';
import type { RootState, AppDispatch } from './store';

// Use throughout your app instead of plain `useDispatch` and `useSelector`
export const useAppDispatch = () => useDispatch<AppDispatch>();
export const useAppSelector: TypedUseSelectorHook<RootState> = useSelector;

// Custom hooks for optimistic updates
export const useOptimisticUpdate = () => {
  const dispatch = useAppDispatch();
  const hasOptimisticUpdates = useAppSelector((state) => {
    const contacts = state.contacts.optimisticUpdates;
    const deals = state.deals.optimisticUpdates;
    const tasks = state.tasks.optimisticUpdates;
    return Object.keys(contacts).length > 0 || Object.keys(deals).length > 0 || Object.keys(tasks).length > 0;
  });

  return {
    hasOptimisticUpdates,
    clearAllOptimisticUpdates: () => {
      dispatch({ type: 'contacts/clearOptimisticUpdates' });
      dispatch({ type: 'deals/clearOptimisticUpdates' });
      dispatch({ type: 'tasks/clearOptimisticUpdates' });
    },
  };
};

// Helper hook for optimistic operations with error handling
export const useOptimisticOperation = () => {
  const { clearAllOptimisticUpdates } = useOptimisticUpdate();
  
  return {
    withOptimisticUpdate: async <T>(
      optimisticAction: () => void,
      asyncOperation: () => Promise<T>,
      rollbackAction?: () => void
    ): Promise<T | null> => {
      try {
        optimisticAction();
        const result = await asyncOperation();
        return result;
      } catch (error) {
        if (rollbackAction) {
          rollbackAction();
        }
        console.error('Optimistic update failed:', error);
        return null;
      }
    },
  };
};