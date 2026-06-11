export interface Contact {
  id: string;
  name: string;
  email: string;
  phone: string;
  status: 'active' | 'inactive' | 'pending';
  group: 'personal' | 'business' | 'other';
  lastContact: string;
  address?: string;
  notes?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ContactFilters {
  search: string;
  status: string;
  group: string;
}

export interface SortConfig {
  key: string;
  direction: 'asc' | 'desc';
}