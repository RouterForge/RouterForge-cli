import React, { useState, useMemo } from 'react';
import styles from './ContactsTable.module.css';
import { Contact, ContactFilters, SortConfig } from '../../types/contacts';

interface ContactsTableProps {
  contacts: Contact[];
  onContactSelect?: (contactId: string) => void;
}

const ContactsTable: React.FC<ContactsTableProps> = ({ contacts, onContactSelect }) => {
  const [sortConfig, setSortConfig] = useState<SortConfig>({
    key: 'name',
    direction: 'asc',
  });
  const [filters, setFilters] = useState<ContactFilters>({
    search: '',
    status: 'all',
    group: 'all',
  });

  const sortedAndFilteredContacts = useMemo(() => {
    let filtered = contacts;

    // Apply filters
    if (filters.search) {
      const searchLower = filters.search.toLowerCase();
      filtered = filtered.filter(
        (contact) =>
          contact.name.toLowerCase().includes(searchLower) ||
          contact.email.toLowerCase().includes(searchLower) ||
          contact.phone.includes(filters.search)
      );
    }

    if (filters.status !== 'all') {
      filtered = filtered.filter((contact) => contact.status === filters.status);
    }

    if (filters.group !== 'all') {
      filtered = filtered.filter((contact) => contact.group === filters.group);
    }

    // Apply sorting
    const sorted = [...filtered].sort((a, b) => {
      const aValue = a[sortConfig.key as keyof Contact];
      const bValue = b[sortConfig.key as keyof Contact];

      if (aValue < bValue) {
        return sortConfig.direction === 'asc' ? -1 : 1;
      }
      if (aValue > bValue) {
        return sortConfig.direction === 'asc' ? 1 : -1;
      }
      return 0;
    });

    return sorted;
  }, [contacts, sortConfig, filters]);

  const handleSort = (key: string) => {
    setSortConfig((prev) => ({
      key,
      direction: prev.key === key && prev.direction === 'asc' ? 'desc' : 'asc',
    }));
  };

  const handleFilterChange = (filterName: keyof ContactFilters, value: string) => {
    setFilters((prev) => ({
      ...prev,
      [filterName]: value,
    }));
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return styles.statusActive;
      case 'inactive':
        return styles.statusInactive;
      case 'pending':
        return styles.statusPending;
      default:
        return '';
    }
  };

  return (
    <div className={styles.container}>
      <div className={styles.filters}>
        <input
          type="text"
          placeholder="Search contacts..."
          value={filters.search}
          onChange={(e) => handleFilterChange('search', e.target.value)}
          className={styles.searchInput}
        />

        <select
          value={filters.status}
          onChange={(e) => handleFilterChange('status', e.target.value)}
          className={styles.filterSelect}
        >
          <option value="all">All Statuses</option>
          <option value="active">Active</option>
          <option value="inactive">Inactive</option>
          <option value="pending">Pending</option>
        </select>

        <select
          value={filters.group}
          onChange={(e) => handleFilterChange('group', e.target.value)}
          className={styles.filterSelect}
        >
          <option value="all">All Groups</option>
          <option value="personal">Personal</option>
          <option value="business">Business</option>
          <option value="other">Other</option>
        </select>
      </div>

      <div className={styles.tableWrapper}>
        <table className={styles.table}>
          <thead>
            <tr>
              {[
                { key: 'name', label: 'Name' },
                { key: 'email', label: 'Email' },
                { key: 'phone', label: 'Phone' },
                { key: 'status', label: 'Status' },
                { key: 'group', label: 'Group' },
                { key: 'lastContact', label: 'Last Contact' },
              ].map((column) => (
                <th
                  key={column.key}
                  onClick={() => handleSort(column.key)}
                  className={styles.tableHeader}
                >
                  {column.label}
                  {sortConfig.key === column.key && (
                    <span className={styles.sortIndicator}>
                      {sortConfig.direction === 'asc' ? ' ↑' : ' ↓'}
                    </span>
                  )}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {sortedAndFilteredContacts.map((contact) => (
              <tr
                key={contact.id}
                className={styles.tableRow}
                onClick={() => onContactSelect?.(contact.id)}
              >
                <td className={styles.tableCell}>{contact.name}</td>
                <td className={styles.tableCell}>{contact.email}</td>
                <td className={styles.tableCell}>{contact.phone}</td>
                <td className={styles.tableCell}>
                  <span className={`${styles.statusBadge} ${getStatusColor(contact.status)}`}>
                    {contact.status}
                  </span>
                </td>
                <td className={styles.tableCell}>{contact.group}</td>
                <td className={styles.tableCell}>
                  {new Date(contact.lastContact).toLocaleDateString()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className={styles.pagination}>
        <span className={styles.resultCount}>
          Showing {sortedAndFilteredContacts.length} of {contacts.length} contacts
        </span>
      </div>
    </div>
  );
};

export default ContactsTable;