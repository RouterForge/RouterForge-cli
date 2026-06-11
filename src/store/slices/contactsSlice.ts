import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { Contact } from '../../types';

interface ContactsState {
  items: Contact[];
  loading: boolean;
  error: string | null;
}

const initialState: ContactsState = {
  items: [],
  loading: false,
  error: null,
};

const contactsSlice = createSlice({
  name: 'contacts',
  initialState,
  reducers: {
    fetchContactsStart: (state) => {
      state.loading = true;
      state.error = null;
    },
    fetchContactsSuccess: (state, action: PayloadAction<Contact[]>) => {
      state.items = action.payload;
      state.loading = false;
    },
    fetchContactsFailure: (state, action: PayloadAction<string>) => {
      state.loading = false;
      state.error = action.payload;
    },
    addContact: (state, action: PayloadAction<Contact>) => {
      state.items.push(action.payload);
    },
    updateContact: (state, action: PayloadAction<Contact>) => {
      const index = state.items.findIndex(
        (contact) => contact.id === action.payload.id
      );
      if (index !== -1) {
        state.items[index] = action.payload;
      }
    },
    deleteContact: (state, action: PayloadAction<string>) => {
      state.items = state.items.filter(
        (contact) => contact.id !== action.payload
      );
    },
  },
});

export const {
  fetchContactsStart,
  fetchContactsSuccess,
  fetchContactsFailure,
  addContact,
  updateContact,
  deleteContact,
} = contactsSlice.actions;

export default contactsSlice.reducer;