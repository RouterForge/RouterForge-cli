import { createSelector } from '@reduxjs/toolkit';
import { RootState } from '../../app/store';

const selectDealsState = (state: RootState) => state.deals;

export const selectAllDeals = createSelector(
  [selectDealsState],
  (deals) => deals.ids.map(id => deals.entities[id])
);

export const selectDealById = (dealId: string) =>
  createSelector(
    [selectDealsState],
    (deals) => deals.entities[dealId]
  );

export const selectDealsLoading = createSelector(
  [selectDealsState],
  (deals) => deals.loading
);

export const selectDealsError = createSelector(
  [selectDealsState],
  (deals) => deals.error
);

export const selectTotalDealValue = createSelector(
  [selectAllDeals],
  (deals) => deals.reduce((total, deal) => total + deal.value, 0)
);

export const selectDealsByStage = (stage: string) =>
  createSelector(
    [selectAllDeals],
    (deals) => deals.filter(deal => deal.stage === stage)
  );

export const selectDealsByContact = (contactId: string) =>
  createSelector(
    [selectAllDeals],
    (deals) => deals.filter(deal => deal.contactId === contactId)
  );

export const selectUpcomingDeals = createSelector(
  [selectAllDeals],
  (deals) => deals.filter(deal => deal.closeDate > new Date().toISOString())
);