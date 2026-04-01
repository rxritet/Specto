import { Routes } from '@angular/router';
import { authGuard, guestGuard } from './auth.guard';

export const routes: Routes = [
	{
		path: 'login',
		canActivate: [guestGuard],
		loadComponent: () => import('./pages/login-page.component').then((m) => m.LoginPageComponent),
	},
	{
		path: 'dashboard',
		canActivate: [authGuard],
		loadComponent: () => import('./pages/dashboard-page.component').then((m) => m.DashboardPageComponent),
	},
	{
		path: 'accounts',
		canActivate: [authGuard],
		loadComponent: () => import('./pages/accounts-page.component').then((m) => m.AccountsPageComponent),
	},
	{
		path: 'payments',
		canActivate: [authGuard],
		loadComponent: () => import('./pages/payments-page.component').then((m) => m.PaymentsPageComponent),
	},
	{ path: '', pathMatch: 'full', redirectTo: 'dashboard' },
	{ path: '**', redirectTo: 'dashboard' },
];
