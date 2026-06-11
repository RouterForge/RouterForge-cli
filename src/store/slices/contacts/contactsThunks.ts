import { createAsyncThunk } from '@reduxjs/toolkit';
import api from '../../../services/api';
import type { Contact, CreateContactRequest, UpdateContactRequest } from '../../../types/contact';

export const fetchContacts = createAsyncThunk(
  'contacts/fetchContacts',
  async (_, { rejectWithValue }) => {
    try {
      const response = await api.get<Contact[]>('/contacts');
      return response.data;
    } catch (error) {
      return rejectWithValue(error instanceof Error ? error.message : 'Failed to fetch contacts');
    }
  }
);

export const fetchContactById = createAsyncThunk(
  'contacts/fetchContactById',
  async (contactId: string, { rejectWithValue }) => {
    try {
      const response = await api.get<Contact>(`/contacts/${contactId}`);
      return response.data;
    } catch (error) {
      return rejectWithValue(error instanceof Error ? error.message : 'Failed to fetch contact');
    }
  }
);

export const createContact = createAsyncThunk(
  'contacts/createContact',
  async (contactData: CreateContactRequest, { rejectWithValue }) => {
    try {
      const response = await api.post<Contact>('/contacts', contactData);
      return response.data;
    } catch (error) {
      return rejectWithValue(error instanceof Error ? error.message : 'Failed to create contact');
    }
  }
);

export const updateContact = createAsyncThunk(
  'contacts/updateContact',
  async ({ contactId, contactData }: { contactId: string; contactData: UpdateContactRequest }, { rejectWithValue }) => {
    try {
      const response = await api.put<Contact>(`/contacts/${contactId}`, contactData);
      return response.data;
    } catch (error) {
      return rejectWithValue(error instanceof Error ? error.message : 'Failed to update contact');
    }
  }
);

export const deleteContact = createAsyncThunk(
  'contacts/deleteContact',
  async (contactId: string, { rejectWithValue }) => {
    try {
      await api.delete(`/contacts/${contactId}`);
      return contactId;
    } catch (error) {
      return rejectWithValue(error instanceof Error ? error.message : 'Failed to delete contact');
    }
  }
);