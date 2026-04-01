import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { Router, RouterLink } from '@angular/router';
import { Account, BankingService } from '../banking.service';

@Component({
  selector: 'app-dashboard-page',
  standalone: true,
  imports: [CommonModule, RouterLink],
  templateUrl: './dashboard-page.component.html',
  styleUrl: './dashboard-page.component.scss'
})
export class DashboardPageComponent implements OnInit {
  accounts: Account[] = [];
  isLoading = false;
  errorMessage = '';

  constructor(
    private readonly bankingService: BankingService,
    private readonly router: Router,
  ) {}

  ngOnInit(): void {
    this.loadAccounts();
  }

  loadAccounts(): void {
    this.isLoading = true;
    this.errorMessage = '';

    this.bankingService.getAccounts().subscribe({
      next: (accounts) => {
        this.accounts = accounts;
        this.isLoading = false;
      },
      error: (err: unknown) => {
        this.handleApiError(err);
        this.isLoading = false;
      },
    });
  }

  goToPayments(): void {
    void this.router.navigate(['/payments']);
  }

  goToAccounts(): void {
    void this.router.navigate(['/accounts']);
  }

  formatCurrency(amount: number, currency: string): string {
    return (amount / 100).toLocaleString('ru-RU', {
      style: 'currency',
      currency,
    });
  }

  get totalBalance(): number {
    return this.accounts.reduce((sum, acc) => sum + acc.balance, 0);
  }

  private handleApiError(err: unknown): void {
    if (err instanceof HttpErrorResponse) {
      if (err.status === 401) {
        this.errorMessage = 'Сессия истекла. Выполните вход снова.';
        void this.router.navigate(['/login']);
        return;
      }

      const backendMessage = err.error?.error;
      if (typeof backendMessage === 'string' && backendMessage.length > 0) {
        this.errorMessage = backendMessage;
        return;
      }
    }

    this.errorMessage = 'Не удалось загрузить данные кабинета.';
  }
}
