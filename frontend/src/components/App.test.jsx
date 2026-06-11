import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import App from './App';

describe('App component', () => {
  it('renders Header with title', () => {
    render(<App />);
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Big Pickle');
  });

  it('renders Footer', () => {
    render(<App />);
    const year = new Date().getFullYear();
    expect(screen.getByText(new RegExp(`© ${year}`))).toBeInTheDocument();
  });

  it('renders initial tasks', () => {
    render(<App />);
    expect(screen.getByText('Learn React')).toBeInTheDocument();
    expect(screen.getByText('Write tests')).toBeInTheDocument();
  });

  it('toggles task completion', () => {
    render(<App />);
    const checkboxes = screen.getAllByRole('checkbox');
    expect(checkboxes[0]).not.toBeChecked();
    fireEvent.click(checkboxes[0]);
    expect(checkboxes[0]).toBeChecked();
  });

  it('adds a new task when button clicked', () => {
    render(<App />);
    const addButton = screen.getByRole('button', { name: /add task/i });
    fireEvent.click(addButton);
    expect(screen.getByText('New task')).toBeInTheDocument();
  });

  it('renders correct number of tasks initially', () => {
    render(<App />);
    const items = screen.getAllByRole('listitem');
    expect(items).toHaveLength(2);
  });
});