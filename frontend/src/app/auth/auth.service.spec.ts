import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { Router } from '@angular/router';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthService, TOKEN_KEY } from './auth.service';

describe('AuthService', () => {
  let service: AuthService;
  let http: HttpTestingController;
  const router = { navigateByUrl: vi.fn() };
  beforeEach(() => {
    localStorage.clear();
    router.navigateByUrl.mockReset();
    TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting(), { provide: Router, useValue: router }] });
    service = TestBed.inject(AuthService);
    http = TestBed.inject(HttpTestingController);
  });
  afterEach(() => http.verify());

  it('stores a valid login token and minimal user', () => {
    service.login(' Coordinator@CareFlow.Local ', 'secret').subscribe();
    const request = http.expectOne('/api/auth/login');
    expect(request.request.body).toEqual({ email: 'Coordinator@CareFlow.Local', password: 'secret' });
    request.flush({ token: 'jwt', user: { id: 'u1', first_name: 'Alex', last_name: 'Morgan', email: 'coordinator@careflow.local', role: 'coordinator' } });
    expect(localStorage.getItem(TOKEN_KEY)).toBe('jwt');
    expect(service.role()).toBe('coordinator');
  });

  it('removes authentication data on logout', () => {
    localStorage.setItem(TOKEN_KEY, 'jwt');
    service.logout();
    expect(localStorage.getItem(TOKEN_KEY)).toBeNull();
    expect(router.navigateByUrl).toHaveBeenCalledWith('/login');
  });
});
