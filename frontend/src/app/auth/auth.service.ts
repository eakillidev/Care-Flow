import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Router } from '@angular/router';
import { BehaviorSubject, Observable, tap } from 'rxjs';
import { AuthUser, LoginResponse, UserRole } from './auth.models';

export const TOKEN_KEY = 'careflow_token';
const USER_KEY = 'careflow_user';

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly userSubject = new BehaviorSubject<AuthUser | null>(this.readStoredUser());
  readonly user$ = this.userSubject.asObservable();

  constructor(private readonly http: HttpClient, private readonly router: Router) {}

  login(email: string, password: string): Observable<LoginResponse> {
    return this.http.post<LoginResponse>('/api/auth/login', { email: email.trim(), password }).pipe(
      tap(response => {
        localStorage.setItem(TOKEN_KEY, response.token);
        localStorage.setItem(USER_KEY, JSON.stringify(response.user));
        this.userSubject.next(response.user);
      })
    );
  }

  logout(): void {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(USER_KEY);
    this.userSubject.next(null);
    void this.router.navigateByUrl('/login');
  }

  token(): string | null { return localStorage.getItem(TOKEN_KEY); }
  currentUser(): AuthUser | null { return this.userSubject.value; }
  role(): UserRole | null { return this.currentUser()?.role ?? null; }
  isAuthenticated(): boolean { return Boolean(this.token() && this.currentUser()); }

  private readStoredUser(): AuthUser | null {
    const value = localStorage.getItem(USER_KEY);
    if (!value) return null;
    try {
      const user = JSON.parse(value) as AuthUser;
      return user?.id && (user.role === 'coordinator' || user.role === 'caregiver') ? user : null;
    } catch {
      localStorage.removeItem(USER_KEY);
      return null;
    }
  }
}
