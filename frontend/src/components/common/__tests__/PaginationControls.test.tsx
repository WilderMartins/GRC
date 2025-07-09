import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import PaginationControls from '../PaginationControls';

describe('PaginationControls', () => {
  const mockOnPageChange = jest.fn();

  const defaultProps = {
    currentPage: 1,
    totalPages: 5,
    totalItems: 45,
    pageSize: 10,
    onPageChange: mockOnPageChange,
    isLoading: false,
  };

  beforeEach(() => {
    mockOnPageChange.mockClear();
  });

  test('não renderiza nada se totalPages for 0', () => {
    render(<PaginationControls {...defaultProps} totalPages={0} />);
    expect(screen.queryByRole('navigation')).not.toBeInTheDocument();
  });

  // Ajuste: O componente renderiza para 1 página total, mas os botões estarão desabilitados.
  // A mensagem "Mostrando X a Y de Z" ainda aparecerá.
  // Se a intenção é não renderizar para 1 página, a lógica no componente precisaria ser `if (totalPages <= 1) return null;`
  // A lógica atual é `if (totalPages <= 0) return null;`
  test('renderiza para 1 página total (botões desabilitados)', () => {
    render(<PaginationControls {...defaultProps} totalPages={1} totalItems={7} pageSize={10} currentPage={1}/>);
    expect(screen.getByRole('navigation')).toBeInTheDocument();
    expect(screen.getByText('Anterior')).toBeDisabled();
    expect(screen.getByText('Próxima')).toBeDisabled();
    expect(screen.getByText(/Mostrando 1 a 7 de 7 resultados/)).toBeInTheDocument();
  });


  test('renderiza corretamente com múltiplas páginas', () => {
    render(<PaginationControls {...defaultProps} />);
    expect(screen.getByRole('navigation')).toBeInTheDocument();
    expect(screen.getByText('Anterior')).toBeInTheDocument();
    expect(screen.getByText('Próxima')).toBeInTheDocument();
  });

  test('exibe a contagem correta de itens na primeira página', () => {
    render(<PaginationControls {...defaultProps} currentPage={1} />);
    expect(screen.getByText(/Mostrando/)).toHaveTextContent('Mostrando 1 a 10 de 45 resultados');
  });

  test('exibe a contagem correta de itens em uma página intermediária', () => {
    render(<PaginationControls {...defaultProps} currentPage={3} />);
    expect(screen.getByText(/Mostrando/)).toHaveTextContent('Mostrando 21 a 30 de 45 resultados');
  });

  test('exibe a contagem correta de itens na última página', () => {
    render(<PaginationControls {...defaultProps} currentPage={5} />);
    expect(screen.getByText(/Mostrando/)).toHaveTextContent('Mostrando 41 a 45 de 45 resultados');
  });

  test('exibe "Nenhum resultado encontrado" se totalItems for 0 e totalPages for 1 (ou mais)', () => {
    // O componente renderiza se totalPages >= 1.
    render(<PaginationControls {...defaultProps} totalItems={0} totalPages={1} currentPage={1} />);
    expect(screen.getByText("Nenhum resultado encontrado.")).toBeInTheDocument();
    // Botões devem estar desabilitados pois totalPages é 1
    expect(screen.getByText('Anterior')).toBeDisabled();
    expect(screen.getByText('Próxima')).toBeDisabled();
  });


  test('botão "Anterior" está desabilitado na primeira página', () => {
    render(<PaginationControls {...defaultProps} currentPage={1} />);
    expect(screen.getByText('Anterior')).toBeDisabled();
  });

  test('botão "Próxima" está desabilitado na última página', () => {
    render(<PaginationControls {...defaultProps} currentPage={5} totalPages={5} />);
    expect(screen.getByText('Próxima')).toBeDisabled();
  });

  test('ambos os botões estão habilitados em uma página intermediária', () => {
    render(<PaginationControls {...defaultProps} currentPage={3} totalPages={5} />);
    expect(screen.getByText('Anterior')).not.toBeDisabled();
    expect(screen.getByText('Próxima')).not.toBeDisabled();
  });

  test('botões estão desabilitados se isLoading for true', () => {
    render(<PaginationControls {...defaultProps} isLoading={true} currentPage={3} totalPages={5} />);
    expect(screen.getByText('Anterior')).toBeDisabled();
    expect(screen.getByText('Próxima')).toBeDisabled();
  });

  test('chama onPageChange com a página correta ao clicar em "Próxima"', () => {
    render(<PaginationControls {...defaultProps} currentPage={2} totalPages={5} />);
    fireEvent.click(screen.getByText('Próxima'));
    expect(mockOnPageChange).toHaveBeenCalledWith(3);
  });

  test('chama onPageChange com a página correta ao clicar em "Anterior"', () => {
    render(<PaginationControls {...defaultProps} currentPage={3} totalPages={5} />);
    fireEvent.click(screen.getByText('Anterior'));
    expect(mockOnPageChange).toHaveBeenCalledWith(2);
  });

  test('não chama onPageChange se "Próxima" for clicado na última página', () => {
    render(<PaginationControls {...defaultProps} currentPage={5} totalPages={5} />);
    fireEvent.click(screen.getByText('Próxima'));
    expect(mockOnPageChange).not.toHaveBeenCalled();
  });

  test('não chama onPageChange se "Anterior" for clicado na primeira página', () => {
    render(<PaginationControls {...defaultProps} currentPage={1} totalPages={5} />);
    fireEvent.click(screen.getByText('Anterior'));
    expect(mockOnPageChange).not.toHaveBeenCalled();
  });
});
