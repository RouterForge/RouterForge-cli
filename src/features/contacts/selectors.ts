import { createSelector } from '@reduxjs/toolkit';
import { RootState } from '../../app/store';

const selectContactsState = (state: RootState) => state.contacts;

export const selectAllContacts = createSelector(
  [selectContactsState],
  (contacts) => contacts.ids.map(id => contacts.entities[id])
);

export const selectContactById = (contactId: string) => 
  createSelector(
    [selectContactsState],
    (contacts) => contacts.entities[contactId]
  );

export const selectContactsLoading = createSelector(
  [selectContactsState],
  (contacts) => contacts.loading
);

export const selectContactsError = createSelector(
  [selectContactsState],
  (contacts) => contacts.error
);

export const selectContactCount = createSelector(
  [selectAllContacts],
  (contacts) => contacts.length
);

export const selectContactsByCompany = (company: string) =>
  createSelector(
    [selectAllContacts],
    (contacts) => contacts.filter(contact => contact.company === company)
  );