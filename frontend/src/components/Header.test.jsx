import React from 'react';
import { render, screen } from '@testing-library/react';
import Header from './Header';

describe('Header component', () => {
  const title = 'My App';
  const links = ['Home', 'About', 'Contact'];

  it('renders title', () => {
    render(<Header title={title} links={links} />);
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent(title);
  });

  it('renders navigation links', () => {
    render(<Header title={title} links={links} />);
    const nav = screen.getByRole('navigation');
    expect(nav).toBeInTheDocument();
    links.forEach(link => {
      expect(screen.getByText(link)).toBeInTheDocument();
    });
  });

  it('renders correct number of links', () => {
    render(<Header title={title} links={links} />);
    const allLinks = screen.getAllByRole('link');
    expect(allLinks).toHaveLength(links.length);
  });

  it('renders without links', () => {
    render(<Header title={title} links={[]} />);
    const nav = screen.getByRole('navigation');
    expect(nav).toBeEmptyDOMElement();
  });
});