import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { AuthService, LoginRequest, RegisterRequest } from '../auth.service';

@Component({
  selector: 'app-login-page',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule],
  templateUrl: './login-page.component.html',
  styleUrl: './login-page.component.scss'
})
export class LoginPageComponent implements OnInit {
  private readonly fb = inject(FormBuilder);

  mode: 'login' | 'register' = 'login';
  isSubmitting = false;
  errorMessage = '';

  readonly loginForm = this.fb.nonNullable.group({
    email: this.fb.nonNullable.control('', [Validators.required, Validators.email]),
    password: this.fb.nonNullable.control('', [Validators.required, Validators.minLength(6)]),
  });

  readonly registerForm = this.fb.nonNullable.group({
    name: this.fb.nonNullable.control('', [Validators.required, Validators.minLength(2)]),
    email: this.fb.nonNullable.control('', [Validators.required, Validators.email]),
    password: this.fb.nonNullable.control('', [Validators.required, Validators.minLength(6)]),
  });

  constructor(
    private readonly authService: AuthService,
    private readonly router: Router,
  ) {}

  ngOnInit(): void {
    this.authService.ensureSession().subscribe({
      next: (user) => {
        if (user) {
          void this.router.navigate(['/dashboard']);
        }
      },
    });
  }

  switchMode(mode: 'login' | 'register'): void {
    this.mode = mode;
    this.errorMessage = '';
  }

  submitLogin(): void {
    if (this.loginForm.invalid || this.isSubmitting) {
      this.loginForm.markAllAsTouched();
      return;
    }

    const req: LoginRequest = this.loginForm.getRawValue();
    this.runAuthRequest(this.authService.login(req));
  }

  submitRegister(): void {
    if (this.registerForm.invalid || this.isSubmitting) {
      this.registerForm.markAllAsTouched();
      return;
    }

    const req: RegisterRequest = this.registerForm.getRawValue();
    this.runAuthRequest(this.authService.register(req));
  }

  private runAuthRequest(request$: ReturnType<AuthService['login']>): void {
    this.isSubmitting = true;
    this.errorMessage = '';

    request$.subscribe({
      next: () => {
        this.isSubmitting = false;
        void this.router.navigate(['/dashboard']);
      },
      error: (err: unknown) => {
        this.isSubmitting = false;
        this.errorMessage = this.resolveError(err);
      },
    });
  }

  private resolveError(err: unknown): string {
    if (err instanceof HttpErrorResponse) {
      const backendMessage = err.error?.error;
      if (typeof backendMessage === 'string' && backendMessage.length > 0) {
        return backendMessage;
      }
    }

    return 'Не удалось выполнить авторизацию. Проверьте данные.';
  }
}
