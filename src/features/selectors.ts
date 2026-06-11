import { createSelector } from '@reduxjs/toolkit';
import { RootState } from '../app/store';
import { selectAllContacts } from './contacts/selectors';
import { selectAllDeals, selectTotalDealValue } from './deals/selectors';
import { selectAllTasks, selectPendingTasks, selectOverdueTasks } from './tasks/selectors';

export const selectDashboardStats = createSelector(
  [selectAllContacts, selectAllDeals, selectAllTasks],
  (contacts, deals, tasks) => ({
    contactCount: contacts.length,
    dealCount: deals.length,
    taskCount: tasks.length,
    totalDealValue: deals.reduce((total, deal) => total + deal.value, 0),
    completedTasks: tasks.filter(task => task.completed).length,
    pendingTasks: tasks.filter(task => !task.completed).length
  })
);

export const selectCrmOverview = createSelector(
  [selectAllContacts, selectAllDeals, selectPendingTasks, selectOverdueTasks],
  (contacts, deals, pendingTasks, overdueTasks) => ({
    totalContacts: contacts.length,
    activeDeals: deals.filter(deal => deal.stage !== 'closed').length,
    pendingTaskCount: pendingTasks.length,
    overdueTaskCount: overdueTasks.length,
    upcomingDeals: deals.filter(deal => 
      deal.closeDate > new Date().toISOString() && deal.stage !== 'closed'
    ).length
  })
);