import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { Account, BankingService } from '../banking.service';

@Component({
  selector: 'app-accounts-page',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule],
  templateUrl: './accounts-page.component.html',
  styleUrl: './accounts-page.component.scss'
})
export class AccountsPageComponent implements OnInit {
  private readonly fb = inject(FormBuilder);

  accounts: Account[] = [];
  isLoading = false;
  isSubmitting = false;
  successMessage = '';
  errorMessage = '';

  readonly createForm = this.fb.nonNullable.group({
    currency: this.fb.nonNullable.control('KZT', [Validators.required, Validators.minLength(3), Validators.maxLength(3)]),
  });

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
        this.handleApiError(err, 'Не удалось загрузить счета.');
        this.isLoading = false;
      },
    });
  }

  createAccount(): void {
    if (this.createForm.invalid || this.isSubmitting) {
      this.createForm.markAllAsTouched();
      return;
    }

    this.isSubmitting = true;
    this.successMessage = '';
    this.errorMessage = '';

    const currency = this.createForm.controls.currency.value.trim().toUpperCase();
    this.bankingService.createAccount({ currency }).subscribe({
      next: () => {
        this.successMessage = `Счет ${currency} успешно открыт.`;
        this.isSubmitting = false;
        this.loadAccounts();
      },
      error: (err: unknown) => {
        this.handleApiError(err, 'Не удалось открыть счет.');
        this.isSubmitting = false;
      },
    });
  }

  formatCurrency(amount: number, currency: string): string {
    return (amount / 100).toLocaleString('ru-RU', {
      style: 'currency',
      currency,
    });
  }

  private handleApiError(err: unknown, fallbackMessage: string): void {
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

    this.errorMessage = fallbackMessage;
  }
}
