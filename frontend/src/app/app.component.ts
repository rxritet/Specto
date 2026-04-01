import { Component, OnInit } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { CommonModule } from '@angular/common';
import { BankingService, Account } from './banking.service';
import { HttpClientModule } from '@angular/common/http';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [RouterOutlet, CommonModule, HttpClientModule],
  providers: [BankingService],
  templateUrl: './app.component.html',
  styleUrl: './app.component.scss'
})
export class AppComponent implements OnInit {
  title = 'Specto Bank';
  accounts: Account[] = [];

  constructor(private bankingService: BankingService) {}

  ngOnInit() {
    this.loadAccounts();
  }

  loadAccounts() {
    this.bankingService.getAccounts().subscribe({
      next: (data) => {
        this.accounts = data;
      },
      error: (err) => {
        console.error('Failed to load accounts. Migration might be pending.', err);
      }
    });
  }

  formatCurrency(amount: number, currency: string) {
    // Баланс хранится в центах/тиынах
    return (amount / 100).toLocaleString('ru-RU', { 
      style: 'currency', 
      currency: currency 
    });
  }
}
