import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

export interface Account {
  id: number;
  user_id: number;
  currency: string;
  balance: number;
  created_at: string;
  updated_at: string;
}

export interface CreateAccountRequest {
  currency: string;
}

export interface TransferRequest {
  receiver_account_id: number;
  amount: number;
  currency: string;
  description: string;
}

@Injectable({
  providedIn: 'root'
})
export class BankingService {
  private apiUrl = '/api'; // Проксируется через nginx

  constructor(private http: HttpClient) {}

  getAccounts(): Observable<Account[]> {
    return this.http.get<Account[]>(`${this.apiUrl}/accounts`);
  }

  createAccount(req: CreateAccountRequest): Observable<Account> {
    return this.http.post<Account>(`${this.apiUrl}/accounts`, req);
  }

  transfer(accountId: number, req: TransferRequest): Observable<any> {
    return this.http.post(`${this.apiUrl}/accounts/${accountId}/transfer`, req);
  }
}
