import { createSlice, createAsyncThunk, PayloadAction } from '@reduxjs/toolkit';
import { Contact, ContactFormData } from '../types';
import { ContactsAPI } from '../api/contacts';

interface ContactsState {
  items: Contact[];
  selectedContactId: string | null;
  status: 'idle' | 'loading' | 'succeeded' | 'failed';
  error: string | null;
  optimisticUpdates: Record<string, Contact>;
}

const initialState: ContactsState = {
  items: [],
  selectedContactId: null,
  status: 'idle',
  error: null,
  optimisticUpdates: {},
};

// Async thunks for API communication
export const fetchContacts = createAsyncThunk(
  'contacts/fetchContacts',
  async () => {
    return await ContactsAPI.getAll();
  }
);

export const createContact = createAsyncThunk(
  'contacts/createContact',
  async (contactData: ContactFormData, { dispatch, getState }) => {
    // Optimistic update
    const tempId = `temp-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
    const optimisticContact: Contact = {
      ...contactData,
      id: tempId,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      companyId: contactData.companyId || '',
      tags: contactData.tags || [],
      socialProfiles: contactData.socialProfiles || {},
    };

    // Dispatch optimistic update action
    dispatch(addContactOptimistic({ tempId, contact: optimisticContact }));

    try {
      // Make actual API call
      const realContact = await ContactsAPI.create(contactData);
      // Replace optimistic with real contact
      dispatch(replaceOptimisticContact({ tempId, realContact }));
      return realContact;
    } catch (error) {
      // Rollback optimistic update
      dispatch(removeOptimisticContact(tempId));
      throw error;
    }
  }
);

export const updateContact = createAsyncThunk(
  'contacts/updateContact',
  async ({ id, data }: { id: string; data: Partial<ContactFormData> }, { dispatch, getState }) => {
    // Get current state before optimistic update
    const state = getState() as { contacts: ContactsState };
    const originalContact = state.contacts.items.find(c => c.id === id);
    
    // Optimistic update
    const optimisticContact: Contact = {
      ...originalContact!,
      ...data,
      updatedAt: new Date().toISOString(),
    };

    dispatch(updateContactOptimistic({ id, contact: optimisticContact }));

    try {
      const realContact = await ContactsAPI.update(id, data);
      dispatch(replaceOptimisticContact({ tempId: id, realContact }));
      return realContact;
    } catch (error) {
      // Rollback with original contact
      if (originalContact) {
        dispatch(replaceOptimisticContact({ tempId: id, realContact: originalContact }));
      }
      throw error;
    }
  }
);

export const deleteContact = createAsyncThunk(
  'contacts/deleteContact',
  async (id: string, { dispatch, getState }) => {
    // Get current state for rollback
    const state = getState() as { contacts: ContactsState };
    const contactToDelete = state.contacts.items.find(c => c.id === id);

    // Optimistic removal
    dispatch(removeContactOptimistic(id));

    try {
      await ContactsAPI.delete(id);
      // Contact is already removed from state
      return id;
    } catch (error) {
      // Rollback - add contact back
      if (contactToDelete) {
        dispatch(addContactOptimistic({ tempId: id, contact: contactToDelete }));
      }
      throw error;
    }
  }
);

const contactsSlice = createSlice({
  name: 'contacts',
  initialState,
  reducers: {
    addContactOptimistic: (state, action: PayloadAction<{ tempId: string; contact: Contact }>) => {
      const { tempId, contact } = action.payload;
      state.items.push(contact);
      state.optimisticUpdates[tempId] = contact;
    },
    
    updateContactOptimistic: (state, action: PayloadAction<{ id: string; contact: Contact }>) => {
      const { id, contact } = action.payload;
      const index = state.items.findIndex(c => c.id === id);
      if (index !== -1) {
        state.items[index] = contact;
        state.optimisticUpdates[id] = contact;
      }
    },
    
    removeContactOptimistic: (state, action: PayloadAction<string>) => {
      const id = action.payload;
      state.items = state.items.filter(c => c.id !== id);
      // Store for potential rollback
      const original = state.items.find(c => c.id === id);
      if (original) {
        state.optimisticUpdates[id] = original;
      }
    },
    
    replaceOptimisticContact: (state, action: PayloadAction<{ tempId: string; realContact: Contact }>) => {
      const { tempId, realContact } = action.payload;
      const index = state.items.findIndex(c => c.id === tempId);
      if (index !== -1) {
        state.items[index] = realContact;
      }
      delete state.optimisticUpdates[tempId];
    },
    
    removeOptimisticContact: (state, action: PayloadAction<string>) => {
      const tempId = action.payload;
      state.items = state.items.filter(c => c.id !== tempId);
      delete state.optimisticUpdates[tempId];
    },
    
    setSelectedContact: (state, action: PayloadAction<string | null>) => {
      state.selectedContactId = action.payload;
    },
    
    clearOptimisticUpdates: (state) => {
      state.optimisticUpdates = {};
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchContacts.pending, (state) => {
        state.status = 'loading';
      })
      .addCase(fetchContacts.fulfilled, (state, action) => {
        state.status = 'succeeded';
        state.items = action.payload;
        state.error = null;
      })
      .addCase(fetchContacts.rejected, (state, action) => {
        state.status = 'failed';
        state.error = action.error.message || 'Failed to fetch contacts';
      })
      .addCase(createContact.rejected, (state, action) => {
        state.error = action.error.message || 'Failed to create contact';
      })
      .addCase(updateContact.rejected, (state, action) => {
        state.error = action.error.message || 'Failed to update contact';
      })
      .addCase(deleteContact.rejected, (state, action) => {
        state.error = action.error.message || 'Failed to delete contact';
      });
  },
});

export const {
  addContactOptimistic,
  updateContactOptimistic,
  removeContactOptimistic,
  replaceOptimisticContact,
  removeOptimisticContact,
  setSelectedContact,
  clearOptimisticUpdates,
} = contactsSlice.actions;

export default contactsSlice.reducer;