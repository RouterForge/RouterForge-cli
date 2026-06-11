import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import TaskList from './TaskList';

describe('TaskList component', () => {
  const tasks = [
    { id: 1, text: 'Task 1', completed: false },
    { id: 2, text: 'Task 2', completed: true },
  ];

  it('renders tasks', () => {
    render(<TaskList tasks={tasks} />);
    expect(screen.getByText(/task 1/i)).toBeInTheDocument();
    expect(screen.getByText(/task 2/i)).toBeInTheDocument();
  });

  it('renders empty state when no tasks', () => {
    render(<TaskList tasks={[]} />);
    expect(screen.getByText(/no tasks found/i)).toBeInTheDocument();
  });

  it('passes onToggle to TaskItem', () => {
    const onToggle = jest.fn();
    render(<TaskList tasks={tasks} onToggle={onToggle} />);
    const checkbox1 = screen.getAllByRole('checkbox')[0];
    fireEvent.click(checkbox1);
    expect(onToggle).toHaveBeenCalledWith(1);
  });

  it('renders correct number of checkboxes', () => {
    render(<TaskList tasks={tasks} />);
    const checkboxes = screen.getAllByRole('checkbox');
    expect(checkboxes).toHaveLength(tasks.length);
  });

  it('renders list container', () => {
    render(<TaskList tasks={tasks} />);
    expect(screen.getByRole('list')).toBeInTheDocument();
  });
});