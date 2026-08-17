import { TestBed } from '@angular/core/testing';
import { Router, UrlTree } from '@angular/router';
import { describe, expect, it } from 'vitest';
import { AuthService } from './auth.service';
import { roleGuard } from './auth.guard';

describe('roleGuard', () => {
  it('redirects an unauthenticated protected route to login', () => {
    const router = { createUrlTree: (parts: string[]) => ({ parts }) as unknown as UrlTree };
    TestBed.configureTestingModule({ providers: [{ provide: AuthService, useValue: { isAuthenticated: () => false, role: () => null } }, { provide: Router, useValue: router }] });
    const result = TestBed.runInInjectionContext(() => roleGuard('coordinator')({} as never, {} as never)) as unknown as { parts: string[] };
    expect(result.parts).toEqual(['/login']);
  });

  it('redirects an authenticated caregiver away from coordinator routes', () => {
    const router = { createUrlTree: (parts: string[]) => ({ parts }) as unknown as UrlTree };
    TestBed.configureTestingModule({ providers: [{ provide: AuthService, useValue: { isAuthenticated: () => true, role: () => 'caregiver' } }, { provide: Router, useValue: router }] });
    const result = TestBed.runInInjectionContext(() => roleGuard('coordinator')({} as never, {} as never)) as unknown as { parts: string[] };
    expect(result.parts).toEqual(['/caregiver']);
  });
});
