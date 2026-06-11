import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { Deal } from '../../types';

interface DealsState {
  items: Deal[];
  loading: boolean;
  error: string | null;
}

const initialState: DealsState = {
  items: [],
  loading: false,
  error: null,
};

const dealsSlice = createSlice({
  name: 'deals',
  initialState,
  reducers: {
    fetchDealsStart: (state) => {
      state.loading = true;
      state.error = null;
    },
    fetchDealsSuccess: (state, action: PayloadAction<Deal[]>) => {
      state.items = action.payload;
      state.loading = false;
    },
    fetchDealsFailure: (state, action: PayloadAction<string>) => {
      state.loading = false;
      state.error = action.payload;
    },
    addDeal: (state, action: PayloadAction<Deal>) => {
      state.items.push(action.payload);
    },
    updateDeal: (state, action: PayloadAction<Deal>) => {
      const index = state.items.findIndex(
        (deal) => deal.id === action.payload.id
      );
      if (index !== -1) {
        state.items[index] = action.payload;
      }
    },
    deleteDeal: (state, action: PayloadAction<string>) => {
      state.items = state.items.filter((deal) => deal.id !== action.payload);
    },
  },
});

export const {
  fetchDealsStart,
  fetchDealsSuccess,
  fetchDealsFailure,
  addDeal,
  updateDeal,
  deleteDeal,
} = dealsSlice.actions;

export default dealsSlice.reducer;