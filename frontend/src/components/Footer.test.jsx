import React from 'react';
import { render, screen } from '@testing-library/react';
import Footer from './Footer';

describe('Footer component', () => {
  it('renders copyright year', () => {
    render(<Footer />);
    const year = new Date().getFullYear();
    expect(screen.getByText(new RegExp(`© ${year}`))).toBeInTheDocument();
  });

  it('renders footer text', () => {
    render(<Footer />);
    expect(screen.getByText(/Big Pickle/)).toBeInTheDocument();
  });
});