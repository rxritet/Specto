import { Injectable } from '@angular/core';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { BehaviorSubject, Observable, of, throwError } from 'rxjs';
import { catchError, tap } from 'rxjs/operators';

export interface User {
  id: number;
  email: string;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  email: string;
  name: string;
  password: string;
}

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private readonly apiUrl = '/api/auth';
  private readonly currentUserSubject = new BehaviorSubject<User | null>(null);

  readonly currentUser$ = this.currentUserSubject.asObservable();

  constructor(private readonly http: HttpClient) {}

  ensureSession(): Observable<User | null> {
    const current = this.currentUserSubject.value;
    if (current) {
      return of(current);
    }

    return this.me().pipe(
      catchError((err: HttpErrorResponse) => {
        if (err.status === 401) {
          this.currentUserSubject.next(null);
          return of(null);
        }
        return throwError(() => err);
      })
    );
  }

  me(): Observable<User> {
    return this.http.get<User>(`${this.apiUrl}/me`).pipe(
      tap((user) => this.currentUserSubject.next(user))
    );
  }

  login(req: LoginRequest): Observable<User> {
    return this.http.post<User>(`${this.apiUrl}/login`, req).pipe(
      tap((user) => this.currentUserSubject.next(user))
    );
  }

  register(req: RegisterRequest): Observable<User> {
    return this.http.post<User>(`${this.apiUrl}/register`, req).pipe(
      tap((user) => this.currentUserSubject.next(user))
    );
  }

  logout(): Observable<{ status: string }> {
    return this.http.post<{ status: string }>(`${this.apiUrl}/logout`, {}).pipe(
      tap(() => this.currentUserSubject.next(null))
    );
  }
}
