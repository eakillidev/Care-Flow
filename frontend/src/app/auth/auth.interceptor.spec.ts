import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { authInterceptor } from './auth.interceptor';
import { AuthService } from './auth.service';

describe('authInterceptor', () => {
  let http: HttpClient;
  let controller: HttpTestingController;
  beforeEach(() => {
    TestBed.configureTestingModule({ providers: [provideHttpClient(withInterceptors([authInterceptor])), provideHttpClientTesting(), { provide: AuthService, useValue: { token: () => 'jwt-token' } }] });
    http = TestBed.inject(HttpClient); controller = TestBed.inject(HttpTestingController);
  });
  afterEach(() => controller.verify());
  it('attaches a bearer token to authenticated API requests', () => {
    http.get('/api/caregiver/visits').subscribe();
    const request = controller.expectOne('/api/caregiver/visits');
    expect(request.request.headers.get('Authorization')).toBe('Bearer jwt-token');
    request.flush([]);
  });
  it('does not attach a token to login', () => {
    http.post('/api/auth/login', {}).subscribe();
    const request = controller.expectOne('/api/auth/login');
    expect(request.request.headers.has('Authorization')).toBe(false);
    request.flush({});
  });
});
