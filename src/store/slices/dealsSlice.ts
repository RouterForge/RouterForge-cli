import { createSlice, createAsyncThunk, PayloadAction } from '@reduxjs/toolkit';
import { Deal, DealFormData } from '../types';
import { DealsAPI } from '../api/deals';

interface DealsState {
  items: Deal[];
  selectedDealId: string | null;
  status: 'idle' | 'loading' | 'succeeded' | 'failed';
  error: string | null;
  optimisticUpdates: Record<string, Deal>;
}

const initialState: DealsState = {
  items: [],
  selectedDealId: null,
  status: 'idle',
  error: null,
  optimisticUpdates: {},
};

export const fetchDeals = createAsyncThunk('deals/fetchDeals', async () => {
  return await DealsAPI.getAll();
});

export const createDeal = createAsyncThunk(
  'deals/createDeal',
  async (dealData: DealFormData, { dispatch }) => {
    // Optimistic update
    const tempId = `temp-deal-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
    const optimisticDeal: Deal = {
      ...dealData,
      id: tempId,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      stage: dealData.stage || 'lead',
      value: dealData.value || 0,
      probability: dealData.probability || 0,
      expectedCloseDate: dealData.expectedCloseDate || new Date().toISOString(),
      assignedTo: dealData.assignedTo || '',
      tags: dealData.tags || [],
    };

    dispatch(addDealOptimistic({ tempId, deal: optimisticDeal }));

    try {
      const realDeal = await DealsAPI.create(dealData);
      dispatch(replaceOptimisticDeal({ tempId, realDeal }));
      return realDeal;
    } catch (error) {
      dispatch(removeOptimisticDeal(tempId));
      throw error;
    }
  }
);

export const updateDeal = createAsyncThunk(
  'deals/updateDeal',
  async ({ id, data }: { id: string; data: Partial<DealFormData> }, { dispatch, getState }) => {
    const state = getState() as { deals: DealsState };
    const originalDeal = state.deals.items.find(d => d.id === id);

    const optimisticDeal: Deal = {
      ...originalDeal!,
      ...data,
      updatedAt: new Date().toISOString(),
    };

    dispatch(updateDealOptimistic({ id, deal: optimisticDeal }));

    try {
      const realDeal = await DealsAPI.update(id, data);
      dispatch(replaceOptimisticDeal({ tempId: id, realDeal }));
      return realDeal;
    } catch (error) {
      if (originalDeal) {
        dispatch(replaceOptimisticDeal({ tempId: id, realDeal: originalDeal }));
      }
      throw error;
    }
  }
);

export const updateDealStage = createAsyncThunk(
  'deals/updateDealStage',
  async ({ id, stage }: { id: string; stage: string }, { dispatch, getState }) => {
    const state = getState() as { deals: DealsState };
    const originalDeal = state.deals.items.find(d => d.id === id);

    // Optimistic stage update
    const optimisticDeal: Deal = {
      ...originalDeal!,
      stage,
      updatedAt: new Date().toISOString(),
    };

    dispatch(updateDealOptimistic({ id, deal: optimisticDeal }));

    try {
      const realDeal = await DealsAPI.updateStage(id, stage);
      dispatch(replaceOptimisticDeal({ tempId: id, realDeal }));
      return realDeal;
    } catch (error) {
      if (originalDeal) {
        dispatch(replaceOptimisticDeal({ tempId: id, realDeal: originalDeal }));
      }
      throw error;
    }
  }
);

export const deleteDeal = createAsyncThunk(
  'deals/deleteDeal',
  async (id: string, { dispatch, getState }) => {
    const state = getState() as { deals: DealsState };
    const dealToDelete = state.deals.items.find(d => d.id === id);

    dispatch(removeDealOptimistic(id));

    try {
      await DealsAPI.delete(id);
      return id;
    } catch (error) {
      if (dealToDelete) {
        dispatch(addDealOptimistic({ tempId: id, deal: dealToDelete }));
      }
      throw error;
    }
  }
);

const dealsSlice = createSlice({
  name: 'deals',
  initialState,
  reducers: {
    addDealOptimistic: (state, action: PayloadAction<{ tempId: string; deal: Deal }>) => {
      const { tempId, deal } = action.payload;
      state.items.push(deal);
      state.optimisticUpdates[tempId] = deal;
    },
    
    updateDealOptimistic: (state, action: PayloadAction<{ id: string; deal: Deal }>) => {
      const { id, deal } = action.payload;
      const index = state.items.findIndex(d => d.id === id);
      if (index !== -1) {
        state.items[index] = deal;
        state.optimisticUpdates[id] = deal;
      }
    },
    
    removeDealOptimistic: (state, action: PayloadAction<string>) => {
      const id = action.payload;
      const dealToRemove = state.items.find(d => d.id === id);
      state.items = state.items.filter(d => d.id !== id);
      if (dealToRemove) {
        state.optimisticUpdates[id] = dealToRemove;
      }
    },
    
    replaceOptimisticDeal: (state, action: PayloadAction<{ tempId: string; realDeal: Deal }>) => {
      const { tempId, realDeal } = action.payload;
      const index = state.items.findIndex(d => d.id === tempId);
      if (index !== -1) {
        state.items[index] = realDeal;
      }
      delete state.optimisticUpdates[tempId];
    },
    
    removeOptimisticDeal: (state, action: PayloadAction<string>) => {
      const tempId = action.payload;
      state.items = state.items.filter(d => d.id !== tempId);
      delete state.optimisticUpdates[tempId];
    },
    
    setSelectedDeal: (state, action: PayloadAction<string | null>) => {
      state.selectedDealId = action.payload;
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchDeals.pending, (state) => {
        state.status = 'loading';
      })
      .addCase(fetchDeals.fulfilled, (state, action) => {
        state.status = 'succeeded';
        state.items = action.payload;
        state.error = null;
      })
      .addCase(fetchDeals.rejected, (state, action) => {
        state.status = 'failed';
        state.error = action.error.message || 'Failed to fetch deals';
      })
      .addCase(createDeal.rejected, (state, action) => {
        state.error = action.error.message || 'Failed to create deal';
      })
      .addCase(updateDeal.rejected, (state, action) => {
        state.error = action.error.message || 'Failed to update deal';
      })
      .addCase(updateDealStage.rejected, (state, action) => {
        state.error = action.error.message || 'Failed to update deal stage';
      })
      .addCase(deleteDeal.rejected, (state, action) => {
        state.error = action.error.message || 'Failed to delete deal';
      });
  },
});

export const {
  addDealOptimistic,
  updateDealOptimistic,
  removeDealOptimistic,
  replaceOptimisticDeal,
  removeOptimisticDeal,
  setSelectedDeal,
} = dealsSlice.actions;

export default dealsSlice.reducer;