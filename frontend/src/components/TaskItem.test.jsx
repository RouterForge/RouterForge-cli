import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import TaskItem from './TaskItem';

describe('TaskItem component', () => {
  const task = { id: 1, text: 'Test task', completed: false };
  const onToggle = jest.fn();

  it('renders task text', () => {
    render(<TaskItem task={task} onToggle={onToggle} />);
    expect(screen.getByText(/test task/i)).toBeInTheDocument();
  });

  it('does not have line-through when not completed', () => {
    render(<TaskItem task={task} onToggle={onToggle} />);
    const span = screen.getByText(/test task/i);
    expect(span).not.toHaveStyle('text-decoration: line-through');
  });

  it('has line-through when completed', () => {
    const completedTask = { ...task, completed: true };
    render(<TaskItem task={completedTask} onToggle={onToggle} />);
    const span = screen.getByText(/test task/i);
    expect(span).toHaveStyle('text-decoration: line-through');
  });

  it('checkbox reflects completed status', () => {
    render(<TaskItem task={task} onToggle={onToggle} />);
    const checkbox = screen.getByRole('checkbox');
    expect(checkbox).not.toBeChecked();
  });

  it('calls onToggle when checkbox clicked', () => {
    render(<TaskItem task={task} onToggle={onToggle} />);
    const checkbox = screen.getByRole('checkbox');
    fireEvent.click(checkbox);
    expect(onToggle).toHaveBeenCalledWith(task.id);
  });

  it('renders aria roles', () => {
    render(<TaskItem task={task} onToggle={onToggle} />);
    expect(screen.getByRole('listitem')).toBeInTheDocument();
  });
});