import { createAsyncThunk } from '@reduxjs/toolkit';
import api from '../../../services/api';
import type { Deal, CreateDealRequest, UpdateDealRequest } from '../../../types/deal';

export const fetchDeals = createAsyncThunk(
  'deals/fetchDeals',
  async (_, { rejectWithValue }) => {
    try {
      const response = await api.get<Deal[]>('/deals');
      return response.data;
    } catch (error) {
      return rejectWithValue(error instanceof Error ? error.message : 'Failed to fetch deals');
    }
  }
);

export const fetchDealById = createAsyncThunk(
  'deals/fetchDealById',
  async (dealId: string, { rejectWithValue }) => {
    try {
      const response = await api.get<Deal>(`/deals/${dealId}`);
      return response.data;
    } catch (error) {
      return rejectWithValue(error instanceof Error ? error.message : 'Failed to fetch deal');
    }
  }
);

export const createDeal = createAsyncThunk(
  'deals/createDeal',
  async (dealData: CreateDealRequest, { rejectWithValue }) => {
    try {
      const response = await api.post<Deal>('/deals', dealData);
      return response.data;
    } catch (error) {
      return rejectWithValue(error instanceof Error ? error.message : 'Failed to create deal');
    }
  }
);

export const updateDeal = createAsyncThunk(
  'deals/updateDeal',
  async ({ dealId, dealData }: { dealId: string; dealData: UpdateDealRequest }, { rejectWithValue }) => {
    try {
      const response = await api.put<Deal>(`/deals/${dealId}`, dealData);
      return response.data;
    } catch (error) {
      return rejectWithValue(error instanceof Error ? error.message : 'Failed to update deal');
    }
  }
);

export const deleteDeal = createAsyncThunk(
  'deals/deleteDeal',
  async (dealId: string, { rejectWithValue }) => {
    try {
      await api.delete(`/deals/${dealId}`);
      return dealId;
    } catch (error) {
      return rejectWithValue(error instanceof Error ? error.message : 'Failed to delete deal');
    }
  }
);