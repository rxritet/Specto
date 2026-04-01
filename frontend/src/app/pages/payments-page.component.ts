import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { Account, BankingService, TransferRequest } from '../banking.service';

@Component({
  selector: 'app-payments-page',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule],
  templateUrl: './payments-page.component.html',
  styleUrl: './payments-page.component.scss'
})
export class PaymentsPageComponent implements OnInit {
  private readonly fb = inject(FormBuilder);

  accounts: Account[] = [];
  isLoading = false;
  isSubmitting = false;
  errorMessage = '';
  successMessage = '';

  readonly transferForm = this.fb.nonNullable.group({
    senderAccountId: this.fb.nonNullable.control(0, [Validators.required, Validators.min(1)]),
    receiverAccountId: this.fb.nonNullable.control(0, [Validators.required, Validators.min(1)]),
    amount: this.fb.nonNullable.control(0, [Validators.required, Validators.min(0.01)]),
    currency: this.fb.nonNullable.control('KZT', [Validators.required, Validators.minLength(3), Validators.maxLength(3)]),
    description: this.fb.nonNullable.control('', [Validators.maxLength(120)]),
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
        if (accounts.length > 0 && this.transferForm.controls.senderAccountId.value === 0) {
          this.transferForm.controls.senderAccountId.setValue(accounts[0].id);
          this.transferForm.controls.currency.setValue(accounts[0].currency);
        }
        this.isLoading = false;
      },
      error: (err: unknown) => {
        this.handleApiError(err, 'Не удалось загрузить счета для перевода.');
        this.isLoading = false;
      },
    });
  }

  onSenderChanged(): void {
    const senderID = this.transferForm.controls.senderAccountId.value;
    const sender = this.accounts.find((acc) => acc.id === Number(senderID));
    if (sender) {
      this.transferForm.controls.currency.setValue(sender.currency);
    }
  }

  submitTransfer(): void {
    if (this.transferForm.invalid || this.isSubmitting) {
      this.transferForm.markAllAsTouched();
      return;
    }

    const amountCents = Math.round(this.transferForm.controls.amount.value * 100);
    if (amountCents <= 0) {
      this.errorMessage = 'Сумма должна быть больше нуля.';
      return;
    }

    this.isSubmitting = true;
    this.errorMessage = '';
    this.successMessage = '';

    const senderAccountId = Number(this.transferForm.controls.senderAccountId.value);
    const req: TransferRequest = {
      receiver_account_id: Number(this.transferForm.controls.receiverAccountId.value),
      amount: amountCents,
      currency: this.transferForm.controls.currency.value.trim().toUpperCase(),
      description: this.transferForm.controls.description.value.trim(),
    };

    this.bankingService.transfer(senderAccountId, req).subscribe({
      next: () => {
        this.successMessage = 'Перевод выполнен успешно.';
        this.isSubmitting = false;
        this.transferForm.patchValue({
          receiverAccountId: 0,
          amount: 0,
          description: '',
        });
        this.loadAccounts();
      },
      error: (err: unknown) => {
        this.handleApiError(err, 'Не удалось выполнить перевод.');
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
